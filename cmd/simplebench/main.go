package main

import (
	"fmt"
	"os"

	"github.com/thiagonache/bench"
)

func main() {
	if err := bench.RunCLI(os.Args[1:]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
