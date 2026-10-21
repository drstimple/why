package diagnose

import "testing"

func TestLedgerNeverClaimsRootCause(t *testing.T) {
	r := Report{PID: 42, Facts: []Fact{{Name: "pid", Value: "42", Source: "ps"}, {Name: "parent_pid", Value: "1", Source: "ps"}}}
	l := BuildLedger(r)
	if len(l.Evidence) != 2 || len(l.Hypotheses) != 1 || l.Hypotheses[0].Verdict != "unproven" {
		t.Fatalf("unexpected ledger: %+v", l)
	}
}
func TestLedgerSampleEvidence(t *testing.T) {
	r := Report{Sample: &Sample{BeforeCPUPercent: 1, AfterCPUPercent: 80, Source: "ps"}}
	l := BuildLedger(r)
	if len(l.Hypotheses) != 2 || l.Hypotheses[1].Verdict != "insufficient evidence" {
		t.Fatalf("unexpected: %+v", l)
	}
}
