package main

import (
	"fmt"
	"os"

	"github.com/thiagonache/bench"
)

func main() {
	tester, err := bench.NewTester(
		bench.FromArgs(os.Args[1:]),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	tester.Run()
}
