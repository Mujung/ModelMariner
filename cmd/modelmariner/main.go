// Command modelmariner is an offline LLM trace evaluation and routing-policy
// compiler. It ingests provider-neutral JSONL traces, computes reliability and
// Pareto frontiers, evaluates hard budget/latency/privacy policies, simulates
// routing against the recorded traces, explains its selections, and emits
// deterministic policy and report artifacts. It never contacts a provider.
//
// Usage:
//
//	modelmariner analyze --traces file.jsonl [--policy policy.json] [flags]
//	modelmariner validate --traces file.jsonl
//	modelmariner version
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Mujung/modelmariner/internal/pareto"
	"github.com/Mujung/modelmariner/internal/policy"
	"github.com/Mujung/modelmariner/internal/reliability"
	"github.com/Mujung/modelmariner/internal/report"
	"github.com/Mujung/modelmariner/internal/routing"
	"github.com/Mujung/modelmariner/internal/trace"
)

// version is overridable at build time via -ldflags "-X main.version=...".
var version = "1.0.0"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "modelmariner: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage(os.Stderr)
		return fmt.Errorf("no command given")
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "analyze":
		return cmdAnalyze(rest)
	case "validate":
		return cmdValidate(rest)
	case "version", "--version", "-v":
		fmt.Printf("modelmariner %s\n", version)
		return nil
	case "help", "--help", "-h":
		printUsage(os.Stdout)
		return nil
	default:
		printUsage(os.Stderr)
		return fmt.Errorf("unknown command %q", cmd)
	}
}

// analyzeFlags collects everything the analyze command accepts.
type analyzeFlags struct {
	traces        string
	policy        string
	outDir        string
	format        string
	strict        bool
	deterministic bool
}

func parseAnalyzeFlags(args []string) (analyzeFlags, error) {
	f := analyzeFlags{format: "both", deterministic: true}
	for i := 0; i < len(args); i++ {
		a := args[i]
		next := func() (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("flag %s requires a value", a)
			}
			i++
			return args[i], nil
		}
		var err error
		switch {
		case a == "--traces" || a == "-t":
			f.traces, err = next()
		case strings.HasPrefix(a, "--traces="):
			f.traces = strings.TrimPrefix(a, "--traces=")
		case a == "--policy" || a == "-p":
			f.policy, err = next()
		case strings.HasPrefix(a, "--policy="):
			f.policy = strings.TrimPrefix(a, "--policy=")
		case a == "--out" || a == "-o":
			f.outDir, err = next()
		case strings.HasPrefix(a, "--out="):
			f.outDir = strings.TrimPrefix(a, "--out=")
		case a == "--format":
			f.format, err = next()
		case strings.HasPrefix(a, "--format="):
			f.format = strings.TrimPrefix(a, "--format=")
		case a == "--strict":
			f.strict = true
		case a == "--with-timestamp":
			f.deterministic = false
		default:
			return f, fmt.Errorf("unknown flag %q", a)
		}
		if err != nil {
			return f, err
		}
	}
	if f.traces == "" {
		return f, fmt.Errorf("--traces is required")
	}
	switch f.format {
	case "json", "text", "both":
	default:
		return f, fmt.Errorf("--format must be json, text, or both (got %q)", f.format)
	}
	return f, nil
}

func cmdAnalyze(args []string) error {
	f, err := parseAnalyzeFlags(args)
	if err != nil {
		return err
	}

	ingest, err := loadTraces(f.traces, f.strict)
	if err != nil {
		return err
	}
	if len(ingest.Traces) == 0 {
		return fmt.Errorf("no valid traces found in %s", f.traces)
	}

	sum := reliability.Compute(ingest.Traces)
	par := pareto.Compute(sum)

	var set policy.Set
	var sims []routing.Simulation
	if f.policy != "" {
		pf, err := os.Open(f.policy)
		if err != nil {
			return fmt.Errorf("opening policy file: %w", err)
		}
		set, err = policy.Load(pf)
		pf.Close()
		if err != nil {
			return err
		}
		sims = routing.SimulateAll(set, sum, ingest.Traces)
	}

	rep := report.Build(report.BuildInput{
		Ingest:        ingest,
		Traces:        ingest.Traces,
		Reliability:   sum,
		Pareto:        par,
		Policies:      set,
		Simulations:   sims,
		Deterministic: f.deterministic,
		Now:           time.Now(),
	})

	// Emit to stdout in the requested format.
	if f.format == "text" || f.format == "both" {
		fmt.Print(rep.Text())
	}
	if f.format == "json" {
		b, err := rep.JSON()
		if err != nil {
			return err
		}
		os.Stdout.Write(b)
	}

	// Write files if an output directory was requested.
	if f.outDir != "" {
		if err := writeArtifacts(f.outDir, rep, sims); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "wrote report artifacts to %s\n", f.outDir)
	}
	return nil
}

func writeArtifacts(dir string, rep report.Report, sims []routing.Simulation) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}
	repJSON, err := rep.JSON()
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "report.json"), repJSON, 0o644); err != nil {
		return fmt.Errorf("writing report.json: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "report.txt"), []byte(rep.Text()), 0o644); err != nil {
		return fmt.Errorf("writing report.txt: %w", err)
	}
	if len(sims) > 0 {
		arts := report.CompilePolicyArtifacts(sims)
		artJSON, err := report.ArtifactsJSON(arts)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, "policies.json"), artJSON, 0o644); err != nil {
			return fmt.Errorf("writing policies.json: %w", err)
		}
	}
	return nil
}

func cmdValidate(args []string) error {
	var path string
	strict := false
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--traces" || args[i] == "-t":
			if i+1 >= len(args) {
				return fmt.Errorf("--traces requires a value")
			}
			i++
			path = args[i]
		case strings.HasPrefix(args[i], "--traces="):
			path = strings.TrimPrefix(args[i], "--traces=")
		case args[i] == "--strict":
			strict = true
		default:
			return fmt.Errorf("unknown flag %q", args[i])
		}
	}
	if path == "" {
		return fmt.Errorf("--traces is required")
	}
	ingest, err := loadTraces(path, strict)
	if err != nil {
		return err
	}
	fmt.Printf("validated %s: %d line(s), %d accepted, %d rejected\n",
		path, ingest.Total, len(ingest.Traces), ingest.Rejected)
	for _, ve := range ingest.Errors {
		fmt.Printf("  reject: %s\n", ve.Error())
	}
	if ingest.Rejected > 0 && strict {
		return fmt.Errorf("%d line(s) rejected under --strict", ingest.Rejected)
	}
	return nil
}

// loadTraces opens a JSONL file and ingests it. Under strict mode a single bad
// line aborts the run; otherwise bad lines are collected and skipped.
func loadTraces(path string, strict bool) (trace.IngestResult, error) {
	fh, err := os.Open(path)
	if err != nil {
		return trace.IngestResult{}, fmt.Errorf("opening traces: %w", err)
	}
	defer fh.Close()
	opts := trace.DefaultOptions()
	opts.SkipInvalid = !strict
	res, err := trace.Ingest(fh, opts)
	if err != nil {
		return res, err
	}
	if strict && res.Rejected > 0 {
		return res, fmt.Errorf("%d invalid line(s) under --strict mode", res.Rejected)
	}
	return res, nil
}

func printUsage(w *os.File) {
	fmt.Fprint(w, `modelmariner — offline LLM trace evaluation & routing-policy compiler

USAGE
  modelmariner analyze  --traces FILE [--policy FILE] [--out DIR] [--format json|text|both] [--strict] [--with-timestamp]
  modelmariner validate --traces FILE [--strict]
  modelmariner version

COMMANDS
  analyze   Ingest traces, compute reliability & Pareto frontiers, evaluate
            policies, simulate routing, and emit report/policy artifacts.
  validate  Ingest and validate traces only; report accepted/rejected counts.
  version   Print the compiler version.

FLAGS (analyze)
  --traces, -t   Path to a JSONL trace file (required).
  --policy, -p   Path to a JSON policy set. When omitted, only reliability and
                 Pareto analysis are produced.
  --out, -o      Directory to write report.json, report.txt, and policies.json.
  --format       Output written to stdout: json, text, or both (default both).
  --strict       Treat any invalid trace line as a fatal error.
  --with-timestamp  Include a wall-clock generation time (breaks determinism).

modelmariner is fully offline: it reasons only over recorded traces and never
contacts a model provider.
`)
}
