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
	loadgen := simplebench.NewLoadGen(simplebench.WithRequests(10))
	server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		totalReqs++
		fmt.Fprintf(rw, "HelloWorld")
	}))
	loadgen.Client = server.Client()
	wantReqs := 10
	loadgen.Wg.Add(10)
	err := loadgen.DoRequest(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if wantReqs != totalReqs {
		t.Errorf("want %d got %d requests", wantReqs, totalReqs)
	}

}

func TestRequestNonOK(t *testing.T) {
	t.Parallel()
	called := false
	loadgen := simplebench.NewLoadGen()
	server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		called = true
		http.Error(rw, "ForceFailing", http.StatusTeapot)
	}))
	loadgen.Client = server.Client()
	loadgen.Wg.Add(1)
	err := loadgen.DoRequest(server.URL)
	if err == nil {
		t.Fatal("Expecting error but not found")
	}
	if !called {
		t.Fatal("Request not done")
	}

}

func TestNewLoadGenDefault(t *testing.T) {
	t.Parallel()
	loadgen := simplebench.NewLoadGen()

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
	loadgen := simplebench.NewLoadGen(
		simplebench.WithRequests(10),
		simplebench.WithHTTPUserAgent("CustomUserAgent"),
	)

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

func TestTotalTime(t *testing.T) {
	t.Parallel()
	loadgen := simplebench.NewLoadGen()
	loadgen.Wg.Add(1)
	loadgen.DoRequest("https://bitfieldconsulting.com")
	wantMinTotalTime := 100 * time.Millisecond
	gotTotalTime := loadgen.ElapsedTimeSinceStart()
	if gotTotalTime == 0 {
		t.Fatal("total time of zero seconds is invalid")
	}
	if wantMinTotalTime >= gotTotalTime {
		t.Errorf("total time: want bigger than %d, got %d", wantMinTotalTime, gotTotalTime)
	}
}
