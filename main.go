package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/go-chi/chi"
	"github.com/urfave/cli"
)

func main() {

	app := cli.NewApp()
	app.Author = "Andrii Zhezhel"
	app.Email = "andrii.zhezhel@gmail.com"
	app.Name = "fetcher"
	app.Version = "0.1.0"

	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name:   "p,port",
			EnvVar: "PORT",
			Value:  8080,
		},
	}

	app.Action = func(c *cli.Context) error {
		var stop = make(chan os.Signal)

		signal.Notify(stop, syscall.SIGTERM)
		signal.Notify(stop, syscall.SIGINT)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		r := getRouter(NewPool(ctx))

		port := strconv.Itoa(c.Int("port"))
		log.Println("Listening...", port)

		srv := http.Server{Addr: ":" + port, Handler: chi.ServerBaseContext(ctx, r)}

		go func() {
			<-stop
			err := srv.Shutdown(ctx)
			if err != nil {
				log.Println(err)
			}
			cancel()

		}()

		if err := srv.ListenAndServe(); err != nil {
			return cli.NewExitError(err, 1)
		}
		return nil
	}

	log.Fatal(app.Run(os.Args))
}
