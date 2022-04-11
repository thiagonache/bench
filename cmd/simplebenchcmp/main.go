package main

import (
	"fmt"
	"os"

	"github.com/thiagonache/bench"
)

var usage = "Please do not press this button again"

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}
	delta, err := bench.ReadStatsFiles(os.Args[1], os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(delta)
}
