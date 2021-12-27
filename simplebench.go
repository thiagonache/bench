package simplebench

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

type LoadGen struct {
	Client         *http.Client
	requests       int
	URL, UserAgent string
	Wg             *sync.WaitGroup
}

type Option func(*LoadGen)

func NewLoadGen(opts ...Option) *LoadGen {
	loadgen := &LoadGen{
		Client:    &http.Client{Timeout: 30 * time.Second},
		requests:  1,
		UserAgent: "SimpleBench 0.0.1 Alpha",
		Wg:        &sync.WaitGroup{},
	}
	for _, o := range opts {
		o(loadgen)
	}
	return loadgen
}

func WithRequests(reqs int) Option {
	return func(lg *LoadGen) { lg.requests = reqs }
}

func WithHTTPUserAgent(userAgent string) Option {
	return func(lg *LoadGen) { lg.UserAgent = userAgent }
}

func (lg LoadGen) GetRequests() int { return lg.requests }

func (lg LoadGen) GetHTTPUserAgent() string { return lg.UserAgent }

func (lg LoadGen) DoRequest(url string) error {
	defer lg.Wg.Done()
	for x := 0; x < lg.requests; x++ {
		req, err := http.NewRequest(http.MethodGet, url, nil)
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
