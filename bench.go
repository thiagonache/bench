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

func (lg Tester) GetHTTPUserAgent() string {
	return lg.userAgent
}

func (lg Tester) GetHTTPClient() *http.Client {
	return lg.client
}

func (lg Tester) GetStartTime() time.Time {
	return lg.startAt
}

func (lg Tester) GetStats() Stats {
	lg.stats.Fastest = lg.TimeRecorder.ExecutionsTime[0]
	lg.stats.Slowest = lg.TimeRecorder.ExecutionsTime[0]
	for _, v := range lg.TimeRecorder.ExecutionsTime {
		if v < lg.stats.Fastest {
			lg.stats.Fastest = v
		} else if v > lg.stats.Slowest {
			lg.stats.Slowest = v
		}
	}
	return lg.stats
}

func (lg Tester) GetRequests() int {
	return lg.requests
}

func (lg *Tester) DoRequest(url string) {
	lg.RecordRequest()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		lg.LogStdErr(err.Error())
		lg.RecordFailure()
		return
	}
	req.Header.Set("user-agent", lg.GetHTTPUserAgent())
	req.Header.Set("accept", "*/*")
	startTime := Time()
	resp, err := lg.client.Do(req)
	endTime := Time()
	if err != nil {
		lg.RecordFailure()
		lg.LogStdErr(err.Error())
		return
	}
	elapsedTime := endTime.Sub(startTime)
	lg.TimeRecorder.RecordTime(elapsedTime)
	if resp.StatusCode != http.StatusOK {
		lg.LogStdErr(fmt.Sprintf("unexpected status code %d\n", resp.StatusCode))
		lg.RecordFailure()
		return
	}
	lg.RecordSuccess()
}

func (lg *Tester) Run() {
	lg.wg.Add(lg.requests)
	go func() {
		for range time.NewTicker(time.Millisecond).C {
			url := <-lg.work
			go func() {
				lg.DoRequest(url)
				lg.wg.Done()
			}()
		}
	}()
	for x := 0; x < lg.requests; x++ {
		lg.work <- lg.url
	}
	lg.wg.Wait()
	stats := lg.GetStats()
	lg.LogStdOut(fmt.Sprintf("URL %q benchmark is done\n", lg.url))
	lg.LogStdOut(fmt.Sprintf("Time: %v Requests: %d Success: %d Failures: %d\n", time.Since(lg.startAt), lg.stats.Requests, lg.stats.Success, lg.stats.Failures))
	lg.LogStdOut(fmt.Sprintf("Fastest: %v Slowest: %v\n", stats.Fastest, stats.Slowest))
}

func (lg *Tester) RecordRequest() {
	atomic.AddUint64(&lg.stats.Requests, 1)
}

func (lg *Tester) RecordSuccess() {
	atomic.AddUint64(&lg.stats.Success, 1)
}

func (lg *Tester) RecordFailure() {
	atomic.AddUint64(&lg.stats.Failures, 1)
}

func (lg Tester) LogStdOut(msg string) {
	fmt.Fprint(lg.stdout, msg, "\n")
}

func (lg Tester) LogStdErr(msg string) {
	fmt.Fprint(lg.stderr, msg, "\n")
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
