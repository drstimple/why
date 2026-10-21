package diagnose

import "fmt"

// Evidence is a claim grounded in a named collector. IDs are stable within a report.
type Evidence struct {
	ID     string `json:"id"`
	Claim  string `json:"claim"`
	Source string `json:"source"`
	Kind   string `json:"kind"`
}
type Hypothesis struct {
	ID         string   `json:"id"`
	Claim      string   `json:"claim"`
	Supporting []string `json:"supporting_evidence"`
	Missing    []string `json:"missing_evidence"`
	Verdict    string   `json:"verdict"`
}
type Ledger struct {
	Evidence   []Evidence   `json:"evidence"`
	Hypotheses []Hypothesis `json:"hypotheses"`
	NextSteps  []string     `json:"next_steps"`
}

// BuildLedger deliberately makes no causal claims from CPU, ancestry, or memory alone.
func BuildLedger(r Report) Ledger {
	l := Ledger{Evidence: []Evidence{}, Hypotheses: []Hypothesis{}, NextSteps: []string{}}
	for i, f := range r.Facts {
		l.Evidence = append(l.Evidence, Evidence{ID: fmt.Sprintf("E%03d", i+1), Claim: fmt.Sprintf("%s = %s", f.Name, f.Value), Source: f.Source, Kind: "observed"})
	}
	if r.Sample != nil {
		s := r.Sample
		l.Evidence = append(l.Evidence, Evidence{ID: fmt.Sprintf("E%03d", len(l.Evidence)+1), Claim: fmt.Sprintf("ps CPU estimate %.1f%% -> %.1f%%; memory estimate %.1f%% -> %.1f%%", s.BeforeCPUPercent, s.AfterCPUPercent, s.BeforeMemoryPercent, s.AfterMemoryPercent), Source: s.Source, Kind: "observed"})
	}
	parentRefs := []string{}
	for i, f := range r.Facts {
		if f.Name == "parent_pid" {
			parentRefs = append(parentRefs, fmt.Sprintf("E%03d", i+1))
		}
	}
	l.Hypotheses = append(l.Hypotheses, Hypothesis{ID: "H001", Claim: "The process was started by its reported parent", Supporting: parentRefs, Missing: []string{"Process creation event or audit trail", "Verified process start identities"}, Verdict: "unproven"})
	l.NextSteps = append(l.NextSteps, "Check service-manager logs or audit events for process creation context.")
	if r.Sample != nil {
		l.Hypotheses = append(l.Hypotheses, Hypothesis{ID: "H002", Claim: "The process is CPU-bound", Supporting: []string{fmt.Sprintf("E%03d", len(l.Evidence))}, Missing: []string{"Interval CPU time measurements", "Thread scheduling and wait-state evidence"}, Verdict: "insufficient evidence"})
		l.NextSteps = append(l.NextSteps, "Capture interval CPU time and thread states before attributing CPU pressure.")
	}
	return l
}
