package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/thiagonache/bench"
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
	tester, err := bench.NewTester(url, bench.WithRequests(reqs))
	if err != nil {
		panic(fmt.Errorf("error creating NewTester object: %v", err))
	}
	tester.Run()
}
