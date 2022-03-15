package bench

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"sync"
	"sync/atomic"
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
		Timeout: 30 * time.Second,
	}
	ErrNoURL           = errors.New("no URL to test")
	ErrTimeNotRecorded = errors.New("no execution time recorded")
)

type Tester struct {
	Concurrency    int
	client         *http.Client
	Graphs         bool
	OutputPath     string
	requests       int
	startAt        time.Time
	stats          Stats
	stdout, stderr io.Writer
	TimeRecorder   TimeRecorder
	URL            string
	userAgent      string
	wg             *sync.WaitGroup
	work           chan struct{}
}

func NewTester(opts ...Option) (*Tester, error) {
	tester := &Tester{
		client:      &http.Client{Timeout: 30 * time.Second},
		Concurrency: DefaultConcurrency,
		OutputPath:  DefaultOutputPath,
		requests:    DefaultNumRequests,
		startAt:     time.Now(),
		stats:       Stats{},
		stderr:      os.Stderr,
		stdout:      os.Stdout,
		TimeRecorder: TimeRecorder{
			ExecutionsTime: []float64{},
			MU:             &sync.Mutex{},
		},
		userAgent: DefaultUserAgent,
		wg:        &sync.WaitGroup{},
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
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid URL %q", u)
	}
	if tester.requests < 1 {
		return nil, fmt.Errorf("%d is invalid number of requests", tester.requests)
	}
	tester.work = make(chan struct{})
	return tester, nil
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
		t.stdout = w
		return nil
	}
}

func WithStderr(w io.Writer) Option {
	return func(lg *Tester) error {
		lg.stderr = w
		return nil
	}
}

func WithConcurrency(c int) Option {
	return func(lg *Tester) error {
		lg.Concurrency = c
		return nil
	}
}

func FromArgs(args []string) Option {
	return func(t *Tester) error {
		fset := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
		fset.SetOutput(t.stderr)
		reqs := fset.Int("r", 1, "number of requests to be performed in the benchmark")
		graphs := fset.Bool("g", false, "generate graphs")
		err := fset.Parse(args)
		if err != nil {
			return err
		}
		args = fset.Args()
		if len(args) < 1 {
			fset.Usage()
			return ErrNoURL
		}
		t.URL = args[0]
		t.requests = *reqs
		t.Graphs = *graphs
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
		t.OutputPath = outputPath
		return nil
	}
}

func (t Tester) HTTPUserAgent() string {
	return t.userAgent
}

func (t Tester) HTTPClient() *http.Client {
	return t.client
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
	t.wg.Add(t.Concurrency)
	go func() {
		for x := 0; x < t.requests; x++ {
			t.work <- struct{}{}
		}
		close(t.work)
	}()
	go func() {
		for x := 0; x < t.Concurrency; x++ {
			go func() {
				t.DoRequest()
				t.wg.Done()
			}()
		}
	}()
	t.wg.Wait()
	err := t.SetMetrics()
	if err != nil {
		return err
	}
	if t.Graphs {
		err = t.Boxplot()
		if err != nil {
			return err
		}
		err = t.Histogram()
		if err != nil {
			return err
		}
	}
	t.LogFStdOut("URL: %q benchmark is done\n", t.URL)
	t.LogFStdOut("Time: %v Requests: %d Success: %d Failures: %d\n", time.Since(t.startAt), t.stats.Requests, t.stats.Successes, t.stats.Failures)
	t.LogFStdOut("90th percentile: %v 99th percentile: %v\n", t.stats.Perc90, t.stats.Perc99)
	t.LogFStdOut("Fastest: %v Mean: %v Slowest: %v\n", t.stats.Fastest, t.stats.Mean, t.stats.Slowest)
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
	err = p.Save(600, 400, fmt.Sprintf("%s/%s", t.OutputPath, "boxplot.png"))
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
	err = p.Save(600, 400, fmt.Sprintf("%s/%s", t.OutputPath, "histogram.png"))
	if err != nil {
		return err
	}
	return nil
}

func (t *Tester) RecordRequest() {
	atomic.AddUint64(&t.stats.Requests, 1)
}

func (t *Tester) RecordSuccess() {
	atomic.AddUint64(&t.stats.Successes, 1)
}

func (t *Tester) RecordFailure() {
	atomic.AddUint64(&t.stats.Failures, 1)
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
	perc90Index := int(math.Round(float64(len(times))*0.9)) - 1
	t.stats.Perc90 = times[perc90Index]
	perc99Index := int(math.Round(float64(len(times))*0.99)) - 1
	t.stats.Perc99 = times[perc99Index]

	nreq := 0.0
	totalTime := 0.0
	t.stats.Fastest = times[0]
	t.stats.Slowest = times[0]
	for _, v := range times {
		nreq++
		totalTime += v
		if v < t.stats.Fastest {
			t.stats.Fastest = v
			continue
		}
		if v > t.stats.Slowest {
			t.stats.Slowest = v
		}
	}
	t.stats.Mean = totalTime / nreq
	return nil
}

type Stats struct {
	Failures  uint64
	Fastest   float64
	Mean      float64
	Perc90    float64
	Perc99    float64
	Requests  uint64
	Slowest   float64
	Successes uint64
}

type Option func(*Tester) error

type TimeRecorder struct {
	ExecutionsTime []float64
	MU             *sync.Mutex
}

func (t *TimeRecorder) RecordTime(executionTime float64) {
	t.MU.Lock()
	defer t.MU.Unlock()
	t.ExecutionsTime = append(t.ExecutionsTime, executionTime)
}
