package diagnose

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Fact struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Source string `json:"source"`
}
type Explanation struct {
	Kind     string   `json:"kind"`
	Text     string   `json:"text"`
	Evidence []string `json:"evidence"`
}
type Ancestor struct {
	PID       int    `json:"pid"`
	ParentPID int    `json:"parent_pid"`
	Command   string `json:"command"`
}
type Report struct {
	SchemaVersion string        `json:"schema_version"`
	Sample        *Sample       `json:"sample,omitempty"`
	PID           int           `json:"pid"`
	Command       string        `json:"command"`
	Facts         []Fact        `json:"facts"`
	Ancestors     []Ancestor    `json:"ancestors"`
	Explanations  []Explanation `json:"explanations"`
	Unknowns      []string      `json:"unknowns"`
}

type psRunner func(context.Context, int, string) (string, error)

func realPS(ctx context.Context, pid int, fields string) (string, error) {
	out, err := exec.CommandContext(ctx, "ps", "-p", strconv.Itoa(pid), "-o", fields).Output()
	if err != nil {
		return "", fmt.Errorf("ps inspection failed for PID %d: %w", pid, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func parsePS(line string) (Ancestor, []string, error) {
	v := strings.Fields(line)
	if len(v) < 8 {
		return Ancestor{}, nil, fmt.Errorf("unexpected ps output: %q", line)
	}
	pid, err := strconv.Atoi(v[0])
	if err != nil {
		return Ancestor{}, nil, fmt.Errorf("invalid PID in ps output: %w", err)
	}
	ppid, err := strconv.Atoi(v[1])
	if err != nil {
		return Ancestor{}, nil, fmt.Errorf("invalid parent PID in ps output: %w", err)
	}
	return Ancestor{PID: pid, ParentPID: ppid, Command: strings.Join(v[7:], " ")}, v, nil
}

const fields = "pid=,ppid=,user=,stat=,%cpu=,%mem=,etime=,comm="

func Process(pid int) (Report, error) { return process(pid, realPS) }

func process(pid int, run psRunner) (Report, error) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		return Report{}, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	if pid <= 0 {
		return Report{}, fmt.Errorf("PID must be positive")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	output, err := run(ctx, pid, fields)
	if err != nil {
		return Report{}, err
	}
	root, v, err := parsePS(output)
	if err != nil {
		return Report{}, err
	}
	if root.PID != pid {
		return Report{}, fmt.Errorf("ps returned PID %d, expected %d", root.PID, pid)
	}
	r := Report{SchemaVersion: "0.4", PID: pid, Command: root.Command, Ancestors: []Ancestor{}, Facts: []Fact{}, Explanations: []Explanation{}, Unknowns: []string{}}
	keys := []string{"pid", "parent_pid", "user", "state", "cpu_percent", "memory_percent", "elapsed", "command"}
	for i, k := range keys {
		value := v[i]
		if i == 7 {
			value = root.Command
		}
		r.Facts = append(r.Facts, Fact{Name: k, Value: value, Source: "ps"})
	}
	r.Explanations = append(r.Explanations, Explanation{Kind: "observed", Text: fmt.Sprintf("Process %d is visible to ps; its parent PID is %d.", pid, root.ParentPID), Evidence: []string{"ps:pid", "ps:parent_pid"}})
	seen := map[int]bool{pid: true}
	next := root.ParentPID
	for depth := 0; next > 0 && depth < 32; depth++ {
		if seen[next] {
			r.Unknowns = append(r.Unknowns, "Process ancestry contains a cycle or reused PID; traversal stopped.")
			break
		}
		seen[next] = true
		line, e := run(ctx, next, fields)
		if e != nil {
			r.Unknowns = append(r.Unknowns, fmt.Sprintf("Could not inspect ancestor PID %d (it may have exited or be inaccessible).", next))
			break
		}
		a, _, e := parsePS(line)
		if e != nil || a.PID != next {
			r.Unknowns = append(r.Unknowns, fmt.Sprintf("Ancestor PID %d returned inconsistent process information.", next))
			break
		}
		r.Ancestors = append(r.Ancestors, a)
		if a.ParentPID == 0 {
			break
		}
		next = a.ParentPID
		if depth == 31 {
			r.Unknowns = append(r.Unknowns, "Ancestry truncated at 32 levels.")
		}
	}
	r.Unknowns = append(r.Unknowns, "Parentage does not establish why the process was launched.", "CPU and memory readings are instantaneous observations, not root-cause evidence.", "Process IDs may be reused; ancestry is a best-effort snapshot, not a verified historical chain.")
	return r, nil
}
