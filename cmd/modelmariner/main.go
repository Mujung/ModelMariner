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
