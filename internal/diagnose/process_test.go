package diagnose

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestProcessAncestry(t *testing.T) {
	rows := map[int]string{42: "42 7 alice S 0.0 0.1 01:20 myprogram", 7: "7 1 root S 0.0 0.1 03:10 launcher", 1: "1 0 root S 0.0 0.1 10:00 init"}
	run := func(_ context.Context, pid int, _ string) (string, error) { return rows[pid], nil }
	r, err := process(42, run)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Ancestors) != 2 || r.Ancestors[0].PID != 7 || r.Ancestors[1].PID != 1 {
		t.Fatalf("bad ancestry: %+v", r.Ancestors)
	}
	if r.SchemaVersion != "0.4" {
		t.Fatal("schema version missing")
	}
}
func TestProcessMissingAncestor(t *testing.T) {
	run := func(_ context.Context, pid int, _ string) (string, error) {
		if pid == 42 {
			return "42 7 alice S 0.0 0.1 01:20 program", nil
		}
		return "", errors.New("gone")
	}
	r, err := process(42, run)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Ancestors) != 0 || !strings.Contains(r.Unknowns[0], "Could not inspect ancestor") {
		t.Fatalf("expected unknown: %+v", r.Unknowns)
	}
}
func TestProcessRejectsWrongPID(t *testing.T) {
	run := func(_ context.Context, _ int, _ string) (string, error) { return "9 1 root S 0.0 0.1 01:20 other", nil }
	if _, err := process(42, run); err == nil {
		t.Fatal("expected error")
	}
}
func TestProcessCycle(t *testing.T) {
	rows := map[int]string{42: "42 7 alice S 0.0 0.1 01:20 program", 7: "7 42 root S 0.0 0.1 01:20 parent"}
	run := func(_ context.Context, pid int, _ string) (string, error) { return rows[pid], nil }
	r, err := process(42, run)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Ancestors) != 1 || !strings.Contains(r.Unknowns[0], "cycle") {
		t.Fatalf("cycle not detected: %+v", r)
	}
}
