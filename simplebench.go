package simplebench

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type LoadGen struct {
	Client    *http.Client
	Requests  int
	StartAt   time.Time
	Stats     Stats
	userAgent string
	URL       string
	Wg        *sync.WaitGroup
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
		Client:    &http.Client{Timeout: 30 * time.Second},
		Requests:  1,
		userAgent: "SimpleBench 0.0.1 Alpha",
		URL:       URL,
		StartAt:   time.Now(),
		Wg:        &sync.WaitGroup{},
	}
	for _, o := range opts {
		o(loadgen)
	}
	return loadgen, nil
}

func WithRequests(reqs int) Option {
	return func(lg *LoadGen) { lg.Requests = reqs }
}

func WithHTTPUserAgent(userAgent string) Option {
	return func(lg *LoadGen) { lg.userAgent = userAgent }
}

func WithHTTPClient(client *http.Client) Option {
	return func(lg *LoadGen) { lg.Client = client }
}

func (lg LoadGen) GetHTTPUserAgent() string { return lg.userAgent }

func (lg *LoadGen) DoRequest(url string) {
	defer lg.Wg.Done()
	atomic.AddUint64(&lg.Stats.Requests, 1)
	req, err := http.NewRequest(http.MethodGet, lg.URL, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		atomic.AddUint64(&lg.Stats.Failures, 1)
	}
	req.Header.Set("user-agent", lg.GetHTTPUserAgent())
	req.Header.Set("accept", "*/*")
	resp, err := lg.Client.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		atomic.AddUint64(&lg.Stats.Failures, 1)
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "unexpected status code %d\n", resp.StatusCode)
		atomic.AddUint64(&lg.Stats.Failures, 1)
	}
	atomic.AddUint64(&lg.Stats.Success, 1)
}

func (lg *LoadGen) Run() {
	bencher := func() <-chan string {
		work := make(chan string, lg.Requests)
		go func() {
			defer close(work)
			for x := 0; x < lg.Requests; x++ {
				work <- lg.URL
			}
		}()
		return work
	}

	work := bencher()
	for url := range work {
		lg.Wg.Add(1)
		go lg.DoRequest(url)
	}
	lg.Wg.Wait()
	fmt.Printf("URL %q benchmark is done\n", lg.URL)
	fmt.Printf("Time: %v Requests: %d Success: %d Failures: %d\n", time.Since(lg.StartAt), lg.Stats.Requests, lg.Stats.Success, lg.Stats.Failures)
}
