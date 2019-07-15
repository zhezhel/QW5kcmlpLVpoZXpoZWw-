package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"time"
)

type Result struct {
	Response  []byte    `json:"response"`
	Duration  duration  `json:"duration"`
	CreatedAt time.Time `json:"created_at"`
}

type Task struct {
	ID       *int64    `json:"id,omitempty"`
	URL      *string   `json:"url,omitempty"`
	Interval *duration `json:"interval,omitempty"`
}

type Pool struct {
	Ctx context.Context
	sync.Mutex

	workersID     map[int64]*Worker
	workersString map[string]*Worker
	counter       int64
}

type Worker struct {
	ID   int64
	URL  string
	Job  chan Task
	Task Task
	sync.Mutex
	Results []Result
	Context context.Context
	Cancel  context.CancelFunc
}

func (w *Worker) Fetching() {
	Client := &http.Client{
		Timeout: time.Second * 5,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 5 * time.Second,
		},
	}
	req, _ := http.NewRequest("GET", *w.Task.URL, nil)
	req = req.WithContext(w.Context)

	for {
		select {
		case <-w.Context.Done():
			return
		case t := <-w.Job:
			w.Task = t
		case <-time.After(time.Duration(*w.Task.Interval)):
			timer := time.Now()
			data := []byte{}

			res, err := Client.Do(req)
			if err != nil {
				data = nil
			} else {
				data, _ = ioutil.ReadAll(res.Body)
			}
			d := time.Now().Sub(timer)
			w.Lock()
			w.Results = append(w.Results, Result{Response: data, Duration: duration(d), CreatedAt: time.Now()})
			w.Unlock()
		}
	}
}

func (p *Pool) Submit(task Task) int64 {
	worker, ok := p.workersString[*task.URL]
	p.Lock()
	defer p.Unlock()

	if !ok {
		p.counter++
		ctx, cancel := context.WithCancel(p.Ctx)
		id := p.counter
		task.ID = &id
		worker = &Worker{
			ID:      id,
			URL:     *task.URL,
			Task:    task,
			Job:     make(chan Task, 1),
			Context: ctx,
			Cancel:  cancel,
			Results: []Result{},
		}

		p.workersID[worker.ID] = worker
		p.workersString[*task.URL] = worker

		go worker.Fetching()
	}
	task.ID = &worker.ID
	worker.Job <- task

	return worker.ID
}

func (p *Pool) Delete(id int64) {
	p.Lock()
	defer p.Unlock()
	worker, ok := p.workersID[id]
	if !ok {
		return
	}
	worker.Cancel()

	delete(p.workersID, worker.ID)
	delete(p.workersString, worker.URL)
}

func (p Pool) Tasks() []Task {
	p.Lock()
	defer p.Unlock()
	values := make([]Task, len(p.workersID), len(p.workersID))
	idx := 0
	for _, v := range p.workersID {
		values[idx] = v.Task
		idx++
	}
	return values
}

func (p Pool) Results(task Task) []Result {
	p.Lock()
	defer p.Unlock()
	fmt.Println(*task.ID)
	worker, ok := p.workersID[*task.ID]
	if !ok {
		return []Result{}
	}
	worker.Lock()
	defer worker.Unlock()
	return worker.Results
}

func (p Pool) TaskById(id int64) (Task, error) {
	p.Lock()
	defer p.Unlock()
	worker, ok := p.workersID[id]
	if !ok {
		return Task{}, errors.New("not found")
	}
	worker.Lock()
	defer worker.Unlock()
	return worker.Task, nil
}

func NewPool(ctx context.Context) *Pool {
	p := Pool{Ctx: ctx}
	p.counter = int64(0)
	p.workersID = make(map[int64]*Worker)
	p.workersString = make(map[string]*Worker)
	return &p
}
