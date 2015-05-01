package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/sebest/hooky/models"
	"github.com/sebest/hooky/restapi"
	"github.com/sebest/hooky/scheduler"
	"github.com/sebest/hooky/store"
	"github.com/stretchr/graceful"
)

var (
	bind             = flag.String("bind", ":8000", "address to bind on")
	mongoURI         = flag.String("mongo-uri", "localhost/hooky", "MongoDB URI to connect to.")
	maxMongoQuerier  = flag.Int("max-mongo-query", 1, "maximum number of parallel queries on MongoDB")
	maxHttpRequester = flag.Int("max-http-request", 20, "maximum number of parallel HTTP requests")
)

func main() {
	flag.Parse()

	s, err := store.New(*mongoURI)
	if err != nil {
		log.Fatal(err)
	}
	tm := models.NewTasksManager(s)
	sched := scheduler.New(tm, *maxMongoQuerier, *maxHttpRequester)
	sched.Start()
	ra, err := restapi.New(tm)
	if err != nil {
		log.Fatal(err)
	}

	server := &graceful.Server{
		Timeout: 10 * time.Second,
		Server: &http.Server{
			Addr:    *bind,
			Handler: ra.MakeHandler(),
		},
	}

	server.ListenAndServe()
	fmt.Println("exiting")
	sched.Stop()
	fmt.Println("exited")
}
