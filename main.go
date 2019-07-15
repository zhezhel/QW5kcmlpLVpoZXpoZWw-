package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
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

	app.Action = func(c *cli.Context) error {
		var stop = make(chan os.Signal)

		signal.Notify(stop, syscall.SIGTERM)
		signal.Notify(stop, syscall.SIGINT)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		r := getRouter(NewPool(ctx))
		srv := http.Server{Addr: ":8080", Handler: chi.ServerBaseContext(ctx, r)}

		go func() {
			<-stop
			srv.Shutdown(ctx)

			cancel()
		}()

		if err := srv.ListenAndServe(); err != nil {
			return cli.NewExitError(err, 1)
		}

		return nil
	}

	app.Run(os.Args)
}
