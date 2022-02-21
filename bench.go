package bench

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DefaultNumRequests = 1
	DefaultUserAgent   = "Bench 0.0.1 Alpha"
)

var (
	ErrNoURL           = errors.New("no URL to test")
	ErrTimeNotRecorded = errors.New("no execution time recorded")
	DefaultHTTPClient  = &http.Client{
		Timeout: 30 * time.Second,
	}
)

type Tester struct {
	client         *http.Client
	requests       int
	startAt        time.Time
	stats          Stats
	userAgent      string
	URL            []string
	stdout, stderr io.Writer
	wg             *sync.WaitGroup
	work           chan string
	TimeRecorder   TimeRecorder
}

func NewTester(opts ...Option) (*Tester, error) {
	tester := &Tester{
		client:    &http.Client{Timeout: 30 * time.Second},
		requests:  DefaultNumRequests,
		userAgent: DefaultUserAgent,
		startAt:   time.Now(),
		stdout:    os.Stdout,
		stderr:    os.Stderr,
		wg:        &sync.WaitGroup{},
		stats:     Stats{},
		TimeRecorder: TimeRecorder{
			MU:             &sync.Mutex{},
			ExecutionsTime: []time.Duration{},
		},
	}
	for _, o := range opts {
		err := o(tester)
		if err != nil {
			return nil, err
		}
	}
	if len(tester.URL) < 1 {
		return nil, ErrNoURL
	}
	for _, URL := range tester.URL {
		u, err := url.Parse(URL)
		if err != nil {
			return nil, err
		}
		if u.Scheme == "" || u.Host == "" {
			return nil, fmt.Errorf("invalid URL %q", u)
		}
	}

	if tester.requests < 1 {
		return nil, fmt.Errorf("%d is invalid number of requests", tester.requests)
	}
	tester.work = make(chan string, tester.requests)
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

func WithInputsFromArgs(args []string) Option {
	return func(t *Tester) error {
		fset := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
		fset.SetOutput(t.stderr)
		reqs := fset.Int("r", 1, "number of requests to be performed in the benchmark")
		err := fset.Parse(args)
		if err != nil {
			return err
		}
		args = fset.Args()
		if len(args) < 1 {
			fset.Usage()
			return ErrNoURL
		}
		t.URL = args
		t.requests = *reqs
		return nil
	}
}

func WithURL(URL []string) Option {
	return func(t *Tester) error {
		t.URL = URL
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

func (t *Tester) DoRequest(url string) {
	t.RecordRequest()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.LogStdErr(err.Error())
		t.RecordFailure()
		return
	}
	req.Header.Set("user-agent", t.HTTPUserAgent())
	req.Header.Set("accept", "*/*")
	startTime := time.Now()
	resp, err := t.client.Do(req)
	endTime := time.Now()
	if err != nil {
		t.RecordFailure()
		t.LogStdErr(err.Error())
		return
	}
	elapsedTime := endTime.Sub(startTime)
	t.TimeRecorder.RecordTime(elapsedTime)
	if resp.StatusCode != http.StatusOK {
		t.LogFStdErr("unexpected status code %d\n", resp.StatusCode)
		t.RecordFailure()
		return
	}
	t.RecordSuccess()
}

func (t *Tester) Run() {
	t.wg.Add(t.requests * len(t.URL))
	go func() {
		for range time.NewTicker(time.Millisecond).C {
			url := <-t.work
			go func() {
				t.DoRequest(url)
				t.wg.Done()
			}()
		}
	}()
	for x := 0; x < t.requests; x++ {
		for _, v := range t.URL {
			t.work <- v
		}
	}
	t.wg.Wait()
	t.SetMetrics()
	t.LogFStdOut("URL: %q benchmark is done\n", t.URL)
	t.LogFStdOut("Time: %v Requests: %d Success: %d Failures: %d\n", time.Since(t.startAt), t.stats.Requests, t.stats.Successes, t.stats.Failures)
	t.LogFStdOut("Fastest: %v Mean: %v Slowest: %v\n", t.stats.Fastest, t.stats.Mean, t.stats.Slowest)
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
	if len(t.TimeRecorder.ExecutionsTime) < 1 {
		return ErrTimeNotRecorded
	}
	nreq := 0
	total := 0 * time.Millisecond
	t.stats.Fastest = t.TimeRecorder.ExecutionsTime[0]
	t.stats.Slowest = t.TimeRecorder.ExecutionsTime[0]
	for _, v := range t.TimeRecorder.ExecutionsTime {
		nreq++
		total += v
		if v < t.stats.Fastest {
			t.stats.Fastest = v
		} else if v > t.stats.Slowest {
			t.stats.Slowest = v
		}
	}
	t.stats.Mean = total / time.Duration(nreq)
	return nil
}

type Stats struct {
	Requests, Successes, Failures uint64
	Slowest, Fastest, Mean        time.Duration
}

type Option func(*Tester) error

type TimeRecorder struct {
	MU             *sync.Mutex
	ExecutionsTime []time.Duration
}

func (t *TimeRecorder) RecordTime(executionTime time.Duration) {
	t.MU.Lock()
	defer t.MU.Unlock()
	t.ExecutionsTime = append(t.ExecutionsTime, executionTime)
}
