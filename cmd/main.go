package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/thiagonache/simplebench"
)

func main() {
	var reqs int
	flag.IntVar(&reqs, "r", 1, "number of requests to be performed in the benchmark")
	flag.Parse()
	if len(os.Args) <= 1 {
		fmt.Println("Please, inform an url to benchmark")
		os.Exit(1)
	}
	url := os.Args[1]
	loadgen, err := simplebench.NewLoadGen(url, simplebench.WithRequests(reqs))
	if err != nil {
		panic(fmt.Errorf("error creating NewLoadGen object: %v", err))
	}
	loadgen.Run()
}
