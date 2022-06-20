package bench_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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
	tester, err := bench.NewTester(
		bench.WithURL(server.URL),
		bench.WithHTTPClient(server.Client()),
		bench.WithStdout(io.Discard),
		bench.WithStderr(io.Discard),
	)
	if err != nil {
		t.Fatal(err)
	}
	err = tester.Run()
	if err != nil {
		t.Fatal(err)
	}
	stats := tester.Stats()
	if stats.Requests != 1 {
		t.Errorf("want 1 request, got %d", stats.Requests)
	}
	if stats.Successes != 0 {
		t.Errorf("want 0 successes, got %d", stats.Successes)
	}
	if stats.Failures != 1 {
		t.Errorf("want 1 failure, got %d", stats.Failures)
	}
}

func TestFailedConnectionRecordedAsFailure(t *testing.T) {
	t.Parallel()

	tester, err := bench.NewTester(
		bench.WithURL("http://127.0.0.1:1000000"),
		bench.WithStdout(io.Discard),
		bench.WithStderr(io.Discard),
	)
	if err != nil {
		t.Fatal(err)
	}
	err = tester.Run()
	if err != nil {
		t.Fatal(err)
	}
	stats := tester.Stats()
	if stats.Requests != 1 {
		t.Errorf("want 1 request, got %d", stats.Requests)
	}
	if stats.Successes != 0 {
		t.Errorf("want 0 successes, got %d", stats.Successes)
	}
	if stats.Failures != 1 {
		t.Errorf("want 1 failure, got %d", stats.Failures)
	}
}

func TestNewTester_ByDefaultIsSetForDefaultNumRequests(t *testing.T) {
	t.Parallel()
	tester, err := bench.NewTester(
		bench.WithURL("http://fake.url"),
	)
	if err != nil {
		t.Fatal(err)
	}
	want := bench.DefaultNumRequests
	got := tester.Requests()
	if want != got {
		t.Errorf("want tester configured for default number of requests (%d), got %d", want, got)
	}
}

func TestNewTester_ByDefaultIsSetForDefaultOutputPath(t *testing.T) {
	t.Parallel()
	tester, err := bench.NewTester(
		bench.WithURL("http://fake.url"),
	)
	if err != nil {
		t.Fatal(err)
	}
	want := bench.DefaultOutputPath
	got := tester.OutputPath()
	if want != got {
		t.Errorf("want tester output path for default output path (%q), got %q", want, got)
	}
}

func TestNewTester_ByDefaultIsSetForDefaultConcurrency(t *testing.T) {
	t.Parallel()
	tester, err := bench.NewTester(
		bench.WithURL("http://fake.url"),
	)
	if err != nil {
		t.Fatal(err)
	}
	want := bench.DefaultConcurrency
	got := tester.Concurrency()
	if want != got {
		t.Errorf("want tester concurrency for default concurrency (%d), got %d", want, got)
	}
}

func TestNewTester_ByDefaultIsSetForHTTPGetMethod(t *testing.T) {
	t.Parallel()
	tester, err := bench.NewTester(
		bench.WithURL("http://fake.url"),
	)
	if err != nil {
		t.Fatal(err)
	}
	want := http.MethodGet
	got := tester.HTTPMethod()
	if want != got {
		t.Errorf("want tester default method (%q), got %q", want, got)
	}
}

func TestNewTester_WithHTTPMethodXSetsMethodX(t *testing.T) {
	t.Parallel()
	tester, err := bench.NewTester(
		bench.WithURL("http://fake.url"),
		bench.WithHTTPMethod(http.MethodPost),
	)
	if err != nil {
		t.Fatal(err)
	}
	want := http.MethodPost
	got := tester.HTTPMethod()
	if want != got {
		t.Errorf("want http method %q, got %q", want, got)
	}
}

func TestNewTester_WithHTTPMethodDownCaseSetsUpperCase(t *testing.T) {
	t.Parallel()
	tester, err := bench.NewTester(
		bench.WithURL("http://fake.url"),
		bench.WithHTTPMethod("post"),
	)
	if err != nil {
		t.Fatal(err)
	}
	want := http.MethodPost
	got := tester.HTTPMethod()
	if want != got {
		t.Errorf("want http method %q, got %q", want, got)
	}
}

func TestFromArgs_MFlagSetsHTTPMethod(t *testing.T) {
	t.Parallel()
	args := []string{"-m", "DELETE", "-u", "http://fake.url/users/1"}
	tester, err := bench.NewTester(
		bench.WithStderr(io.Discard),
		bench.FromArgs(args),
	)
	if err != nil {
		t.Fatal(err)
	}
	want := http.MethodDelete
	got := tester.HTTPMethod()
	if want != got {
		t.Errorf("wants -m flag to set http method to %q, got %q", want, got)
	}
}

func TestFromArgs_MFlagDownCaseSetsUpperCase(t *testing.T) {
	t.Parallel()
	args := []string{"-m", "delete", "-u", "http://fake.url/users/1"}
	tester, err := bench.NewTester(
		bench.WithStderr(io.Discard),
		bench.FromArgs(args),
	)
	if err != nil {
		t.Fatal(err)
	}
	want := http.MethodDelete
	got := tester.HTTPMethod()
	if want != got {
		t.Errorf("wants -m flag to set http method to %q, got %q", want, got)
	}
}

func TestRun_MethodXDoesMethodXHTTPRequest(t *testing.T) {
	t.Parallel()
	server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodDelete:
			fmt.Fprintln(rw, "OK")
		default:
			http.Error(rw, "Error", http.StatusMethodNotAllowed)
		}
	}))
	tester, err := bench.NewTester(
		bench.WithURL(server.URL),
		bench.WithHTTPMethod(http.MethodDelete),
		bench.WithHTTPClient(server.Client()),
		bench.WithStderr(io.Discard),
		bench.WithStdout(io.Discard),
	)
	if err != nil {
		t.Fatal(err)
	}
	err = tester.Run()
	if err != nil {
		t.Fatal(err)
	}
	if tester.Stats().Failures > 0 {
		t.Errorf("want failures to be zero but got %d", tester.Stats().Failures)
	}
}

func TestNewTester_WithNConcurrentSetsNConcurrency(t *testing.T) {
	t.Parallel()
	tester, err := bench.NewTester(
		bench.WithURL("http://fake.url"),
		bench.WithConcurrency(10),
	)
	if err != nil {
		t.Fatal(err)
	}
	got := tester.Concurrency()
	if got != 10 {
		t.Errorf("want tester configured for 10 concurrent requests, got %d", got)
	}
}

func TestFromArgs_CFlagSetsNConcurrency(t *testing.T) {
	t.Parallel()
	args := []string{"-c", "10", "-u", "http://fake.url"}
	tester, err := bench.NewTester(
		bench.WithStderr(io.Discard),
		bench.FromArgs(args),
	)
	if err != nil {
		t.Fatal(err)
	}
	want := 10
	got := tester.Concurrency()
	if want != got {
		t.Errorf("reqs: want %d, got %d", want, got)
	}
}

func TestNewTester_WithOutputPathSetsOutputPath(t *testing.T) {
	t.Parallel()
	tester, err := bench.NewTester(
		bench.WithURL("http://fake.url"),
		bench.WithOutputPath("/tmp"),
	)
	if err != nil {
		t.Fatal(err)
	}
	want := "/tmp"
	got := tester.OutputPath()
	if want != got {
		t.Errorf("want tester output path configured for /tmp, got %q", got)
	}
}

func TestNewTester_WithNRequestsIsConfiguredForNRequests(t *testing.T) {
	t.Parallel()
	tester, err := bench.NewTester(
		bench.WithURL("http://fake.url"),
		bench.WithRequests(10),
	)
	if err != nil {
		t.Fatal(err)
	}
	got := tester.Requests()
	if got != 10 {
		t.Errorf("want tester configured for 10 requests, got %d", got)
	}
}

func TestNewTester_WithInvalidRequestsReturnsError(t *testing.T) {
	t.Parallel()
	_, err := bench.NewTester(
		bench.WithURL("http://fake.url"),
		bench.WithRequests(-1),
	)
	if err == nil {
		t.Fatal("want error for invalid number of requests (-1)")
	}
}

func TestNewTester_ByDefaultSetsDefaultHTTPUserAgent(t *testing.T) {
	t.Parallel()
	tester, err := bench.NewTester(
		bench.WithURL("http://fake.url"),
	)
	if err != nil {
		t.Fatal(err)
	}
	want := bench.DefaultUserAgent
	got := tester.HTTPUserAgent()
	if want != got {
		t.Errorf("want default user agent (%q), got %q", want, got)
	}
}

func TestNewTester_WithUserAgentXSetsUserAgentX(t *testing.T) {
	t.Parallel()
	tester, err := bench.NewTester(
		bench.WithURL("http://fake.url"),
		bench.WithHTTPUserAgent("CustomUserAgent"),
	)
	if err != nil {
		t.Fatal(err)
	}
	want := "CustomUserAgent"
	got := tester.HTTPUserAgent()
	if want != got {
		t.Errorf("user-agent: want %q, got %q", want, got)
	}
}

func TestNewTester_ByDefaultSetsDefaultHTTPClient(t *testing.T) {
	t.Parallel()
	tester, err := bench.NewTester(
		bench.WithURL("http://fake.url"),
	)
	if err != nil {
		t.Fatal(err)
	}

	want := bench.DefaultHTTPClient
	got := tester.HTTPClient()
	if !cmp.Equal(want, got) {
		t.Errorf(cmp.Diff(want, got))
	}
}

func TestNewTester_WithHTTPClientXSetsHTTPClientX(t *testing.T) {
	t.Parallel()
	tester, err := bench.NewTester(
		bench.WithURL("http://fake.url"),
		bench.WithHTTPClient(&http.Client{
			Timeout: 45 * time.Second,
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	want := &http.Client{
		Timeout: 45 * time.Second,
	}
	got := tester.HTTPClient()
	if !cmp.Equal(want, got) {
		t.Errorf(cmp.Diff(want, got))
	}
}

func TestNewTester_WithInvalidURLReturnsError(t *testing.T) {
	t.Parallel()
	inputs := []string{
		"bogus-no-scheme-or-domain",
		"bogus-no-host://",
		"bogus-no-scheme.fake",
		"http://\n.fake",
	}
	for _, url := range inputs {
		_, err := bench.NewTester(
			bench.WithURL(url),
		)
		if err == nil {
			t.Errorf("want error for invalid URL %q", url)
		}
	}
}

func TestNewTester_WithValidURLReturnsNoError(t *testing.T) {
	t.Parallel()
	_, err := bench.NewTester(
		bench.WithURL("http://fake.url"),
	)
	if err != nil {
		t.Errorf("error not expected but found: %q", err)
	}
}

func TestRunReturnsValidStatsAndTime(t *testing.T) {
	t.Parallel()
	server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(rw, "HelloWorld")
	}))
	tester, err := bench.NewTester(
		bench.WithURL(server.URL),
		bench.WithRequests(100),
		bench.WithHTTPClient(server.Client()),
		bench.WithStdout(io.Discard),
		bench.WithStderr(io.Discard),
	)
	if err != nil {
		t.Fatal(err)
	}
	err = tester.Run()
	if err != nil {
		t.Fatal(err)
	}
	stats := tester.Stats()
	if stats.Requests != 100 {
		t.Errorf("want 100 requests made, got %d", stats.Requests)
	}
	if stats.Successes != 100 {
		t.Errorf("want 100 successes, got %d", stats.Successes)
	}
	if stats.Failures != 0 {
		t.Errorf("want 0 failures, got %d", stats.Failures)
	}
	if stats.Requests != stats.Successes+stats.Failures {
		t.Error("want total requests to be the sum of successes + failures")
	}
	if tester.EndAt() == 0 {
		t.Fatal("zero milliseconds is an invalid time")
	}
}

func TestRecordTime_CalledMultipleTimesSetCorrectPercentilesAndReturnsNoError(t *testing.T) {
	t.Parallel()
	tester, err := bench.NewTester(
		bench.WithURL("http://fake.url"),
	)
	if err != nil {
		t.Fatal(err)
	}
	tester.TimeRecorder.RecordTime(5)
	tester.TimeRecorder.RecordTime(6)
	tester.TimeRecorder.RecordTime(7)
	tester.TimeRecorder.RecordTime(8)
	tester.TimeRecorder.RecordTime(10)
	tester.TimeRecorder.RecordTime(11)
	tester.TimeRecorder.RecordTime(13)

	tester.CalculatePercentiles()
	stats := tester.Stats()
	if stats.P50 != 8 {
		t.Errorf("want 50th percentile request time of 8ms, got %v", stats.P50)
	}
	if stats.P90 != 11 {
		t.Errorf("want 90th percentile request time of 11ms, got %v", stats.P90)
	}
	if stats.P99 != 13 {
		t.Errorf("want 99th percentile request time of 13ms, got %v", stats.P99)
	}
}

func TestLogPrintsToStdoutAndStderr(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	tester, err := bench.NewTester(
		bench.WithURL("http://fake.url"),
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

func TestLogfPrintsToStdoutAndStderr(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	tester, err := bench.NewTester(
		bench.WithURL("http://fake.url"),
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

func TestFromArgs_RFlagSetsNRequests(t *testing.T) {
	t.Parallel()
	args := []string{"-r", "10", "-u", "http://fake.url"}
	tester, err := bench.NewTester(
		bench.WithStderr(io.Discard),
		bench.FromArgs(args),
	)
	if err != nil {
		t.Fatal(err)
	}
	wantReqs := 10
	gotReqs := tester.Requests()
	if wantReqs != gotReqs {
		t.Errorf("reqs: want %d, got %d", wantReqs, gotReqs)
	}
}

func TestNewTester_WithGraphsSetsGraphsMode(t *testing.T) {
	t.Parallel()
	tester, err := bench.NewTester(
		bench.WithURL("http://fake.url"),
		bench.WithStderr(io.Discard),
		bench.WithGraphs(true),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !tester.Graphs() {
		t.Error("want graphs to be true")
	}
}

func TestNewTester_ByDefaultSetsNoGraphsMode(t *testing.T) {
	t.Parallel()
	tester, err := bench.NewTester(
		bench.WithURL("http://fake.url"),
		bench.WithStderr(io.Discard),
	)
	if err != nil {
		t.Fatal(err)
	}
	if tester.Graphs() {
		t.Error("want graphs to be false")
	}
}

func TestFromArgs_GFlagSetsGraphsMode(t *testing.T) {
	t.Parallel()
	args := []string{"-g", "-u", "http://fake.url"}
	tester, err := bench.NewTester(
		bench.WithStderr(io.Discard),
		bench.FromArgs(args),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !tester.Graphs() {
		t.Error("want graphs to be true")
	}
}

func TestFromArgs_ByDefaultSetsNoGraphsMode(t *testing.T) {
	t.Parallel()
	args := []string{"-u", "http://fake.url"}
	tester, err := bench.NewTester(
		bench.WithStderr(io.Discard),
		bench.FromArgs(args),
	)
	if err != nil {
		t.Fatal(err)
	}
	if tester.Graphs() {
		t.Error("want graphs to be false")
	}
}

func TestNewTester_WithTrueGraphsModeGeneratesGraphs(t *testing.T) {
	t.Parallel()
	server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(rw, "HelloWorld")
	}))
	tempDir := t.TempDir()
	tester, err := bench.NewTester(
		bench.WithURL(server.URL),
		bench.WithHTTPClient(server.Client()),
		bench.WithStdout(io.Discard),
		bench.WithStderr(io.Discard),
		bench.WithOutputPath(tempDir),
		bench.WithGraphs(true),
	)
	if err != nil {
		t.Fatal(err)
	}
	err = tester.Run()
	if err != nil {
		t.Fatal(err)
	}
	filePath := fmt.Sprintf("%s/%s", tester.OutputPath(), "boxplot.png")
	_, err = os.Stat(filePath)
	if err != nil {
		t.Errorf("want file %q to exist", filePath)
	}
	filePath = fmt.Sprintf("%s/%s", tester.OutputPath(), "histogram.png")
	_, err = os.Stat(filePath)
	if err != nil {
		t.Errorf("want file %q to exist", filePath)
	}
}

func TestNewTester_ByDefaultDoesNotGenerateGraphs(t *testing.T) {
	t.Parallel()

	server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(rw, "HelloWorld")
	}))
	tempDir := t.TempDir()
	tester, err := bench.NewTester(
		bench.WithURL(server.URL),
		bench.WithHTTPClient(server.Client()),
		bench.WithStdout(io.Discard),
		bench.WithStderr(io.Discard),
		bench.WithOutputPath(tempDir),
	)
	if err != nil {
		t.Fatal(err)
	}
	err = tester.Run()
	if err != nil {
		t.Fatal(err)
	}
	filePath := fmt.Sprintf("%s/%s", tester.OutputPath(), "boxplot.png")
	_, err = os.Stat(filePath)
	if err == nil {
		t.Errorf("want file %q to not exist. Error found: %v", filePath, err)
	}
	filePath = fmt.Sprintf("%s/%s", tester.OutputPath(), "histogram.png")
	_, err = os.Stat(filePath)
	if err == nil {
		t.Errorf("want file %q to not exist. Error found: %v", filePath, err)
	}
}

func TestNewTester_WithURLSetsTesterURL(t *testing.T) {
	t.Parallel()
	tester, err := bench.NewTester(
		bench.WithURL("http://fake.url"),
	)
	if err != nil {
		t.Fatal(err)
	}
	want := "http://fake.url"
	if want != tester.URL {
		t.Fatalf("want tester URL %q, got %q", want, tester.URL)
	}
}

func TestFromArgs_WithArgRSetsTesterURL(t *testing.T) {
	t.Parallel()
	tester, err := bench.NewTester(
		bench.WithStderr(io.Discard),
		bench.FromArgs([]string{"-r", "10", "-u", "http://fake.url"}),
	)
	if err != nil {
		t.Fatal(err)
	}
	want := "http://fake.url"
	if want != tester.URL {
		t.Fatalf("want tester URL %q, got %q", want, tester.URL)
	}
}

func TestNewTester_ByDefaultReturnsErrorNoURL(t *testing.T) {
	t.Parallel()
	_, err := bench.NewTester(
		bench.WithStderr(io.Discard),
	)
	if err == nil {
		t.Error("want error if no URL set")
	}
}

func TestFromArgs_GivenNoArgsReturnsUsageMessage(t *testing.T) {
	t.Parallel()
	_, err := bench.NewTester(
		bench.WithStderr(io.Discard),
		bench.FromArgs([]string{}),
	)
	if !errors.Is(err, bench.ErrNoArgs) {
		t.Errorf("want ErrNoArgs error if no args supplied, got %v", err)
	}
}

func TestFromArgs_WithoutUFlagReturnsErrorNoURL(t *testing.T) {
	t.Parallel()
	_, err := bench.NewTester(
		bench.WithStderr(io.Discard),
		bench.FromArgs([]string{"-r", "10"}),
	)
	if err == nil {
		t.Error("want error if no URL set")
	}
}

func TestNewTester_WithNilStdoutReturnsErrorValueCannotBeNil(t *testing.T) {
	t.Parallel()
	_, err := bench.NewTester(
		bench.WithURL("http://fake.url"),
		bench.WithStdout(nil),
	)
	if !errors.Is(err, bench.ErrValueCannotBeNil) {
		t.Errorf("want ErrValueCannotBeNil error if stdout is nil, got %v", err)
	}
}

func TestNewTester_WithNilStderrReturnsErrorValueCannotBeNil(t *testing.T) {
	t.Parallel()
	_, err := bench.NewTester(
		bench.WithURL("http://fake.url"),
		bench.WithStderr(nil),
	)
	if !errors.Is(err, bench.ErrValueCannotBeNil) {
		t.Errorf("want ErrValueCannotBeNil error if stderr is nil, got %v", err)
	}
}

func TestReadStatsFile_ErrorsIfFileUnreadable(t *testing.T) {
	_, err := bench.ReadStatsFile("bogus")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("want error os.ErrNotExist got %v", err)
	}
}

func TestReadStats_PopulatesCorrectStats(t *testing.T) {
	t.Parallel()
	statsReader := strings.NewReader(`Site: https://google.com
Requests: 10
Successes: 9
Failures: 1
P50(ms): 221.607
P90(ms): 261.139
P99(ms): 319.947`)
	got, err := bench.ReadStats(statsReader)
	if err != nil {
		t.Fatal(err)
	}
	want := bench.Stats{
		P50:       221.607,
		P90:       261.139,
		P99:       319.947,
		Failures:  1,
		Requests:  10,
		Successes: 9,
		URL:       "https://google.com",
	}

	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestReadStatsFile_PopulatesCorrectStatsFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := dir + "/stats.txt"
	file, err := os.Create(path)
	fmt.Fprint(file, bench.Stats{
		P50:       20,
		P90:       30,
		P99:       100,
		Failures:  2,
		Requests:  20,
		Successes: 18,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := bench.Stats{
		P50:       20,
		P90:       30,
		P99:       100,
		Failures:  2,
		Requests:  20,
		Successes: 18,
	}
	got, err := bench.ReadStatsFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestStatsStringer_PopulatesCorrectStats(t *testing.T) {
	t.Parallel()
	stats := bench.Stats{
		URL:       "http://fake.url",
		P50:       100.123,
		P90:       150.000,
		P99:       198.465,
		Failures:  2,
		Requests:  20,
		Successes: 18,
	}
	output := &bytes.Buffer{}
	fmt.Fprint(output, stats)
	want := `Site: http://fake.url
Requests: 20
Successes: 18
Failures: 2
P50(ms): 100.123
P90(ms): 150.000
P99(ms): 198.465`
	got := output.String()
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestCompareStats_StringerPrintsExpectedMessage(t *testing.T) {
	t.Parallel()
	cs := bench.CompareStats{
		S1: bench.Stats{
			URL: "http://fake.url",
			P50: 100,
			P90: 110,
			P99: 120,
		},
		S2: bench.Stats{
			URL: "http://fake.url",
			P50: 99,
			P90: 100,
			P99: 101,
		},
	}
	want := `Site http://fake.url
Metric              Old                 New                 Delta               Percentage
P50(ms)             100.000             99.000              -1.000              -1.00
P90(ms)             110.000             100.000             -10.000             -9.09
P99(ms)             120.000             101.000             -19.000             -15.83
`
	got := cs.String()
	if !cmp.Equal(want, got) {
		t.Fatal(cmp.Diff(want, got))
	}
}

func TestRunCLI_ErrorsIfNoArgs(t *testing.T) {
	t.Parallel()

	err := bench.RunCLI(io.Discard, []string{})
	if !errors.Is(err, bench.ErrUnkownSubCommand) {
		t.Fatalf("want error bench.ErrUnkownSubCommand got %v", err)
	}
}

func TestRunCLI_ErrorsIfUnknownSubCommand(t *testing.T) {
	t.Parallel()

	err := bench.RunCLI(io.Discard, []string{"bogus"})
	if !errors.Is(err, bench.ErrUnkownSubCommand) {
		t.Fatalf("want error bench.ErrUnkownSubCommand got %v", err)
	}
}

func TestRunCLI_RunPrintsStats(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	err := bench.RunCLI(stdout, []string{"run", "-u", "https://bitfieldconsulting.com"})
	if err != nil {
		t.Error(err)
	}
	if !strings.HasPrefix(stdout.String(), "Site: https://bitfieldconsulting.com") {
		t.Errorf(`want output to start with "Site: https://bitfieldconsulting.com" but not found in string %q`, stdout.String())
	}
}

func TestRunCLI_CMPPrintsComparison(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	err := bench.RunCLI(stdout, []string{"cmp", "testdata/statsfile1.txt", "testdata/statsfile2.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Metric") {
		t.Errorf(`want output to contains Metric but not found in string %q`, stdout.String())
	}
}

func TestCMPRun_ErrorsIfLessThanTwoArgs(t *testing.T) {
	t.Parallel()

	err := bench.CMPRun(io.Discard, []string{"bogus"})
	if !errors.Is(err, bench.ErrCMPNoArgs) {
		t.Errorf("want error bench.ErrCMPNoArgs if just one arg got %v", err)
	}

	err = bench.CMPRun(io.Discard, []string{})
	if !errors.Is(err, bench.ErrCMPNoArgs) {
		t.Errorf("want error bench.ErrCMPNoArgs if no args got %v", err)
	}
}

func TestStatsStringerPrintsExpectedMessage(t *testing.T) {
	t.Parallel()

	stats := bench.Stats{
		URL:       "http://fake.url",
		Requests:  100,
		Successes: 100,
		P50:       800.231,
		P90:       880.000,
		P99:       901.987,
	}
	want := `Site: http://fake.url
Requests: 100
Successes: 100
Failures: 0
P50(ms): 800.231
P90(ms): 880.000
P99(ms): 901.987`
	got := stats.String()
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestNewTester_BodyByDefaultIsNil(t *testing.T) {
	t.Parallel()
	tester, err := bench.NewTester(
		bench.WithURL("http://fake.url"),
	)
	if err != nil {
		t.Fatal(err)
	}
	body := tester.Body()
	if body != "" {
		t.Errorf("want tester default body to be empty, got %q", body)
	}
}

func TestNewTester_WithBodyXSetsBodyX(t *testing.T) {
	t.Parallel()
	tester, err := bench.NewTester(
		bench.WithURL("http://fake.url"),
		bench.WithBody(`{"language": "golang"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"language": "golang"}`
	got := tester.Body()
	if want != got {
		t.Errorf("want tester body to be %q, got %q", want, got)
	}
}

func TestFromArgs_BFlagSetsRequestBody(t *testing.T) {
	t.Parallel()
	args := []string{"-b", `{"language": "golang"}`, "-u", "http://fake.url"}
	tester, err := bench.NewTester(
		bench.WithStderr(io.Discard),
		bench.FromArgs(args),
	)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"language": "golang"}`
	got := tester.Body()
	if want != got {
		t.Errorf("want tester body to be %q, got %q", want, got)
	}
}

func TestRun_WithBodySendsCorrectBody(t *testing.T) {
	t.Parallel()
	server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(rw, "CannotReadBody", http.StatusInternalServerError)
		}
		if string(body) != `{"language": "golang"}` {
			http.Error(rw, "Error", http.StatusInternalServerError)
		}
		fmt.Fprintln(rw, "OK")
	}))
	tester, err := bench.NewTester(
		bench.WithURL(server.URL),
		bench.WithBody(`{"language": "golang"}`),
		bench.WithHTTPClient(server.Client()),
		bench.WithStderr(io.Discard),
		bench.WithStdout(io.Discard),
	)
	if err != nil {
		t.Fatal(err)
	}
	err = tester.Run()
	if err != nil {
		t.Fatal(err)
	}
	if tester.Stats().Failures > 0 {
		t.Errorf("want failures to be zero but got %d", tester.Stats().Failures)
	}
}
