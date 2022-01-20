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
}

type Stats struct {
	Requests, Success, Failures uint64
	Slowest                     time.Duration
	MU                          *sync.Mutex
	ExecutionsTime              []time.Duration
}

type Option func(*Tester)

func NewTester(URL string, opts ...Option) (*Tester, error) {
	u, err := url.Parse(URL)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf(fmt.Sprintf("Invalid URL %s", u))
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
		stats: Stats{
			Requests:       0,
			Success:        0,
			Failures:       0,
			Slowest:        0,
			MU:             &sync.Mutex{},
			ExecutionsTime: []time.Duration{},
		},
	}
	for _, o := range opts {
		o(tester)
	}
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
	return lg.stats
}

func (lg Tester) GetRequests() int {
	return lg.requests
}

func (lg *Tester) AddToWG(count int) {
	lg.wg.Add(count)
}

func (lg *Tester) WaitForWG() {
	lg.wg.Wait()
}

func (lg *Tester) DoRequest(url string) {
	defer lg.wg.Done()
	lg.RecordRequest()
	req, err := http.NewRequest(http.MethodGet, lg.url, nil)
	if err != nil {
		lg.LogStdErr(err.Error())
		lg.RecordFailure()
		return
	}
	req.Header.Set("user-agent", lg.GetHTTPUserAgent())
	req.Header.Set("accept", "*/*")
	startTime := time.Now()
	resp, err := lg.client.Do(req)
	elapsedTime := time.Since(startTime)
	lg.RecordTime(elapsedTime)
	if err != nil {
		lg.LogStdErr(err.Error())
		lg.RecordFailure()
		return
	}
	if resp.StatusCode != http.StatusOK {
		lg.LogStdErr(fmt.Sprintf("unexpected status code %d\n", resp.StatusCode))
		lg.RecordFailure()
		return
	}
	lg.RecordSuccess()
}

func (lg *Tester) Run() {
	bencher := func() <-chan string {
		work := make(chan string, lg.requests)
		go func() {
			defer close(work)
			for x := 0; x < lg.requests; x++ {
				work <- lg.url
			}
		}()
		return work
	}

	work := bencher()
	for url := range work {
		lg.AddToWG(1)
		go lg.DoRequest(url)
	}
	lg.WaitForWG()
	lg.LogStdOut(fmt.Sprintf("URL %q benchmark is done\n", lg.url))
	lg.LogStdOut(fmt.Sprintf("Time: %v Requests: %d Success: %d Failures: %d\n", time.Since(lg.startAt), lg.stats.Requests, lg.stats.Success, lg.stats.Failures))
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

func (lg *Tester) RecordTime(executionTime time.Duration) {
	lg.stats.MU.Lock()
	defer lg.stats.MU.Unlock()
	lg.stats.ExecutionsTime = append(lg.stats.ExecutionsTime, executionTime)
}

func (lg Tester) LogStdOut(msg string) {
	fmt.Fprint(lg.stdout, msg)
}

func (lg Tester) LogStdErr(msg string) {
	fmt.Fprint(lg.stderr, msg)
}
