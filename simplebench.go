package simplebench

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type LoadGen struct {
	Client    *http.Client
	requests  int
	StartAt   time.Time
	userAgent string
	URL       string
	Wg        *sync.WaitGroup
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
		requests:  1,
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
	return func(lg *LoadGen) { lg.requests = reqs }
}

func WithHTTPUserAgent(userAgent string) Option {
	return func(lg *LoadGen) { lg.userAgent = userAgent }
}

func (lg LoadGen) GetRequests() int { return lg.requests }

func (lg LoadGen) GetHTTPUserAgent() string { return lg.userAgent }

func (lg LoadGen) DoRequest() error {
	defer lg.Wg.Done()
	for x := 0; x < lg.requests; x++ {
		req, err := http.NewRequest(http.MethodGet, lg.URL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("user-agent", lg.GetHTTPUserAgent())
		req.Header.Set("accept", "*/*")
		resp, err := lg.Client.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code %d", resp.StatusCode)
		}
	}
	return nil
}
