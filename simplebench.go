package simplebench

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

type LoadGen struct {
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
}

type Option func(*LoadGen)

func NewLoadGen(URL string, opts ...Option) (*LoadGen, error) {
	u, err := url.Parse(URL)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf(fmt.Sprintf("Invalid URL %s", u))
	}
	loadgen := &LoadGen{
		client:    &http.Client{Timeout: 30 * time.Second},
		requests:  1,
		userAgent: "SimpleBench 0.0.1 Alpha",
		url:       URL,
		startAt:   time.Now(),
		stdout:    os.Stdout,
		stderr:    os.Stderr,
		wg:        &sync.WaitGroup{},
	}
	for _, o := range opts {
		o(loadgen)
	}
	return loadgen, nil
}

func WithRequests(reqs int) Option {
	return func(lg *LoadGen) {
		lg.requests = reqs
	}
}

func WithHTTPUserAgent(userAgent string) Option {
	return func(lg *LoadGen) {
		lg.userAgent = userAgent
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(lg *LoadGen) {
		lg.client = client
	}
}

func WithStdout(w io.Writer) Option {
	return func(lg *LoadGen) {
		lg.stdout = w
	}
}

func WithStderr(w io.Writer) Option {
	return func(lg *LoadGen) {
		lg.stderr = w
	}
}

func (lg LoadGen) GetHTTPUserAgent() string {
	return lg.userAgent
}

func (lg LoadGen) GetHTTPClient() *http.Client {
	return lg.client
}

func (lg LoadGen) GetStartTime() time.Time {
	return lg.startAt
}

func (lg LoadGen) GetStats() Stats {
	return lg.stats
}

func (lg LoadGen) GetRequests() int {
	return lg.requests
}

func (lg *LoadGen) AddToWG(count int) {
	lg.wg.Add(count)
}

func (lg *LoadGen) WaitForWG() {
	lg.wg.Wait()
}

func (lg *LoadGen) DoRequest(url string) {
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
	resp, err := lg.client.Do(req)
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

func (lg *LoadGen) Run() {
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

func (lg *LoadGen) RecordRequest() {
	atomic.AddUint64(&lg.stats.Requests, 1)
}

func (lg *LoadGen) RecordSuccess() {
	atomic.AddUint64(&lg.stats.Success, 1)
}

func (lg *LoadGen) RecordFailure() {
	atomic.AddUint64(&lg.stats.Failures, 1)
}

func (lg LoadGen) LogStdOut(msg string) {
	fmt.Fprint(lg.stdout, msg)
}

func (lg LoadGen) LogStdErr(msg string) {
	fmt.Fprint(lg.stderr, msg)
}
