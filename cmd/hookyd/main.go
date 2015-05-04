package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/sebest/hooky/restapi"
	"github.com/sebest/hooky/scheduler"
	"github.com/sebest/hooky/store"
	"github.com/stretchr/graceful"
)

var (
	bind             = flag.String("bind", ":8000", "address to bind on")
	mongoURI         = flag.String("mongo-uri", "localhost/hooky", "MongoDB URI to connect to.")
	maxMongoQuerier  = flag.Int("max-mongo-query", 1, "maximum number of parallel queries on MongoDB")
	maxHTTPRequester = flag.Int("max-http-request", 20, "maximum number of parallel HTTP requests")
	touchInterval    = flag.Int("touch-interval", 5, "frequency to update the tasks reservation duration in seconds")
)

func main() {
	flag.Parse()

	s, err := store.New(*mongoURI)
	if err != nil {
		log.Fatal(err)
	}
	sched := scheduler.New(s, *maxMongoQuerier, *maxHTTPRequester, *touchInterval)
	sched.Start()
	ra, err := restapi.New(s)
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
