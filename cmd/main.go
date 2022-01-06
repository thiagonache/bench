package main

import (
	"flag"
	"fmt"

	"github.com/thiagonache/simplebench"
)

func main() {
	var url string
	var reqs int
	flag.StringVar(&url, "u", "", "url to run benchmark")
	flag.IntVar(&reqs, "r", 1, "number of requests to be performed in the benchmark")
	flag.Parse()
	if url == "" {
		flag.Usage()
		return
	}
	loadgen, err := simplebench.NewLoadGen(url, simplebench.WithRequests(reqs))
	if err != nil {
		panic(fmt.Errorf("error creating NewLoadGen object: %v", err))
	}
	loadgen.Run()
}
