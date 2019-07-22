package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
)

type duration time.Duration

func (d duration) MarshalJSON() ([]byte, error) {
	seconds := float64(time.Duration(d).Nanoseconds()) / float64(time.Second)
	return json.Marshal(math.Round(seconds*1000.0) / 1000.0)
}

func (d *duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		*d = duration(time.Duration(value) * time.Second)
		return nil
	default:
		return errors.New("invalid duration")
	}
}

func getRouter(p *Pool) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/api/fetcher", Tasks(p))

	r.Post("/api/fetcher", Submit(p))

	r.Get("/api/fetcher/{id:\\d+}/history", History(p))

	r.Delete("/api/fetcher/{id:\\d+}", Delete(p))

	return r
}

func Tasks(p *Pool) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		render.Status(r, http.StatusOK)
		render.JSON(w, r, p.Tasks())
	}
}

func Submit(p *Pool) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		data, err := ioutil.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
		if err != nil {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}

		task := Task{}
		err = json.Unmarshal(data, &task)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if task.URL == nil || task.Interval == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		task.ID = nil
		id := p.Submit(task)
		render.Status(r, http.StatusOK)
		render.JSON(w, r, map[string]int64{"id": id})
	}
}

func Delete(p *Pool) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		p.Delete(id)
		w.WriteHeader(http.StatusNoContent)
	}
}

func History(p *Pool) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		task, err := p.TaskById(id)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, p.Results(task))

	}
}
