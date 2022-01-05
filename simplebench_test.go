package simplebench_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/thiagonache/simplebench"
)

func TestRequest(t *testing.T) {
	t.Parallel()
	totalReqs := 0

	server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		totalReqs++
		fmt.Fprintf(rw, "HelloWorld")
	}))
	loadgen, err := simplebench.NewLoadGen(server.URL, simplebench.WithRequests(10))
	if err != nil {
		t.Fatal(err)
	}
	wantURL := server.URL
	gotURL := loadgen.URL
	if wantURL != gotURL {
		t.Errorf("want %q got %q", wantURL, gotURL)
	}
	loadgen.Client = server.Client()
	wantReqs := 10
	loadgen.Wg.Add(10)
	err = loadgen.DoRequest()
	if err != nil {
		t.Fatal(err)
	}
	if wantReqs != totalReqs {
		t.Errorf("want %d got %d requests", wantReqs, totalReqs)
	}
	gotTotalTime := time.Since(loadgen.StartAt)
	if gotTotalTime == 0 {
		t.Fatal("total time of zero seconds is invalid")
	}
}

func TestRequestNonOK(t *testing.T) {
	t.Parallel()
	called := false
	server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		called = true
		http.Error(rw, "ForceFailing", http.StatusTeapot)
	}))
	loadgen, err := simplebench.NewLoadGen(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	loadgen.Client = server.Client()
	loadgen.Wg.Add(1)
	err = loadgen.DoRequest()
	if err == nil {
		t.Fatal("Expecting error but not found")
	}
	if !called {
		t.Fatal("Request not made")
	}

}

func TestNewLoadGenDefault(t *testing.T) {
	t.Parallel()
	loadgen, err := simplebench.NewLoadGen("http://fake.url")
	if err != nil {
		t.Fatal(err)
	}

	wantReqs := 1
	gotReqs := loadgen.GetRequests()
	if wantReqs != gotReqs {
		t.Errorf("reqs: want %d, got %d", wantReqs, gotReqs)
	}

	wantUserAgent := "SimpleBench 0.0.1 Alpha"
	gotUserAgent := loadgen.GetHTTPUserAgent()
	if wantUserAgent != gotUserAgent {
		t.Errorf("user-agent: want %q, got %q", wantUserAgent, gotUserAgent)
	}
}

func TestNewLoadGenCustom(t *testing.T) {
	t.Parallel()
	loadgen, err := simplebench.NewLoadGen(
		"http://fake.url",
		simplebench.WithRequests(10),
		simplebench.WithHTTPUserAgent("CustomUserAgent"),
	)
	if err != nil {
		t.Fatal(err)
	}

	wantReqs := 10
	gotReqs := loadgen.GetRequests()
	if wantReqs != gotReqs {
		t.Errorf("reqs: want %d, got %d", wantReqs, gotReqs)
	}

	wantUserAgent := "CustomUserAgent"
	gotUserAgent := loadgen.GetHTTPUserAgent()
	if wantUserAgent != gotUserAgent {
		t.Errorf("user-agent: want %q, got %q", wantUserAgent, gotUserAgent)
	}
}

func TestURLParseInvalid(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		desc string
		url  string
	}{
		{
			desc: "Test bogus http URL",
			url:  "bogus",
		},
		{
			desc: "Test http:// http URL",
			url:  "http://",
		},
		{
			desc: "Test fake.url http URL",
			url:  "fake.url",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			_, err := simplebench.NewLoadGen(tC.url)
			if err == nil {
				t.Error("error expected but not found")
			}

		})
	}
}

func TestURLParseValid(t *testing.T) {
	t.Parallel()
	_, err := simplebench.NewLoadGen("http://fake.url")
	if err != nil {
		t.Error("error not expected but found")
	}
}

func TestWorkGenerator(t *testing.T) {
	t.Parallel()
	loadgen, err := simplebench.NewLoadGen("http://fake.url", simplebench.WithRequests(2))
	if err != nil {
		t.Fatal(err)
	}
	wantMessages := 2
	gotMessages := 0
	work := loadgen.GenerateWork()
	for range work {
		gotMessages++
	}
	if wantMessages != gotMessages {
		t.Errorf("want %d messages in the channel but got %d", wantMessages, gotMessages)
	}
}
