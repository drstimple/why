package diagnose

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

// Sample compares two ps observations. ps %CPU is an OS-defined estimate,
// not CPU time consumed specifically between these observations.
type Sample struct {
	IntervalMS          int64   `json:"interval_ms"`
	BeforeCPUPercent    float64 `json:"before_cpu_percent"`
	AfterCPUPercent     float64 `json:"after_cpu_percent"`
	BeforeMemoryPercent float64 `json:"before_memory_percent"`
	AfterMemoryPercent  float64 `json:"after_memory_percent"`
	Source              string  `json:"source"`
}

func SampleProcess(pid int, interval time.Duration) (Report, error) {
	if interval < 100*time.Millisecond || interval > 10*time.Second {
		return Report{}, fmt.Errorf("sample interval must be between 100ms and 10s")
	}
	return sampleProcess(pid, interval, realPS)
}

func sampleProcess(pid int, interval time.Duration, run psRunner) (Report, error) {
	r, err := process(pid, run)
	if err != nil {
		return r, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), interval+3*time.Second)
	defer cancel()
	before, err := run(ctx, pid, fields)
	if err != nil {
		r.Unknowns = append(r.Unknowns, "Sampling failed: initial observation unavailable.")
		return r, nil
	}
	a, av, err := parsePS(before)
	if err != nil || a.PID != pid {
		r.Unknowns = append(r.Unknowns, "Sampling failed: initial observation inconsistent.")
		return r, nil
	}
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return r, ctx.Err()
	case <-timer.C:
	}
	after, err := run(ctx, pid, fields)
	if err != nil {
		r.Unknowns = append(r.Unknowns, "Sampling failed: process exited or became inaccessible.")
		return r, nil
	}
	b, bv, err := parsePS(after)
	if err != nil || b.PID != pid || a.ParentPID != b.ParentPID || av[2] != bv[2] || a.Command != b.Command {
		r.Unknowns = append(r.Unknowns, "Sampling observations may refer to different processes (PID reuse or process change); comparison discarded.")
		return r, nil
	}
	ac, e1 := strconv.ParseFloat(av[4], 64)
	bc, e2 := strconv.ParseFloat(bv[4], 64)
	am, e3 := strconv.ParseFloat(av[5], 64)
	bm, e4 := strconv.ParseFloat(bv[5], 64)
	if e1 != nil || e2 != nil || e3 != nil || e4 != nil {
		r.Unknowns = append(r.Unknowns, "Sampling metrics could not be parsed.")
		return r, nil
	}
	r.Sample = &Sample{IntervalMS: interval.Milliseconds(), BeforeCPUPercent: ac, AfterCPUPercent: bc, BeforeMemoryPercent: am, AfterMemoryPercent: bm, Source: "ps (two observations)"}
	r.Explanations = append(r.Explanations, Explanation{Kind: "observed", Text: fmt.Sprintf("ps reported CPU %.1f%% then %.1f%%, memory %.1f%% then %.1f%% across approximately %s.", ac, bc, am, bm, interval), Evidence: []string{"ps:cpu_percent", "ps:memory_percent"}})
	r.Unknowns = append(r.Unknowns, "ps CPU percentages are platform-defined estimates, not interval CPU utilization; two samples cannot establish a causal explanation.")
	return r, nil
}
