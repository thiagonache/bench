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
	"github.com/thiagonache/bench"
)

func TestNonOKStatusRecordedAsFailure(t *testing.T) {
	t.Parallel()

	server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		http.Error(rw, "ForceFailing", http.StatusTeapot)
	}))
	tester, err := bench.NewTester(server.URL,
		bench.WithHTTPClient(server.Client()),
		bench.WithStdout(io.Discard),
		bench.WithStderr(io.Discard),
	)
	if err != nil {
		t.Fatal(err)
	}
	tester.Run()
	want := bench.Stats{
		Requests: 1,
		Success:  0,
		Failures: 1,
	}
	got := tester.Stats()
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestNewTesterRequests(t *testing.T) {
	t.Parallel()
	tester, err := bench.NewTester("http://fake.url")
	if err != nil {
		t.Fatal(err)
	}

	wantReqs := 1
	gotReqs := tester.Requests()
	if wantReqs != gotReqs {
		t.Errorf("reqs: want %d, got %d", wantReqs, gotReqs)
	}

	tester, err = bench.NewTester(
		"http://fake.url",
		bench.WithRequests(10),
	)
	if err != nil {
		t.Fatal(err)
	}

	wantReqs = 10
	gotReqs = tester.Requests()
	if wantReqs != gotReqs {
		t.Errorf("reqs: want %d, got %d", wantReqs, gotReqs)
	}
}

func TestNewTesterRequestsInvalid(t *testing.T) {
	t.Parallel()
	_, err := bench.NewTester("http://fake.url", bench.WithRequests(0))
	if err == nil {
		t.Fatal("Error expected but not found")
	}
}
func TestNewTesterHTTPUserAgent(t *testing.T) {
	t.Parallel()
	tester, err := bench.NewTester("http://fake.url")
	if err != nil {
		t.Fatal(err)
	}
	wantUserAgent := "Bench 0.0.1 Alpha"
	gotUserAgent := tester.HTTPUserAgent()
	if wantUserAgent != gotUserAgent {
		t.Errorf("user-agent: want %q, got %q", wantUserAgent, gotUserAgent)
	}

	tester, err = bench.NewTester(
		"http://fake.url",
		bench.WithHTTPUserAgent("CustomUserAgent"),
	)
	if err != nil {
		t.Fatal(err)
	}
	wantUserAgent = "CustomUserAgent"
	gotUserAgent = tester.HTTPUserAgent()
	if wantUserAgent != gotUserAgent {
		t.Errorf("user-agent: want %q, got %q", wantUserAgent, gotUserAgent)
	}
}

func TestNewTesterHTTPClient(t *testing.T) {
	t.Parallel()
	tester, err := bench.NewTester("http://fake.url")
	if err != nil {
		t.Fatal(err)
	}

	wantHTTPClient := &http.Client{}
	wantHTTPClient.Timeout = 30 * time.Second
	gotHTTPClient := tester.HTTPClient()
	if !cmp.Equal(wantHTTPClient, gotHTTPClient) {
		t.Errorf(cmp.Diff(wantHTTPClient, gotHTTPClient))
	}

	tester, err = bench.NewTester(
		"http://fake.url",
		bench.WithHTTPClient(&http.Client{
			Timeout: 45 * time.Second,
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	wantHTTPClient = &http.Client{
		Timeout: 45 * time.Second,
	}
	gotHTTPClient = tester.HTTPClient()
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
			_, err := bench.NewTester(tC.url)
			if err == nil {
				t.Error("error expected but not found")
			}
		})
	}
}

func TestURLParseValid(t *testing.T) {
	t.Parallel()
	_, err := bench.NewTester("http://fake.url")
	if err != nil {
		t.Errorf("error not expected but found: %q", err.Error())
	}
}

func TestRun(t *testing.T) {
	t.Parallel()
	bench.Time = func() time.Time {
		return time.Time{}
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(rw, "HelloWorld")
	}))
	tester, err := bench.NewTester(server.URL,
		bench.WithRequests(100),
		bench.WithHTTPClient(server.Client()),
		bench.WithStdout(io.Discard),
		bench.WithStderr(io.Discard),
	)
	if err != nil {
		t.Fatal(err)
	}
	tester.Run()
	wantStats := bench.Stats{
		Requests: 100,
		Success:  100,
		Failures: 0,
	}
	gotStats := tester.Stats()
	if !cmp.Equal(wantStats, gotStats) {
		t.Error(cmp.Diff(wantStats, gotStats))
	}

	if gotStats.Requests != gotStats.Failures+gotStats.Success {
		t.Errorf("want failures plus success %d got %d", gotStats.Requests, gotStats.Failures+gotStats.Success)
	}

	gotTotalTime := time.Since(tester.StartTime())
	if gotTotalTime == 0 {
		t.Fatal("total time of zero seconds is invalid")
	}
}

func TestRecordStats(t *testing.T) {
	t.Parallel()
	tester, err := bench.NewTester("http://fake.url")
	if err != nil {
		t.Fatal(err)
	}
	tester.RecordRequest()
	tester.RecordSuccess()
	tester.RecordFailure()
	tester.TimeRecorder.RecordTime(100 * time.Millisecond)
	tester.TimeRecorder.RecordTime(200 * time.Millisecond)
	tester.TimeRecorder.RecordTime(100 * time.Millisecond)
	tester.TimeRecorder.RecordTime(50 * time.Millisecond)
	want := bench.Stats{
		Requests: 1,
		Success:  1,
		Failures: 1,
		Slowest:  200 * time.Millisecond,
		Fastest:  50 * time.Millisecond,
	}
	tester.SetFastestAndSlowest()
	got := tester.Stats()
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestLog(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	tester, err := bench.NewTester(
		"http://fake.url",
		bench.WithStdout(stdout),
		bench.WithStderr(stderr),
	)
	if err != nil {
		t.Fatal(err)
	}
	want := "this message goes to stdout"
	tester.LogStdOut("this message goes to stdout")
	got := stdout.String()
	if want != got {
		t.Errorf("want message %q in stdout but found %q", want, got)
	}

	want = "this message goes to stderr"
	tester.LogStdErr("this message goes to stderr")
	got = stderr.String()
	if want != got {
		t.Errorf("want message %q in stderr but found %q", want, got)
	}
}

func TestLogf(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	tester, err := bench.NewTester(
		"http://fake.url",
		bench.WithStdout(stdout),
		bench.WithStderr(stderr),
	)
	if err != nil {
		t.Fatal(err)
	}
	want := "this message goes to stdout"
	tester.LogFStdOut("this %s goes to %s", "message", "stdout")
	got := stdout.String()
	if want != got {
		t.Errorf("want message %q in stdout but found %q", want, got)
	}
	want = "this message goes to stderr"
	tester.LogFStdErr("this %s goes to %s", "message", "stderr")
	got = stderr.String()
	if want != got {
		t.Errorf("want message %q in stderr but found %q", want, got)
	}
}
