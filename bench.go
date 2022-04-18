package bench

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

const (
	DefaultConcurrency = 1
	DefaultNumRequests = 1
	DefaultOutputPath  = "./"
	DefaultUserAgent   = "Bench 0.0.1 Alpha"
)

var (
	DefaultHTTPClient = &http.Client{
		Timeout: 5 * time.Second,
	}
	ErrNoArgs           = errors.New("no arguments")
	ErrCMPNoArgs        = errors.New("no stats file to compare. Please, specify two files")
	ErrNoURL            = errors.New("no URL to test")
	ErrTimeNotRecorded  = errors.New("no execution time recorded")
	ErrValueCannotBeNil = errors.New("value cannot be nil")
	ErrUnkownSubCommand = errors.New("unknown subcommand. Please, specify run or cmp")
)

type Tester struct {
	concurrency    int
	client         *http.Client
	endAt          time.Duration
	exportStats    bool
	graphs         bool
	outputPath     string
	requests       int
	startAt        time.Time
	stdout, stderr io.Writer
	URL            string
	userAgent      string
	wg             *sync.WaitGroup
	work           chan struct{}

	mu           *sync.Mutex
	stats        Stats
	TimeRecorder TimeRecorder
}

func NewTester(opts ...Option) (*Tester, error) {
	tester := &Tester{
		client:      DefaultHTTPClient,
		concurrency: DefaultConcurrency,
		outputPath:  DefaultOutputPath,
		requests:    DefaultNumRequests,
		stats:       Stats{},
		stderr:      os.Stderr,
		stdout:      os.Stdout,
		TimeRecorder: TimeRecorder{
			ExecutionsTime: []float64{},
			mu:             &sync.Mutex{},
		},
		userAgent: DefaultUserAgent,
		wg:        &sync.WaitGroup{},
		mu:        &sync.Mutex{},
	}
	for _, o := range opts {
		err := o(tester)
		if err != nil {
			return nil, err
		}
	}
	if tester.URL == "" {
		return nil, ErrNoURL
	}
	u, err := url.Parse(tester.URL)
	if err != nil {
		return nil, err
	}
	if u.Host == "" {
		return nil, fmt.Errorf("invalid URL %q", u)
	}
	if tester.requests < 1 {
		return nil, fmt.Errorf("%d is invalid number of requests", tester.requests)
	}
	tester.work = make(chan struct{})
	return tester, nil
}

func FromArgs(args []string) Option {
	return func(t *Tester) error {
		fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
		fs.SetOutput(t.stderr)
		reqs := fs.Int("r", 1, "number of requests to be performed in the benchmark")
		graphs := fs.Bool("g", false, "generate graphs")
		exportStats := fs.Bool("s", false, "generate stats file")
		concurrency := fs.Int("c", 1, "number of concurrent requests (users) to run benchmark")
		url := fs.String("u", "", "url to run benchmark")
		if len(args) < 1 {
			fs.Usage()
			return ErrNoArgs
		}
		fs.Parse(args)
		t.URL = *url
		t.requests = *reqs
		t.graphs = *graphs
		t.concurrency = *concurrency
		t.exportStats = *exportStats
		return nil
	}
}

func WithRequests(reqs int) Option {
	return func(t *Tester) error {
		t.requests = reqs
		return nil
	}
}

func WithHTTPUserAgent(userAgent string) Option {
	return func(t *Tester) error {
		t.userAgent = userAgent
		return nil
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(t *Tester) error {
		t.client = client
		return nil
	}
}

func WithStdout(w io.Writer) Option {
	return func(t *Tester) error {
		if w == nil {
			return ErrValueCannotBeNil
		}
		t.stdout = w
		return nil
	}
}

func WithStderr(w io.Writer) Option {
	return func(lg *Tester) error {
		if w == nil {
			return ErrValueCannotBeNil
		}
		lg.stderr = w
		return nil
	}
}

func WithConcurrency(c int) Option {
	return func(lg *Tester) error {
		lg.concurrency = c
		return nil
	}
}

func WithURL(URL string) Option {
	return func(t *Tester) error {
		t.URL = URL
		return nil
	}
}

func WithOutputPath(outputPath string) Option {
	return func(t *Tester) error {
		t.outputPath = outputPath
		return nil
	}
}

func WithGraphs(graphs bool) Option {
	return func(t *Tester) error {
		t.graphs = graphs
		return nil
	}
}

func WithExportStats(exportStats bool) Option {
	return func(t *Tester) error {
		t.exportStats = exportStats
		return nil
	}
}

func (t Tester) Concurrency() int {
	return t.concurrency
}

func (t Tester) EndAt() int64 {
	return t.endAt.Milliseconds()
}

func (t Tester) ExportStats() bool {
	return t.exportStats
}

func (t Tester) Graphs() bool {
	return t.graphs
}

func (t Tester) HTTPUserAgent() string {
	return t.userAgent
}

func (t Tester) HTTPClient() *http.Client {
	return t.client
}

func (t Tester) OutputPath() string {
	return t.outputPath
}

func (t Tester) StartTime() time.Time {
	return t.startAt
}

func (t Tester) Stats() Stats {
	return t.stats
}

func (t Tester) Requests() int {
	return t.requests
}

func (t *Tester) DoRequest() {
	for range t.work {
		t.RecordRequest()
		req, err := http.NewRequest(http.MethodGet, t.URL, nil)
		if err != nil {
			t.LogStdErr(err.Error())
			t.RecordFailure()
			return
		}
		req.Header.Set("user-agent", t.HTTPUserAgent())
		req.Header.Set("accept", "*/*")
		startTime := time.Now()
		resp, err := t.client.Do(req)
		elapsedTime := time.Since(startTime)
		if err != nil {
			t.RecordFailure()
			t.LogStdErr(err.Error())
			return
		}
		t.TimeRecorder.RecordTime(float64(elapsedTime.Nanoseconds()) / 1000000.0)
		if resp.StatusCode != http.StatusOK {
			t.LogFStdErr("unexpected status code %d\n", resp.StatusCode)
			t.RecordFailure()
			return
		}
		t.RecordSuccess()
	}
}

func (t *Tester) Run() error {
	t.wg.Add(t.Concurrency())
	go func() {
		for x := 0; x < t.Requests(); x++ {
			t.work <- struct{}{}
		}
		close(t.work)
	}()
	t.startAt = time.Now()
	go func() {
		for x := 0; x < t.Concurrency(); x++ {
			go func() {
				t.DoRequest()
				t.wg.Done()
			}()
		}
	}()
	t.wg.Wait()
	t.endAt = time.Since(t.startAt)
	err := t.SetMetrics()
	if err != nil {
		return err
	}
	if t.Graphs() {
		err = t.Boxplot()
		if err != nil {
			return err
		}
		err = t.Histogram()
		if err != nil {
			return err
		}
	}
	if t.ExportStats() {
		file, err := os.Create(fmt.Sprintf("%s/%s", t.OutputPath(), "statsfile.txt"))
		if err != nil {
			return err
		}
		defer file.Close()
		err = WriteStats(file, t.Stats())
		if err != nil {
			return err
		}
	}
	t.LogFStdOut("The benchmark of %s URL took %dms\n", t.URL, t.EndAt())
	t.LogFStdOut("Requests: %d Success: %d Failures: %d\n", t.stats.Requests, t.stats.Successes, t.stats.Failures)
	t.LogFStdOut("P50: %.3fms P90: %.3fms P99: %.3fms\n", t.stats.P50, t.stats.P90, t.stats.P99)
	return nil
}

func (t Tester) Boxplot() error {
	p := plot.New()
	p.Title.Text = "Latency boxplot"
	p.Y.Label.Text = "latency (ms)"
	p.X.Label.Text = t.URL
	w := vg.Points(20)
	box, err := plotter.NewBoxPlot(w, 0, plotter.Values(t.TimeRecorder.ExecutionsTime))
	if err != nil {
		return err
	}
	p.Add(box)
	err = p.Save(600, 400, fmt.Sprintf("%s/%s", t.OutputPath(), "boxplot.png"))
	if err != nil {
		return err
	}
	return nil
}

func (t Tester) Histogram() error {
	p := plot.New()
	p.Title.Text = "Latency Histogram"
	p.Y.Label.Text = "n reqs"
	p.X.Label.Text = "latency (ms)"
	hist, err := plotter.NewHist(plotter.Values(t.TimeRecorder.ExecutionsTime), 50)
	if err != nil {
		return err
	}
	p.Add(hist)
	err = p.Save(600, 400, fmt.Sprintf("%s/%s", t.OutputPath(), "histogram.png"))
	if err != nil {
		return err
	}
	return nil
}

func (t *Tester) RecordRequest() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stats.Requests++
}

func (t *Tester) RecordSuccess() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stats.Successes++
}

func (t *Tester) RecordFailure() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stats.Failures++
}

func (t Tester) LogStdOut(msg string) {
	fmt.Fprint(t.stdout, msg)
}

func (t Tester) LogStdErr(msg string) {
	fmt.Fprint(t.stderr, msg)
}

func (t Tester) LogFStdOut(msg string, opts ...interface{}) {
	fmt.Fprintf(t.stdout, msg, opts...)
}

func (t Tester) LogFStdErr(msg string, opts ...interface{}) {
	fmt.Fprintf(t.stderr, msg, opts...)
}

func (t *Tester) SetMetrics() error {
	times := t.TimeRecorder.ExecutionsTime
	if len(times) < 1 {
		return ErrTimeNotRecorded
	}
	sort.Slice(times, func(i, j int) bool {
		return times[i] < times[j]
	})
	p50Idx := int(math.Round(float64(len(times))*0.5)) - 1
	t.stats.P50 = times[p50Idx]
	p90Idx := int(math.Round(float64(len(times))*0.9)) - 1
	t.stats.P90 = times[p90Idx]
	p99Idx := int(math.Round(float64(len(times))*0.99)) - 1
	t.stats.P99 = times[p99Idx]

	nreq := 0.0
	totalTime := 0.0
	for _, v := range times {
		nreq++
		totalTime += v
	}
	t.stats.URL = t.URL
	t.stats.Mean = totalTime / nreq
	return nil
}

type Stats struct {
	URL       string
	Mean      float64
	P50       float64
	P90       float64
	P99       float64
	Failures  int
	Requests  int
	Successes int
}

type TimeRecorder struct {
	mu             *sync.Mutex
	ExecutionsTime []float64
}

func (t *TimeRecorder) RecordTime(executionTime float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ExecutionsTime = append(t.ExecutionsTime, executionTime)
}

type Option func(*Tester) error

func ReadStatsFiles(path1, path2 string) (CompareStats, error) {
	f1, err := os.Open(path1)
	if err != nil {
		return CompareStats{}, err
	}
	defer f1.Close()
	s1, err := ReadStatsFile(path1)
	if err != nil {
		return CompareStats{}, err
	}
	f2, err := os.Open(path2)
	if err != nil {
		return CompareStats{}, err
	}
	defer f2.Close()
	s2, err := ReadStats(f2)
	if err != nil {
		return CompareStats{}, err
	}
	return CompareStats{S1: s1, S2: s2}, nil
}

func ReadStatsFile(path string) (Stats, error) {
	f, err := os.Open(path)
	if err != nil {
		return Stats{}, err
	}
	defer f.Close()
	stats, err := ReadStats(f)
	if err != nil {
		return Stats{}, err
	}
	return stats, nil
}

func ReadStats(r io.Reader) (Stats, error) {
	scanner := bufio.NewScanner(r)
	stats := Stats{}
	for scanner.Scan() {
		pos := strings.Split(scanner.Text(), ",")
		url := pos[0]
		dataRequests := pos[1]
		requests, err := strconv.Atoi(dataRequests)
		if err != nil {
			return Stats{}, err
		}
		dataSuccesses := pos[2]
		successes, err := strconv.Atoi(dataSuccesses)
		if err != nil {
			return Stats{}, err
		}
		dataFailures := pos[3]
		failures, err := strconv.Atoi(dataFailures)
		if err != nil {
			return Stats{}, err
		}
		dataP50 := pos[4]
		p50, err := strconv.ParseFloat(dataP50, 64)
		if err != nil {
			return Stats{}, err
		}
		dataP90 := pos[5]
		p90, err := strconv.ParseFloat(dataP90, 64)
		if err != nil {
			return Stats{}, err
		}
		dataP99 := pos[6]
		p99, err := strconv.ParseFloat(dataP99, 64)
		if err != nil {
			return Stats{}, err
		}
		stats = Stats{
			Failures:  failures,
			P50:       p50,
			P90:       p90,
			P99:       p99,
			Requests:  requests,
			Successes: successes,
			URL:       url,
		}
	}
	if err := scanner.Err(); err != nil {
		return Stats{}, err
	}
	return stats, nil
}

func WriteStats(w io.Writer, stats Stats) error {
	_, err := fmt.Fprintf(w, "%s,%d,%d,%d,%.3f,%.3f,%.3f",
		stats.URL, stats.Requests, stats.Successes, stats.Failures, stats.P50, stats.P90, stats.P99,
	)
	if err != nil {
		return err
	}
	return nil
}

func WriteStatsFile(path string, stats Stats) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	err = WriteStats(f, stats)
	if err != nil {
		return err
	}
	return nil
}

type CompareStats struct {
	S1, S2 Stats
}

func (cs CompareStats) String() string {
	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, "Site %s\n", cs.S1.URL)
	writer := tabwriter.NewWriter(buf, 20, 0, 0, ' ', 0)
	fmt.Fprintln(writer, "Metric\tOld\tNew\tDelta\tPercentage")
	p50Delta := cs.S2.P50 - cs.S1.P50
	fmt.Fprintf(writer, "P50(ms)\t%.3f\t%.3f\t%.3f\t%.2f\n", cs.S1.P50, cs.S2.P50, p50Delta, p50Delta/cs.S1.P50*100)
	p90Delta := cs.S2.P90 - cs.S1.P90
	fmt.Fprintf(writer, "P90(ms)\t%.3f\t%.3f\t%.3f\t%.2f\n", cs.S1.P90, cs.S2.P90, p90Delta, p90Delta/cs.S1.P90*100)
	p99Delta := cs.S2.P99 - cs.S1.P99
	fmt.Fprintf(writer, "P99(ms)\t%.3f\t%.3f\t%.3f\t%.2f\n", cs.S1.P99, cs.S2.P99, p99Delta, p99Delta/cs.S1.P99*100)
	writer.Flush()
	return buf.String()
}

func RunCLI(args []string) error {
	if len(args) < 1 {
		return ErrUnkownSubCommand
	}
	switch args[0] {
	case "run":
		tester, err := NewTester(
			FromArgs(os.Args[2:]),
		)
		if err != nil {
			return err
		}
		tester.Run()
	case "cmp":
		err := CMPRun(args[1:])
		if err != nil {
			return err
		}
	default:
		return ErrUnkownSubCommand
	}
	return nil
}

func CMPRun(args []string) error {
	if len(args) < 2 {
		return ErrCMPNoArgs
	}
	cmpStats, err := ReadStatsFiles(args[0], args[1])
	if err != nil {
		return err
	}
	fmt.Println(cmpStats)
	return nil
}
