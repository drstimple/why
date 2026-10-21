package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"example.com/why/internal/diagnose"
)

const version = "0.6.1"

func printVersion(w io.Writer) {
	fmt.Fprintf(w, "why %s\n", version)
}

func main() {
	whyDepth := countWhyFlags(os.Args[1:])
	showVersion := flag.Bool("version", false, "print version and exit")
	whyWhy := flag.Bool("why", false, "explain why the why command exists")
	shell := flag.Bool("shell", false, "investigate the parent process (usually your invoking shell)")
	sample := flag.Duration("sample", 0, "compare two ps observations (100ms to 10s; e.g. 1s)")
	notCondition := flag.String("not", "", "check an expected condition (tcp:host:port)")
	timeout := flag.Duration("timeout", 2*time.Second, "TCP connection timeout (max 30s)")
	because := flag.Bool("because", false, "show evidence ledger and unproven hypotheses")
	jsonOut := flag.Bool("json", false, "output structured JSON")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: why --version\n       why [--json] [--sample 1s] [--because] <pid>\n       why --shell [--json] [--because] [--sample 1s]\n       why --not tcp:host:port [--timeout 2s] [--json]\n\nEvidence-first diagnostics; no invented causes.")
	}
	flag.CommandLine.Parse(normalizeWhyFlags(os.Args[1:]))
	if *showVersion {
		if flag.NArg() != 0 || *whyWhy || *shell || *sample != 0 || *notCondition != "" || *because || *jsonOut || *timeout != 2*time.Second {
			fmt.Fprintln(os.Stderr, "--version cannot be combined with other options or arguments")
			os.Exit(2)
		}
		printVersion(os.Stdout)
		return
	}
	if *whyWhy {
		if flag.NArg() != 0 || *notCondition != "" || *sample != 0 || *because || *shell || *timeout != 2*time.Second {
			fmt.Fprintln(os.Stderr, "--why accepts no PID or diagnostic options (only --json)")
			os.Exit(2)
		}
		if whyDepth >= 2 {
			printRecursiveWhy(os.Stdout, *jsonOut)
		} else {
			printWhyWhy(os.Stdout, *jsonOut)
		}
		return
	}
	if *notCondition != "" {
		if flag.NArg() != 0 || *sample != 0 || *because || *shell {
			fmt.Fprintln(os.Stderr, "--not cannot be combined with PID, --sample, or --because")
			os.Exit(2)
		}
		result, err := diagnose.CheckTCP(context.Background(), *notCondition, *timeout)
		if err != nil {
			fmt.Fprintln(os.Stderr, "why:", err)
			os.Exit(2)
		}
		if *jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(result)
		} else {
			fmt.Printf("CONDITION %s\nSTATUS %s\n", result.Condition, strings.ToUpper(result.Status))
			for _, e := range result.Evidence {
				fmt.Printf("  %s [%s] %s (%s)\n", e.ID, e.Kind, e.Claim, e.Source)
			}
			for _, u := range result.Unknowns {
				fmt.Printf("  [unknown] %s\n", u)
			}
			for _, n := range result.NextSteps {
				fmt.Printf("  next: %s\n", n)
			}
		}
		// 0: expectation unmet; 1: expectation met; 3: insufficient evidence.
		switch result.Status {
		case "unmet":
			os.Exit(0)
		case "met":
			os.Exit(1)
		default:
			os.Exit(3)
		}
	}
	if (*shell && flag.NArg() != 0) || (!*shell && flag.NArg() != 1) {
		flag.Usage()
		os.Exit(2)
	}
	var pid int
	var err error
	if *shell {
		pid = os.Getppid()
	} else {
		pid, err = strconv.Atoi(flag.Arg(0))
	}
	if err != nil || pid <= 0 {
		fmt.Fprintln(os.Stderr, "PID must be a positive integer")
		os.Exit(2)
	}
	var report diagnose.Report
	if *sample != 0 {
		report, err = diagnose.SampleProcess(pid, time.Duration(*sample))
	} else {
		report, err = diagnose.Process(pid)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "why:", err)
		os.Exit(1)
	}
	if *because {
		ledger := diagnose.BuildLedger(report)
		if *jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(struct {
				Report diagnose.Report `json:"report"`
				Ledger diagnose.Ledger `json:"ledger"`
			}{report, ledger})
			return
		}
		fmt.Printf("WHY %d: %s\n\nEVIDENCE\n", report.PID, report.Command)
		for _, e := range ledger.Evidence {
			fmt.Printf("  %s [%s] %s (%s)\n", e.ID, e.Kind, e.Claim, e.Source)
		}
		fmt.Println("\nHYPOTHESES")
		for _, h := range ledger.Hypotheses {
			fmt.Printf("  %s %s — %s\n", h.ID, h.Claim, h.Verdict)
			for _, m := range h.Missing {
				fmt.Printf("    missing: %s\n", m)
			}
		}
		fmt.Println("\nNEXT STEPS")
		for _, step := range ledger.NextSteps {
			fmt.Printf("  %s\n", step)
		}
		return
	}
	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(report)
		return
	}
	fmt.Printf("PROCESS %d: %s\n", report.PID, report.Command)
	for _, f := range report.Facts {
		fmt.Printf("  %-16s %s\n", strings.ToUpper(f.Name)+":", f.Value)
	}
	fmt.Println("\nANCESTRY (nearest parent first)")
	for _, a := range report.Ancestors {
		fmt.Printf("  %d <- %d  %s\n", a.PID, a.ParentPID, a.Command)
	}
	if report.Sample != nil {
		fmt.Printf("\nSAMPLE (%dms): CPU %.1f%% -> %.1f%%; MEM %.1f%% -> %.1f%%\n", report.Sample.IntervalMS, report.Sample.BeforeCPUPercent, report.Sample.AfterCPUPercent, report.Sample.BeforeMemoryPercent, report.Sample.AfterMemoryPercent)
	}
	fmt.Println("\nEXPLANATION")
	for _, e := range report.Explanations {
		fmt.Printf("  [%s] %s\n", e.Kind, e.Text)
	}
	for _, u := range report.Unknowns {
		fmt.Printf("  [unknown] %s\n", u)
	}
}

// printWhyWhy is deliberately static: a small, honest Easter egg, not a diagnosis.
func printWhyWhy(w io.Writer, asJSON bool) {
	message := struct {
		Question                string `json:"question"`
		Observation             string `json:"observation"`
		Problem                 string `json:"problem"`
		Purpose                 string `json:"purpose"`
		Motivation              string `json:"motivation"`
		CommercialJustification string `json:"commercial_justification"`
		Verdict                 string `json:"verdict"`
	}{
		"Why does why exist?",
		"Unix offers powerful tools for observing system state.",
		"Observations alone do not establish causes.",
		"Help people investigate explanations without confusing hypotheses with facts.",
		"Two nerds thought it would be fun.",
		"None. Not for profit.",
		"Entirely reasonable.",
	}
	if asJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(message)
		return
	}
	fmt.Fprintf(w, "WHY DOES WHY EXIST?\n\nOBSERVATION:\n  %s\n\nPROBLEM:\n  %s\n\nPURPOSE:\n  %s\n\nAUTHOR MOTIVATION:\n  %s\n\nCOMMERCIAL JUSTIFICATION:\n  %s\n\nVERDICT:\n  %s\n", message.Observation, message.Problem, message.Purpose, message.Motivation, message.CommercialJustification, message.Verdict)
}

// normalizeWhyFlags lets --why repeat without affecting ordinary flag parsing.
func normalizeWhyFlags(args []string) []string {
	out := make([]string, 0, len(args))
	seen := false
	for _, arg := range args {
		if arg == "--why" || arg == "-why" {
			if seen {
				continue
			}
			seen = true
		}
		out = append(out, arg)
	}
	return out
}

func countWhyFlags(args []string) int {
	count := 0
	for _, arg := range args {
		if arg == "--why" || arg == "-why" {
			count++
		}
	}
	return count
}

func printRecursiveWhy(w io.Writer, asJSON bool) {
	result := struct {
		Status      string `json:"status"`
		Explanation string `json:"explanation"`
		NextStep    string `json:"next_step"`
		Verdict     string `json:"verdict"`
	}{
		"recursive existential inquiry detected",
		"The question of why \"why\" exists has already been answered. Further investigation would require philosophy.",
		"man philosophy",
		"Go outside.",
	}
	if asJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(result)
		return
	}
	fmt.Fprintf(w, "RECURSIVE EXISTENTIAL INQUIRY DETECTED.\n\nThe question of why \"why\" exists has already\nbeen answered.\n\nFurther investigation would require philosophy.\n\nSUGGESTED COMMAND:\n  %s\n\nVERDICT:\n  %s\n", result.NextStep, result.Verdict)
}
