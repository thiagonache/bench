package main

import (
	"fmt"
	"net/url"

	"github.com/thiagonache/simplebench"
)

func main() {
	site := "https://bitfieldconsulting.com"
	concurrent := 10

	u, err := url.Parse(site)
	if err != nil {
		panic(err)
	}
	if u.Scheme == "" || u.Host == "" {
		panic(fmt.Sprintf("Invalid URL %s", u))
	}
	bencher := func() <-chan string {
		work := make(chan string, concurrent)
		go func() {
			defer close(work)
			for x := 0; x < concurrent; x++ {
				work <- u.String()
			}
		}()
		return work
	}

	loadgen := simplebench.NewLoadGen()
	work := bencher()
	for url := range work {
		loadgen.Wg.Add(1)
		go loadgen.DoRequest(url)
	}
	loadgen.Wg.Wait()
	fmt.Println("Bench is done")
}
