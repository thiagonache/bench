package bench_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/thiagonache/bench"
)

func TestRequestNonOK(t *testing.T) {
	t.Parallel()
	called := false
	server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		called = true
		http.Error(rw, "ForceFailing", http.StatusTeapot)
	}))
	loadgen, err := bench.NewLoadGen(server.URL,
		bench.WithHTTPClient(server.Client()),
		bench.WithStdout(io.Discard),
		bench.WithStderr(io.Discard),
	)
	if err != nil {
		t.Fatal(err)
	}
	loadgen.AddToWG(1)
	loadgen.DoRequest(server.URL)
	want := bench.Stats{
		Requests: 1,
		Success:  0,
		Failures: 1,
	}
	got := loadgen.GetStats()
	if !cmp.Equal(want, got, cmpopts.IgnoreFields(bench.Stats{}, "MU")) {
		t.Error(cmp.Diff(want, got, cmpopts.IgnoreFields(bench.Stats{}, "MU")))
	}
	if !called {
		t.Fatal("Request not made")
	}

}

func TestNewLoadGenDefault(t *testing.T) {
	t.Parallel()
	loadgen, err := bench.NewLoadGen("http://fake.url")
	if err != nil {
		t.Fatal(err)
	}

	wantReqs := 1
	gotReqs := loadgen.GetRequests()
	if wantReqs != gotReqs {
		t.Errorf("reqs: want %d, got %d", wantReqs, gotReqs)
	}

	wantUserAgent := "Bench 0.0.1 Alpha"
	gotUserAgent := loadgen.GetHTTPUserAgent()
	if wantUserAgent != gotUserAgent {
		t.Errorf("user-agent: want %q, got %q", wantUserAgent, gotUserAgent)
	}

	wantHTTPClient := http.DefaultClient
	wantHTTPClient.Timeout = 30 * time.Second
	gotHTTPClient := loadgen.GetHTTPClient()
	if !cmp.Equal(wantHTTPClient, gotHTTPClient) {
		t.Errorf(cmp.Diff(wantHTTPClient, gotHTTPClient))
	}
}

func TestNewLoadGenCustom(t *testing.T) {
	t.Parallel()
	client := http.Client{
		Timeout: 45,
	}
	loadgen, err := bench.NewLoadGen(
		"http://fake.url",
		bench.WithRequests(10),
		bench.WithHTTPUserAgent("CustomUserAgent"),
		bench.WithHTTPClient(&client),
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

	wantHTTPClient := &http.Client{
		Timeout: 45,
	}
	gotHTTPClient := loadgen.GetHTTPClient()
	if !cmp.Equal(wantHTTPClient, gotHTTPClient) {
		t.Errorf(cmp.Diff(wantHTTPClient, gotHTTPClient))
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
			_, err := bench.NewLoadGen(tC.url)
			if err == nil {
				t.Error("error expected but not found")
			}

		})
	}
}

func TestURLParseValid(t *testing.T) {
	t.Parallel()
	_, err := bench.NewLoadGen("http://fake.url")
	if err != nil {
		t.Error("error not expected but found")
	}
}

func TestRun(t *testing.T) {
	t.Parallel()
	server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(rw, "HelloWorld")
	}))
	loadgen, err := bench.NewLoadGen(server.URL,
		bench.WithRequests(1000),
		bench.WithHTTPClient(server.Client()),
		bench.WithStdout(io.Discard),
		bench.WithStderr(io.Discard),
	)
	if err != nil {
		t.Fatal(err)
	}
	loadgen.Run()
	wantStats := bench.Stats{
		Requests: 1000,
		Success:  1000,
		Failures: 0,
	}
	gotStats := loadgen.GetStats()
	if !cmp.Equal(wantStats, gotStats, cmpopts.IgnoreFields(bench.Stats{}, "MU")) {
		t.Error(cmp.Diff(wantStats, gotStats, cmpopts.IgnoreFields(bench.Stats{}, "MU")))
	}

	if gotStats.Requests != gotStats.Failures+gotStats.Success {
		t.Errorf("want failures plus success %d got %d", gotStats.Requests, gotStats.Failures+gotStats.Success)
	}

	gotTotalTime := time.Since(loadgen.GetStartTime())
	if gotTotalTime == 0 {
		t.Fatal("total time of zero seconds is invalid")
	}
}

func TestRecordStats(t *testing.T) {
	t.Parallel()
	loadgen, err := bench.NewLoadGen("http://fake.url")
	if err != nil {
		t.Fatal(err)
	}
	loadgen.RecordRequest()
	loadgen.RecordSuccess()
	loadgen.RecordFailure()
	loadgen.RecordTime(100 * time.Millisecond)
	loadgen.RecordTime(200 * time.Millisecond)
	want := bench.Stats{
		Requests:       1,
		Success:        1,
		Failures:       1,
		ExecutionsTime: []time.Duration{100 * time.Millisecond, 200 * time.Millisecond},
	}
	got := loadgen.GetStats()
	if !cmp.Equal(want, got, cmpopts.IgnoreFields(bench.Stats{}, "MU")) {
		t.Error(cmp.Diff(want, got, cmpopts.IgnoreFields(bench.Stats{}, "MU")))
	}
}

func TestLog(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	loadgen, err := bench.NewLoadGen(
		"http://fake.url",
		bench.WithStdout(stdout),
		bench.WithStderr(stderr),
	)
	if err != nil {
		t.Fatal(err)
	}
	want := "this message goes to stdout"
	loadgen.LogStdOut("this message goes to stdout")
	got := stdout.String()
	if want != got {
		t.Errorf("want message %q in stdout but found %q", want, got)
	}

	want = "this message goes to stderr"
	loadgen.LogStdErr("this message goes to stderr")
	got = stderr.String()
	if want != got {
		t.Errorf("want message %q in stderr but found %q", want, got)
	}
}
