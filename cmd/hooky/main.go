package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/sebest/hooky/client"
	"gopkg.in/yaml.v2"
)

var (
	baseURL     = flag.String("base-url", "http://127.0.0.1:8000", "HookyD base URL")
	crontabFile = flag.String("crontab-file", "", "load a crontab file")
)

func main() {
	flag.Parse()

	if *crontabFile != "" {
		crontab, err := hooky.NewCrontabFromFile(*crontabFile)
		if err != nil {
			log.Fatal(err)
		}
		hooky.SyncCrontab(*baseURL, crontab)
		payload, err := yaml.Marshal(crontab)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(payload))
	}
}
