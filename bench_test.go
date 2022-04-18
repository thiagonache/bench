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

func TestNewTester_WithNConcurrentSetsNConcurrenty(t *testing.T) {
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

func TestRecordTime_CalledMultipleTimesSetsCorrectStatsAndNoError(t *testing.T) {
	t.Parallel()
	tester, err := bench.NewTester(
		bench.WithURL("http://fake.url"),
	)
	if err != nil {
		t.Fatal(err)
	}
	tester.TimeRecorder.RecordTime(50)
	tester.TimeRecorder.RecordTime(100)
	tester.TimeRecorder.RecordTime(200)
	tester.TimeRecorder.RecordTime(100)
	tester.TimeRecorder.RecordTime(50)

	err = tester.SetMetrics()
	if err != nil {
		t.Fatal(err)
	}
	stats := tester.Stats()
	if stats.Mean != 100 {
		t.Errorf("want 100ms mean time, got %v", stats.Mean)
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

	err = tester.SetMetrics()
	if err != nil {
		t.Fatal(err)
	}
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

func TestSetMetrics_ErrorsIfRecordTimeIsNotCalled(t *testing.T) {
	t.Parallel()
	tester, err := bench.NewTester(
		bench.WithURL("http://fake.url"),
	)
	if err != nil {
		t.Fatal(err)
	}

	err = tester.SetMetrics()
	if !errors.Is(err, bench.ErrTimeNotRecorded) {
		t.Errorf("want ErrTimeNotRecorded error if there is no ExecutionsTime, got %q", err)
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
	t.Cleanup(func() {
		err := os.RemoveAll(tester.OutputPath())
		if err != nil {
			fmt.Printf("cannot delete %s\n", tester.OutputPath())
		}
	})
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
	t.Cleanup(func() {
		err := os.RemoveAll(tester.OutputPath())
		if err != nil {
			fmt.Printf("cannot delete %s\n", tester.OutputPath())
		}
	})
}

func TestNewTester_WithStatsSetsExportStatsMode(t *testing.T) {
	t.Parallel()
	tester, err := bench.NewTester(
		bench.WithURL("http://fake.url"),
		bench.WithStderr(io.Discard),
		bench.WithExportStats(true),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !tester.ExportStats() {
		t.Error("want ExportStats to be true")
	}
}

func TestNewTester_ByDefaultSetsNoExportStatsMode(t *testing.T) {
	t.Parallel()
	tester, err := bench.NewTester(
		bench.WithURL("http://fake.url"),
		bench.WithStderr(io.Discard),
	)
	if err != nil {
		t.Fatal(err)
	}
	if tester.ExportStats() {
		t.Error("want ExportStats to be false")
	}
}

func TestFromArgs_WithSFlagEnablesExportStatsMode(t *testing.T) {
	t.Parallel()
	args := []string{"-s", "-u", "http://fake.url"}
	tester, err := bench.NewTester(
		bench.WithStderr(io.Discard),
		bench.FromArgs(args),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !tester.ExportStats() {
		t.Error("want ExportStats to be true")
	}
}

func TestFromArgs_WithoutSFlagDisablesExportStatsMode(t *testing.T) {
	t.Parallel()
	args := []string{"-u", "http://fake.url"}
	tester, err := bench.NewTester(
		bench.WithStderr(io.Discard),
		bench.FromArgs(args),
	)
	if err != nil {
		t.Fatal(err)
	}
	if tester.ExportStats() {
		t.Error("want ExportStats to be false")
	}
}

func TestExportStatsFlagTrueGenerateStatsFile(t *testing.T) {
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
		bench.WithExportStats(true),
	)
	if err != nil {
		t.Fatal(err)
	}
	err = tester.Run()
	if err != nil {
		t.Fatal(err)
	}
	filePath := fmt.Sprintf("%s/%s", tester.OutputPath(), "statsfile.txt")
	_, err = os.Stat(filePath)
	if err != nil {
		t.Errorf("want file %q to exist", filePath)
	}

	t.Cleanup(func() {
		err := os.RemoveAll(tester.OutputPath())
		if err != nil {
			fmt.Printf("cannot delete %s\n", tester.OutputPath())
		}
	})
}

func TestNewTester_ByDefaultDoesNotGenerateStatsFile(t *testing.T) {
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
	filePath := fmt.Sprintf("%s/%s", tester.OutputPath(), "statsfile.txt")
	_, err = os.Stat(filePath)
	if err == nil {
		t.Errorf("want file %q to not exist. Error found: %v", filePath, err)
	}
	t.Cleanup(func() {
		err := os.RemoveAll(tester.OutputPath())
		if err != nil {
			fmt.Printf("cannot delete %s\n", tester.OutputPath())
		}
	})
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

func TestFromArgs_WithURLSetsTesterURL(t *testing.T) {
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
	if !errors.Is(err, bench.ErrNoURL) {
		t.Errorf("want ErrNoURL error if no URL set, got %v", err)
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
	if !errors.Is(err, bench.ErrNoURL) {
		t.Errorf("want ErrNoURL error if no URL set, got %v", err)
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

func TestReadStatsFiles_ReadsTwoFilesAndReturnsCorrectStatsCompares(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	stats1 := bench.Stats{
		P50:       20,
		P90:       30,
		P99:       100,
		Failures:  2,
		Requests:  20,
		Successes: 18,
	}
	f1 := dir + "/stats1.txt"
	err := bench.WriteStatsFile(f1, stats1)
	if err != nil {
		t.Fatal(err)
	}

	stats2 := bench.Stats{
		P50:       5,
		P90:       33,
		P99:       99,
		Failures:  1,
		Requests:  40,
		Successes: 19,
	}
	f2 := dir + "/stats2.txt"
	err = bench.WriteStatsFile(f2, stats2)
	if err != nil {
		t.Fatal(err)
	}
	got, err := bench.ReadStatsFiles(f1, f2)
	if err != nil {
		t.Fatal(err)
	}
	want := bench.CompareStats{
		S1: bench.Stats{
			P50:       20,
			P90:       30,
			P99:       100,
			Failures:  2,
			Requests:  20,
			Successes: 18,
		},
		S2: bench.Stats{
			P50:       5,
			P90:       33,
			P99:       99,
			Failures:  1,
			Requests:  40,
			Successes: 19},
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}

	t.Cleanup(func() {
		err := os.RemoveAll(dir)
		if err != nil {
			fmt.Printf("cannot delete %s\n", dir)
		}
	})
}

func TestReadStatsFiles_ErrorsIfOneOrBothFilesUnreadable(t *testing.T) {
	_, err := bench.ReadStatsFiles("bogus", "even more bogus")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("want error os.ErrNotExist got %v", err)
	}
}

func TestReadStats_PopulatesCorrectStats(t *testing.T) {
	t.Parallel()
	statsReader := strings.NewReader(`http://fake.url,20,19,1,100.123,150.000,198.465`)
	got, err := bench.ReadStats(statsReader)
	if err != nil {
		t.Fatal(err)
	}
	want := bench.Stats{
		P50:       100.123,
		P90:       150.000,
		P99:       198.465,
		Failures:  1,
		Requests:  20,
		Successes: 19,
		URL:       "http://fake.url",
	}

	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestReadStatsFile_PopulatesCorrectStatsFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := dir + "/stats.txt"
	err := bench.WriteStatsFile(path, bench.Stats{
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

func TestWriteStats_PopulatesCorrectStats(t *testing.T) {
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
	err := bench.WriteStats(output, stats)
	if err != nil {
		t.Fatal(err)
	}
	want := `http://fake.url,20,18,2,100.123,150.000,198.465`
	got := output.String()
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestCompareStateStringerPrintsExpectedMessage(t *testing.T) {
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

	err := bench.RunCLI([]string{})
	if !errors.Is(err, bench.ErrUnkownSubCommand) {
		t.Fatalf("want error bench.ErrUnkownSubCommand got %v", err)
	}
}

func TestRunCLI_ErrorsIfUnknownSubCommand(t *testing.T) {
	t.Parallel()

	err := bench.RunCLI([]string{"bogus"})
	if !errors.Is(err, bench.ErrUnkownSubCommand) {
		t.Fatalf("want error bench.ErrUnkownSubCommand got %v", err)
	}
}
