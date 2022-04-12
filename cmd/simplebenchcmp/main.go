package main

import (
	"fmt"
	"os"

	"github.com/thiagonache/bench"
)

var usage = fmt.Sprintf("Usage: %s statsfile1.txt statsfile2.txt", os.Args[0])

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
