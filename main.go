package main

import (
	"context"
	"net/http"
	"os"

	"github.com/urfave/cli"
)

func main() {

	app := cli.NewApp()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "d,database",
			EnvVar: "DATABASE_URL",
			Value:  "sqlite3://:memory:",
		},
		cli.IntFlag{
			Name:   "p,port",
			EnvVar: "PORT",
			Value:  8080,
		},
	}
	app.Action = func(c *cli.Context) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		pool := NewPool(ctx)
		r := getRouter(pool)
		if err := http.ListenAndServe(":8080", r); err != nil {
			return cli.NewExitError(err, 1)
		}

		return nil
	}

	app.Run(os.Args)
}
