# bench

An HTTP load tester tool.

## Basic usage

### Import

```go
import "github.com/thiagonache/bench"
```

### CLI

#### Run

```bash
$ simplebench run -r 10 -c 2 -u https://example.com
Site: https://example.com
Requests: 10
Successes: 10
Failures: 0
P50(ms): 116.786
P90(ms): 484.001
P99(ms): 484.006
```

#### Cmp

```bash
$ simplebench run -r 50 -u https://example.com > stats1.txt
$ simplebench run -r 50 -u https://example.com > stats2.txt
$ simplebench cmp stats{1,2}.txt
Site https://example.com
Metric              Old                 New                 Delta               Percentage
P50(ms)             116.912             116.869             -0.043              -0.04
P90(ms)             117.324             117.077             -0.247              -0.21
P99(ms)             486.683             483.103             -3.580              -0.74
```
