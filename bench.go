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
	// DefaultConcurrency sets the default number of simultaneous users
	DefaultConcurrency = 1
	// DefaultNumRequests sets the default number of requests to be performed
	DefaultNumRequests = 1
	// DefaultOutputPath sets the default output path for the graphs
	DefaultOutputPath = "./"
	// DefaultUserAgent sets the default user agent to be used in the HTTP calls
	DefaultUserAgent = "Bench 0.0.1 Alpha"
)

var (
	// DefaultHTTPClient instantiate the http.Client with 5 seconds timeout
	DefaultHTTPClient = &http.Client{
		Timeout: 5 * time.Second,
	}
	// ErrNoArgs is the error for when no arguments is passed via CLI
	ErrNoArgs = errors.New("no arguments")
	// ErrCMPNoArgs is the error for when no arguments is passed to the cmp subcommand
	ErrCMPNoArgs = errors.New("no stats file to compare. Please, specify two files")
	// ErrTimeNotRecorded is the error for when there is no execution time recorded
	ErrTimeNotRecorded = errors.New("no execution time recorded")
	// ErrValueCannotBeNil is the error for when the interfaces io.Writer or
	// io.Reader is nuil
	ErrValueCannotBeNil = errors.New("value cannot be nil")
	// ErrUnkownSubCommand is the error for when the subcommand is not known
	// (run or cmp)
	ErrUnkownSubCommand = errors.New("unknown subcommand. Please, specify run or cmp")
)

// Tester is the main struct where most information are stored
type Tester struct {
	body           string
	client         *http.Client
	concurrency    int
	endAt          time.Duration
	graphs         bool
	httpMethod     string
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

// NewTester creates a new Tester object, applies functional options and some
// simple checks on the data passed in, and returns a pointer to Tester and an error
func NewTester(opts ...Option) (*Tester, error) {
	tester := &Tester{
		client:      DefaultHTTPClient,
		concurrency: DefaultConcurrency,
		httpMethod:  http.MethodGet,
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
	u, err := url.Parse(tester.URL)
	if err != nil {
		return nil, err
	}
	if u.Host == "" {
		return nil, fmt.Errorf("invalid URL %q", tester.URL)
	}
	if tester.requests < 1 {
		return nil, fmt.Errorf("%d is invalid number of requests", tester.requests)
	}
	tester.work = make(chan struct{})
	return tester, nil
}

// FromArgs creates a new flagset and sets the values in the Tester struct
func FromArgs(args []string) Option {
	return func(t *Tester) error {
		fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
		fs.SetOutput(t.stderr)
		body := fs.String("b", "", "http body for the requests")
		concurrency := fs.Int("c", 1, "number of concurrent requests (users) to run benchmark")
		graphs := fs.Bool("g", false, "generate graphs")
		method := fs.String("m", "GET", "http method for the requests")
		reqs := fs.Int("r", 1, "number of requests to be performed in the benchmark")
		url := fs.String("u", "", "url to run benchmark")
		if len(args) < 1 {
			fs.Usage()
			return ErrNoArgs
		}
		err := fs.Parse(args)
		if err != nil {
			return err
		}
		t.body = *body
		t.concurrency = *concurrency
		t.graphs = *graphs
		t.httpMethod = strings.ToUpper(*method)
		t.requests = *reqs
		t.URL = *url
		return nil
	}
}

// WithRequests is the functional option to set the number of requests while
// initializing a new Tester object
func WithRequests(reqs int) Option {
	return func(t *Tester) error {
		t.requests = reqs
		return nil
	}
}

// WithHTTPUserAgent is the functional option to set the HTTP user agent while
// initializing a new Tester object
func WithHTTPUserAgent(userAgent string) Option {
	return func(t *Tester) error {
		t.userAgent = userAgent
		return nil
	}
}

// WithHTTPClient is the functional option to set a custom http.Client while
// initializing a new Tester object
func WithHTTPClient(client *http.Client) Option {
	return func(t *Tester) error {
		t.client = client
		return nil
	}
}

// WithHTTPMethod is the functional option to set a custom http method while
// initializing a new Tester object
func WithHTTPMethod(method string) Option {
	return func(t *Tester) error {
		t.httpMethod = strings.ToUpper(method)
		return nil
	}
}

// WithStdout is the functional option to set a custom io.Writer for stdout
// while initializing a new Tester object
func WithStdout(w io.Writer) Option {
	return func(t *Tester) error {
		if w == nil {
			return ErrValueCannotBeNil
		}
		t.stdout = w
		return nil
	}
}

// WithStderr is the functional option to set a custom io.Writer for stderr
// while initializing a new Tester object
func WithStderr(w io.Writer) Option {
	return func(lg *Tester) error {
		if w == nil {
			return ErrValueCannotBeNil
		}
		lg.stderr = w
		return nil
	}
}

// WithConcurrency is the functional option to set the number of simultaneous
// users while initializing a new Tester object
func WithConcurrency(c int) Option {
	return func(lg *Tester) error {
		lg.concurrency = c
		return nil
	}
}

// WithURL is the functional option to set the site URL while initializing a new
// Tester object
func WithURL(URL string) Option {
	return func(t *Tester) error {
		t.URL = URL
		return nil
	}
}

// WithOutputPath is the functional option to set the output path for the graphs
// while initializing a new Tester object
func WithOutputPath(outputPath string) Option {
	return func(t *Tester) error {
		t.outputPath = outputPath
		return nil
	}
}

// WithGraphs is the functional option to set whether graphs should be generated
// or not while initializing a new Tester object
func WithGraphs(graphs bool) Option {
	return func(t *Tester) error {
		t.graphs = graphs
		return nil
	}
}

// WithBody is the functional option to set the request body
func WithBody(body string) Option {
	return func(t *Tester) error {
		t.body = body
		return nil
	}
}

// Concurrency returns the value of simultaneous users
func (t Tester) Concurrency() int {
	return t.concurrency
}

// EndAt returns the value that benchmark is done
func (t Tester) EndAt() int64 {
	return t.endAt.Milliseconds()
}

// Graphs returns whether graphs should be generated or not
func (t Tester) Graphs() bool {
	return t.graphs
}

// HTTPUserAgent returns the current HTTP User Agent
func (t Tester) HTTPUserAgent() string {
	return t.userAgent
}

// HTTPClient returns the current http.Client
func (t Tester) HTTPClient() *http.Client {
	return t.client
}

// OutputPath returns the current path where files will be generated
func (t Tester) OutputPath() string {
	return t.outputPath
}

// Stats returns the current stats
func (t Tester) Stats() Stats {
	return t.stats
}

// Requests returns the current number of requests configured
func (t Tester) Requests() int {
	return t.requests
}

// HTTPMethod returns the current HTTP method configured
func (t Tester) HTTPMethod() string {
	return t.httpMethod
}

// Body returns the current HTTP request body
func (t Tester) Body() string {
	return t.body
}

// DoRequest perform the HTTP request, record the stats and success or failure
func (t *Tester) DoRequest() {
	for range t.work {
		t.RecordRequest()
		req, err := http.NewRequest(t.httpMethod, t.URL, strings.NewReader(t.body))
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
			t.LogStdErr(err.Error())
			t.RecordFailure()
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

// Run orchestrates the main program and go routines
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
	t.CalculatePercentiles()
	if t.Graphs() {
		err := t.Boxplot()
		if err != nil {
			return err
		}
		err = t.Histogram()
		if err != nil {
			return err
		}
	}
	fmt.Fprintln(t.stdout, t.stats)
	return nil
}

// Boxplot generates a boxplot graph
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

// Histogram generates a histogram graph
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

// RecordRequest uses mutex to increment one in the total requests
func (t *Tester) RecordRequest() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stats.Requests++
}

// RecordSuccess uses mutex to increment one in the total successes
func (t *Tester) RecordSuccess() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stats.Successes++
}

// RecordFailure uses mutex to increment one in the total failures
func (t *Tester) RecordFailure() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stats.Failures++
}

// LogStdOut is a wrapper to avoid Fprint to t.stdout in several places.
func (t Tester) LogStdOut(msg string) {
	fmt.Fprint(t.stdout, msg)
}

// LogStdErr is a wrapper to avoid Fprint to t.stderr in several places.
func (t Tester) LogStdErr(msg string) {
	fmt.Fprint(t.stderr, msg)
}

// LogFStdOut is a wrapper to avoid Fprintf to t.stdout in several places.
func (t Tester) LogFStdOut(msg string, opts ...interface{}) {
	fmt.Fprintf(t.stdout, msg, opts...)
}

// LogFStdErr is a wrapper to avoid Fprintf to t.stderr in several places.
func (t Tester) LogFStdErr(msg string, opts ...interface{}) {
	fmt.Fprintf(t.stderr, msg, opts...)
}

// CalculatePercentiles check if there is time recorded, calculates p50, p90 and
// p99 metrics plus the total time for all executions
func (t *Tester) CalculatePercentiles() {
	times := t.TimeRecorder.ExecutionsTime
	if len(times) < 1 {
		return
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
	t.stats.URL = t.URL
}

// Stats is the struct to store statistical information about the benchmark
type Stats struct {
	URL       string
	P50       float64
	P90       float64
	P99       float64
	Failures  int
	Requests  int
	Successes int
}

// String returns printable string of the stats
func (s Stats) String() string {
	return fmt.Sprintf(`Site: %s
Requests: %d
Successes: %d
Failures: %d
P50(ms): %.3f
P90(ms): %.3f
P99(ms): %.3f`, s.URL, s.Requests, s.Successes, s.Failures, s.P50, s.P90, s.P99,
	)
}

// TimeRecorder is the struct to store all execution times
type TimeRecorder struct {
	mu             *sync.Mutex
	ExecutionsTime []float64
}

// RecordTime uses mutex to add new execution time in the slice of execution times
func (t *TimeRecorder) RecordTime(executionTime float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ExecutionsTime = append(t.ExecutionsTime, executionTime)
}

// Option is a type for functional options
type Option func(*Tester) error

// ReadStatsFile is a wrapper to avoid user paperwork of opening the file
func ReadStatsFile(path string) (Stats, error) {
	f, err := os.Open(path)
	if err != nil {
		return Stats{}, err
	}
	defer f.Close()
	stats, err := ReadStats(f)
	// write a test for this case
	if err != nil {
		return Stats{}, fmt.Errorf("filename %q, err: %v", path, err)
	}
	return stats, nil
}

// ReadStats reads the stats of a given io.Reader and returns the stats and an error
func ReadStats(r io.Reader) (Stats, error) {
	scanner := bufio.NewScanner(r)
	stats := Stats{}
	for scanner.Scan() {
		text := scanner.Text()
		pos := strings.Split(text, " ")
		if len(pos) < 2 {
			continue
		}
		field := pos[0]
		value := pos[1]
		switch field {
		case "Site:":
			stats.URL = value
		case "Requests:":
			valueConv, err := strconv.Atoi(value)
			if err != nil {
				return Stats{}, err
			}
			stats.Requests = valueConv
		case "Successes:":
			valueConv, err := strconv.Atoi(value)
			if err != nil {
				return Stats{}, err
			}
			stats.Successes = valueConv
		case "Failures:":
			valueConv, err := strconv.Atoi(value)
			if err != nil {
				return Stats{}, err
			}
			stats.Failures = valueConv
		case "P50(ms):":
			valueConv, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return Stats{}, err
			}
			stats.P50 = valueConv
		case "P90(ms):":
			valueConv, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return Stats{}, err
			}
			stats.P90 = valueConv
		case "P99(ms):":
			valueConv, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return Stats{}, err
			}
			stats.P99 = valueConv
		default:
			return Stats{}, fmt.Errorf("unknown statsfile format. Invalid line %q", text)
		}

	}
	if err := scanner.Err(); err != nil {
		return Stats{}, err
	}
	return stats, nil
}

// CompareStats stores two stats to compare them
type CompareStats struct {
	S1, S2 Stats
}

// String returns a printable string from comparison of two stats.
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

// RunCLI is the main entrypoint for the CLI
func RunCLI(w io.Writer, args []string) error {
	if len(args) < 1 {
		return ErrUnkownSubCommand
	}
	switch args[0] {
	case "run":
		tester, err := NewTester(
			FromArgs(args[1:]),
			WithStdout(w),
		)
		if err != nil {
			return err
		}
		err = tester.Run()
		if err != nil {
			return err
		}
	case "cmp":
		err := CMPRun(w, args[1:])
		if err != nil {
			return err
		}
	default:
		return ErrUnkownSubCommand
	}
	return nil
}

// CMPRun is the entrypoint for the subcommand cmp
func CMPRun(w io.Writer, args []string) error {
	if len(args) < 2 {
		return ErrCMPNoArgs
	}
	s1, err := ReadStatsFile(args[0])
	if err != nil {
		return err
	}
	s2, err := ReadStatsFile(args[1])
	if err != nil {
		return err
	}
	fmt.Fprint(w, CompareStats{
		S1: s1,
		S2: s2,
	})
	return nil
}
