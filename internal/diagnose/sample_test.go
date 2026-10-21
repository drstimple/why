package diagnose

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestSampleProcess(t *testing.T) {
	n := 0
	run := func(_ context.Context, pid int, _ string) (string, error) {
		if pid == 1 {
			return "1 0 root S 0.0 0.1 10:00 init", nil
		}
		n++
		if n >= 3 {
			return "42 1 alice S 2.5 0.2 01:20 worker", nil
		}
		return "42 1 alice S 1.0 0.1 01:20 worker", nil
	}
	r, err := sampleProcess(42, time.Millisecond, run)
	if err != nil {
		t.Fatal(err)
	}
	if r.Sample == nil || r.Sample.AfterCPUPercent != 2.5 {
		t.Fatalf("unexpected sample: %+v", r.Sample)
	}
}
func TestSampleRejectsPIDReuse(t *testing.T) {
	n := 0
	run := func(_ context.Context, pid int, _ string) (string, error) {
		if pid == 1 {
			return "1 0 root S 0.0 0.1 10:00 init", nil
		}
		n++
		if n >= 3 {
			return "42 9 alice S 2.5 0.2 00:01 other", nil
		}
		return "42 1 alice S 1.0 0.1 01:20 worker", nil
	}
	r, err := sampleProcess(42, time.Millisecond, run)
	if err != nil {
		t.Fatal(err)
	}
	if r.Sample != nil || !strings.Contains(strings.Join(r.Unknowns, " "), "PID reuse") {
		t.Fatalf("expected discarded sample: %+v", r)
	}
}
func TestSampleIntervalValidation(t *testing.T) {
	if _, err := SampleProcess(1, 50*time.Millisecond); err == nil {
		t.Fatal("expected interval error")
	}
}
