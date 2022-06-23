# bench

[![Go](https://github.com/thiagonache/bench/actions/workflows/go.yml/badge.svg)](https://github.com/thiagonache/bench/actions/workflows/go.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/thiagonache/bench.svg)](https://pkg.go.dev/github.com/thiagonache/bench)
[![Go Report Card](https://goreportcard.com/badge/github.com/thiagonache/bench)](https://goreportcard.com/report/github.com/thiagonache/bench)

An HTTP load tester and compare results tool.

## Import

```go
import "github.com/thiagonache/bench"
```

## CLI

### Run

- GET

  ```bash
  $ simplebench run -r 20 -c 2 -u https://httpbin.org
  Site: https://httpbin.org
  Requests: 20
  Successes: 20
  Failures: 0
  P50(ms): 150.359
  P90(ms): 431.346
  P99(ms): 761.359
  ```

- POST

  ```bash
    $ simplebench run -m POST -t "application/json" -b '{"data":"abc"}' -r 20 -c 2 -u https://httpbin.org/post
    Site: https://httpbin.org/post
    Requests: 20
    Successes: 20
    Failures: 0
    P50(ms): 143.970
    P90(ms): 395.692
    P99(ms): 574.325
  ```

- DELETE

  ```bash
    $ simplebench run -m DELETE -r 20 -c 2 -u https://httpbin.org/delete
    Site: https://httpbin.org/delete
    Requests: 20
    Successes: 20
    Failures: 0
    P50(ms): 145.996
    P90(ms): 596.712
    P99(ms): 640.887
  ```

### Cmp

```bash
$ simplebench run -r 10 -u https://httpbin.org/delay/2 > stats1.txt
$ simplebench run -r 10 -u https://httpbin.org/delay/1 > stats2.txt
$ simplebench cmp stats{1,2}.txt
Site: https://httpbin.org/delay/2
Metric              Old                 New                 Delta               Percentage
P50(ms)             2144.024            1146.673            -997.351            -46.52
P90(ms)             2221.990            1362.111            -859.879            -38.70
P99(ms)             2599.690            1613.528            -986.162            -37.93
```
