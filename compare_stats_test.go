package bench_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/thiagonache/bench"
)

func TestNewCompareStats_ReturnsCorrectCompareStats(t *testing.T) {
	t.Parallel()
	want := &bench.CompareStats{
		S1: bench.Stats{P99: 99},
		S2: bench.Stats{P99: 98},
	}
	got, err := bench.NewCompareStats(
		bench.Stats{P99: 99},
		bench.Stats{P99: 98},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(want.S1, got.S1) {
		t.Errorf("S1 isn't equal: %v", cmp.Diff(want.S1, got.S1))
	}
	if !cmp.Equal(want.S2, got.S2) {
		t.Errorf("S2 isn't equal: %v", cmp.Diff(want.S2, got.S2))
	}
}

func TestNewCompareStats_ErrorsIfS1IsEmpty(t *testing.T) {
	t.Parallel()
	s1 := bench.Stats{}
	s2 := bench.Stats{
		URL: "http://fake.url",
		P50: 1,
		P90: 2,
		P99: 3,
	}
	_, err := bench.NewCompareStats(s1, s2)
	if !errors.Is(err, bench.ErrEmptyStats) {
		t.Fatalf("want error ErrEmptyStats got %v", err)
	}
}

func TestNewCompareStats_ErrorsIfS2IsEmpty(t *testing.T) {
	t.Parallel()
	s1 := bench.Stats{
		URL: "http://fake.url",
		P50: 1,
		P90: 2,
		P99: 3,
	}
	s2 := bench.Stats{}
	_, err := bench.NewCompareStats(s1, s2)
	if !errors.Is(err, bench.ErrEmptyStats) {
		t.Fatalf("want error ErrEmptyStats got %v", err)
	}
}

func TestNewCompareStats_WithNilStdoutReturnsErrorValueCannotBeNil(t *testing.T) {
	t.Parallel()
	_, err := bench.NewCompareStats(
		bench.Stats{P99: 99},
		bench.Stats{P99: 98},
		bench.WithCMPStdout(nil),
	)
	if !errors.Is(err, bench.ErrValueCannotBeNil) {
		t.Errorf("want ErrValueCannotBeNil error if stdout is nil, got %v", err)
	}
}

func TestNewCompareStats_WithNilStderrReturnsErrorValueCannotBeNil(t *testing.T) {
	t.Parallel()
	_, err := bench.NewCompareStats(
		bench.Stats{P99: 99},
		bench.Stats{P99: 98},
		bench.WithCMPStderr(nil),
	)
	if !errors.Is(err, bench.ErrValueCannotBeNil) {
		t.Errorf("want ErrValueCannotBeNil error if stderr is nil, got %v", err)
	}
}
