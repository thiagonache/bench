package bench

import (
	"errors"
	"io"
	"os"
)

var ErrEmptyStats = errors.New("stats cannot be empty")

type CompareStats struct {
	S1, S2         Stats
	Stdout, Stderr io.Writer
}

func NewCompareStats(s1, s2 Stats, opts ...CMPOption) (*CompareStats, error) {
	if s1 == (Stats{}) || s2 == (Stats{}) {
		return &CompareStats{}, ErrEmptyStats
	}
	cs := &CompareStats{
		S1:     s1,
		S2:     s2,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	for _, o := range opts {
		err := o(cs)
		if err != nil {
			return nil, err
		}
	}
	return cs, nil
}

func WithCMPStdout(w io.Writer) CMPOption {
	return func(cs *CompareStats) error {
		if w == nil {
			return ErrValueCannotBeNil
		}
		cs.Stdout = w
		return nil
	}
}

func WithCMPStderr(w io.Writer) CMPOption {
	return func(cs *CompareStats) error {
		if w == nil {
			return ErrValueCannotBeNil
		}
		cs.Stderr = w
		return nil
	}
}

type CMPOption func(*CompareStats) error

// func (s CompareStats) String() string {
// 	buf := &bytes.Buffer{}
// 	fmt.Fprintf(buf, "Site %s\n", s.S1.URL)
// 	writer := tabwriter.NewWriter(buf, 20, 0, 0, ' ', 0)
// 	fmt.Fprintln(writer, "Metric\tOld\tCurrent\tDelta\tPercentage")
// 	p50Delta := s.S2.P50 - s.S1.P50
// 	fmt.Fprintf(writer, "P50(ms)\t%.3f\t%.3f\t%.3f\t%.2f\n", s.S1.P50, s.S2.P50, p50Delta, p50Delta/s.S1.P50*100)
// 	p90Delta := s.S2.P90 - s.S1.P90
// 	fmt.Fprintf(writer, "P90(ms)\t%.3f\t%.3f\t%.3f\t%.2f\n", s.S1.P90, s.S2.P90, p90Delta, p90Delta/s.S1.P90*100)
// 	p99Delta := s.S2.P99 - s.S1.P99
// 	fmt.Fprintf(writer, "P99(ms)\t%.3f\t%.3f\t%.3f\t%.2f\n", s.S1.P99, s.S2.P99, p99Delta, p99Delta/s.S1.P99*100)
// 	writer.Flush()
// 	return buf.String()
// }
