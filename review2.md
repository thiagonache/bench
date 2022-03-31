* We still get an error for URLs without a scheme:

```
go run cmd/main.go example.com
invalid URL "example.com"
exit status 1
```

* We're still reporting too many decimal places in the timing stats:

```
go run cmd/main.go https://example.com
URL: "https://example.com" benchmark is done
Time: 421.975629ms Requests: 1 Success: 1 Failures: 0
90th percentile: 420.44292ms 99th percentile: 420.44292ms
Fastest: 420.44292ms Mean: 420.44292ms Slowest: 420.44292ms
```

* The mean time usually isn't that valuable for things like this. No user experiences the mean latency, and the value of the mean is disproportionately affected by outliers (for example, if 99 out of 100 requests take 1ms, but the 100th request takes 10 seconds, the mean is 100 milliseconds, which is completely unrepresentative of real user experiences). That's why we usually focus on percentiles. A useful percentile to look at is the *median* (P50). Half of "users" experienced latencies below this, and half above.

* I would eliminate the `t.work` channel, as right now it's additional complexity which is never used; it always just contains `t.URL`, so we might as simply call `t.DoRequest()` with no argument and have `DoRequest` look at `t.URL`.

* We can reduce a bit of noise in the `SetMetrics` method by doing something like:

```go
times := t.TimeRecorder.ExecutionsTime
if len(times) < 1 {
    return ErrTimeNotRecorded
}
sort.Slice(times, func(i, j int) bool {
    return times[i] < times[j]
})
...
```

* Try something like https://github.com/montanaflynn/stats for the stats calculations — no point reinventing these yourself.

* I think we can remove `else` here:

```go
		if v < t.stats.Fastest {
			t.stats.Fastest = v
		} else if v > t.stats.Slowest {
			t.stats.Slowest = v
		}
```

* We can remove `SortExecutionsTime` altogether—it doesn't contain anything.

* We can use `Since` to calculate the elapsed time:

```go
elapsedTime := time.Since(start)
```
