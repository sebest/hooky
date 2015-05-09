package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/codegangsta/cli"
	"github.com/sebest/hooky/restapi"
	"github.com/sebest/hooky/scheduler"
	"github.com/sebest/hooky/store"
	"github.com/stretchr/graceful"
)

func main() {
	app := cli.NewApp()
	app.Name = "hooky"
	app.Usage = "the webhooks scheduler"
	app.Version = "0.1"
	app.Author = "Sébastien Estienne"
	app.Email = "sebastien.estienne@gmail.com"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "bind",
			Value:  ":8000",
			Usage:  "address to bind on",
			EnvVar: "HOOKY_BIND",
		},
		cli.StringFlag{
			Name:   "mongo-uri",
			Value:  "localhost/hooky",
			Usage:  "MongoDB URI to connect to",
			EnvVar: "HOOKY_MONGO_URI",
		},
		cli.StringFlag{
			Name:   "admin-password",
			Value:  "admin",
			Usage:  "admin password",
			EnvVar: "HOOKY_ADMIN_PASSWORD",
		},
		cli.IntFlag{
			Name:   "max-mongo-query",
			Value:  1,
			Usage:  "maximum number of parallel queries on MongoDB",
			EnvVar: "HOOKY_MAX_MONGO_QUERY",
		},
		cli.IntFlag{
			Name:   "max-http-request",
			Value:  20,
			Usage:  "maximum number of parallel HTTP requests",
			EnvVar: "HOOKY_MAX_HTTP_REQUEST",
		},
		cli.IntFlag{
			Name:   "touch-interval",
			Value:  5,
			Usage:  "frequency to update the tasks reservation duration in seconds",
			EnvVar: "HOOKY_TOUCH_INTERVAL",
		},
	}
	app.Action = func(c *cli.Context) {
		s, err := store.New(c.String("mongo-uri"))
		if err != nil {
			log.Fatal(err)
		}
		sched := scheduler.New(s, c.Int("max-mongo-query"), c.Int("max-http-request"), c.Int("touch-interval"))
		sched.Start()
		ra, err := restapi.New(s, c.String("admin-password"))
		if err != nil {
			log.Fatal(err)
		}
		server := &graceful.Server{
			Timeout: 10 * time.Second,
			Server: &http.Server{
				Addr:    c.String("bind"),
				Handler: ra.MakeHandler(),
			},
		}
		server.ListenAndServe()
		fmt.Println("exiting")
		sched.Stop()
		fmt.Println("exited")
	}
	app.Run(os.Args)
}
