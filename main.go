package main

import (
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

func main() {
	s, err := store.New("localhost/hooky")
	if err != nil {
		log.Fatal(err)
	}
	m := models.NewManager(s)
	sched := scheduler.New(m, 1, 20)
	sched.Start()
	ra, err := restapi.New(m)
	if err != nil {
		log.Fatal(err)
	}

	server := &graceful.Server{
		Timeout: 10 * time.Second,
		Server: &http.Server{
			Addr:    ":8002",
			Handler: ra.MakeHandler(),
		},
	}

	server.ListenAndServe()
	fmt.Println("exiting")
	sched.Stop()
	fmt.Println("exited")
}
