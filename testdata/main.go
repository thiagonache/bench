package main

import (
	"fmt"
	"html"
	"math/rand"
	"net/http"
	"time"
)

func main() {
	rand.Seed(time.Now().UTC().Unix())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("GET /")
		time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
	})
	http.ListenAndServe(":33000", nil)
}
