package bench

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type Tester struct {
	client         *http.Client
	requests       int
	startAt        time.Time
	stats          Stats
	userAgent      string
	url            string
	stdout, stderr io.Writer
	wg             *sync.WaitGroup
	work           chan string
	TimeRecorder   TimeRecorder
}

func NewTester(URL string, opts ...Option) (*Tester, error) {
	u, err := url.Parse(URL)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid URL %s", u)
	}
	tester := &Tester{
		client:    &http.Client{Timeout: 30 * time.Second},
		requests:  1,
		userAgent: "Bench 0.0.1 Alpha",
		url:       URL,
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
		o(tester)
	}
	if tester.requests == 0 {
		return nil, fmt.Errorf("%d is invalid number of requests", tester.requests)
	}
	tester.work = make(chan string, tester.requests)
	return tester, nil
}

func WithRequests(reqs int) Option {
	return func(lg *Tester) {
		lg.requests = reqs
	}
}

func WithHTTPUserAgent(userAgent string) Option {
	return func(lg *Tester) {
		lg.userAgent = userAgent
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(lg *Tester) {
		lg.client = client
	}
}

func WithStdout(w io.Writer) Option {
	return func(lg *Tester) {
		lg.stdout = w
	}
}

func WithStderr(w io.Writer) Option {
	return func(lg *Tester) {
		lg.stderr = w
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
	startTime := Time()
	resp, err := t.client.Do(req)
	endTime := Time()
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
	t.wg.Add(t.requests)
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
		t.work <- t.url
	}
	t.wg.Wait()
	t.SetFastestAndSlowest()
	t.LogFStdOut("URL %q benchmark is done\n", t.url)
	t.LogFStdOut("Time: %v Requests: %d Success: %d Failures: %d\n", time.Since(t.startAt), t.stats.Requests, t.stats.Success, t.stats.Failures)
	t.LogFStdOut("Fastest: %v Slowest: %v\n", t.stats.Fastest, t.stats.Slowest)
}

func (t *Tester) RecordRequest() {
	atomic.AddUint64(&t.stats.Requests, 1)
}

func (t *Tester) RecordSuccess() {
	atomic.AddUint64(&t.stats.Success, 1)
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

func (t *Tester) SetFastestAndSlowest() {
	t.stats.Fastest = t.TimeRecorder.ExecutionsTime[0]
	t.stats.Slowest = t.TimeRecorder.ExecutionsTime[0]
	for _, v := range t.TimeRecorder.ExecutionsTime {
		if v < t.stats.Fastest {
			t.stats.Fastest = v
		} else if v > t.stats.Slowest {
			t.stats.Slowest = v
		}
	}
}

type Stats struct {
	Requests, Success, Failures uint64
	Slowest, Fastest            time.Duration
}

type Option func(*Tester)

var Time = func() time.Time {
	return time.Now()
}

type TimeRecorder struct {
	MU             *sync.Mutex
	ExecutionsTime []time.Duration
}

func (t *TimeRecorder) RecordTime(executionTime time.Duration) {
	t.MU.Lock()
	defer t.MU.Unlock()
	t.ExecutionsTime = append(t.ExecutionsTime, executionTime)
}
