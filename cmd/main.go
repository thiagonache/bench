package main

import (
	"fmt"
	"time"

	"github.com/thiagonache/simplebench"
)

func main() {
	site := "https://bitfieldconsulting.com"
	reqs := 10

	loadgen, err := simplebench.NewLoadGen(site, simplebench.WithRequests(reqs))
	if err != nil {
		fmt.Errorf("Error creating NewLoadGen object: %v", err)
	}
	loadgen.Run()
	fmt.Printf("URL %q => Bench is done\n", site)
	fmt.Printf("Time: %v Requests: %d Success: %d\n", time.Since(loadgen.StartAt), loadgen.Stats.Requests, loadgen.Stats.Success)
}
