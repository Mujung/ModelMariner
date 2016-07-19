// Package report assembles the outputs of every engine stage into a single
// deterministic document that both the text renderer and the TypeScript
// dashboard consume. Determinism is a first-class requirement: given identical
// inputs the JSON bytes are byte-for-byte identical, which makes reports safe to
// commit, diff, and gate CI on.
package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Mujung/modelmariner/internal/explain"
	"github.com/Mujung/modelmariner/internal/pareto"
	"github.com/Mujung/modelmariner/internal/policy"
	"github.com/Mujung/modelmariner/internal/reliability"
	"github.com/Mujung/modelmariner/internal/routing"
	"github.com/Mujung/modelmariner/internal/trace"
)

// SchemaVersion identifies the report format so the dashboard can guard against
// consuming an incompatible document.
const SchemaVersion = "modelmariner/v1"

// Report is the complete, self-describing analysis document.
type Report struct {
	Schema      string                  `json:"schema"`
	Generated   string                  `json:"generated,omitempty"`
	Input       InputSummary            `json:"input"`
	Reliability []reliability.Aggregate `json:"reliability"`
	Pareto      []pareto.Frontier       `json:"pareto"`
	Policies    []PolicyReport          `json:"policies"`
}

// InputSummary describes what was ingested.
type InputSummary struct {
	TotalLines int      `json:"total_lines"`
	Accepted   int      `json:"accepted"`
	Rejected   int      `json:"rejected"`
	Models     []string `json:"models"`
	Tasks      []string `json:"tasks"`
	Warnings   []string `json:"warnings,omitempty"`
}

// PolicyReport bundles a simulation with its explanations.
type PolicyReport struct {
	Name         string                `json:"name"`
	Description  string                `json:"description,omitempty"`
	Simulation   routing.Simulation    `json:"simulation"`
	Explanations []explain.Explanation `json:"explanations"`
}

// BuildInput carries everything needed to assemble a report.
type BuildInput struct {
	Ingest      trace.IngestResult
	Traces      []trace.Trace
	Reliability reliability.Summary
	Pareto      pareto.Analysis
	Policies    policy.Set
	Simulations []routing.Simulation
	// Deterministic omits the wall-clock timestamp so output is reproducible.
	Deterministic bool
	Now           time.Time
}

// Build assembles a Report from all engine outputs.
func Build(in BuildInput) Report {
	r := Report{
		Schema: SchemaVersion,
		Input: InputSummary{
			TotalLines: in.Ingest.Total,
			Accepted:   len(in.Traces),
			Rejected:   in.Ingest.Rejected,
			Models:     in.Reliability.Models,
			Tasks:      in.Reliability.Tasks,
		},
		Reliability: in.Reliability.Aggregates,
		Pareto:      in.Pareto.Frontiers,
	}
	if !in.Deterministic {
		r.Generated = in.Now.UTC().Format(time.RFC3339)
	}
	for _, ve := range in.Ingest.Errors {
		r.Input.Warnings = append(r.Input.Warnings, ve.Error())
	}
	sort.Strings(r.Input.Warnings)

	byName := map[string]policy.Policy{}
	for _, p := range in.Policies.Policies {
		byName[p.Name] = p
	}
	for _, sim := range in.Simulations {
		pr := PolicyReport{
			Name:         sim.Policy,
			Description:  byName[sim.Policy].Description,
			Simulation:   sim,
			Explanations: explain.All(sim),
		}
		r.Policies = append(r.Policies, pr)
	}
	sort.Slice(r.Policies, func(i, j int) bool { return r.Policies[i].Name < r.Policies[j].Name })
	return r
}

// JSON renders the report as indented, deterministic JSON. HTML escaping is
// disabled so operators (comparison symbols) render as themselves in diffs.
func (r Report) JSON() ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r); err != nil {
		return nil, fmt.Errorf("encoding report: %w", err)
	}
	return buf.Bytes(), nil
}

// Text renders a human-readable report suitable for a terminal or a log.
func (r Report) Text() string {
	var sb strings.Builder
	line := strings.Repeat("=", 72)
	fmt.Fprintf(&sb, "%s\n MODELMARINER — offline routing-policy compilation report\n%s\n", line, line)
	fmt.Fprintf(&sb, "Schema: %s\n", r.Schema)
	if r.Generated != "" {
		fmt.Fprintf(&sb, "Generated: %s\n", r.Generated)
	}
	fmt.Fprintf(&sb, "Ingested: %d line(s) — %d accepted, %d rejected\n",
		r.Input.TotalLines, r.Input.Accepted, r.Input.Rejected)
	fmt.Fprintf(&sb, "Models: %s\n", strings.Join(r.Input.Models, ", "))
	fmt.Fprintf(&sb, "Tasks:  %s\n", strings.Join(r.Input.Tasks, ", "))
	if len(r.Input.Warnings) > 0 {
		fmt.Fprintf(&sb, "Warnings: %d rejected line(s)\n", len(r.Input.Warnings))
	}

	fmt.Fprintf(&sb, "\n%s\n RELIABILITY (per model / task)\n%s\n", line, line)
	fmt.Fprintf(&sb, "%-18s %-14s %6s %8s %10s %9s %8s\n",
		"task", "model", "n", "success", "reliab.LB", "p95 ms", "quality")
	for _, a := range r.Reliability {
		fmt.Fprintf(&sb, "%-18s %-14s %6d %7.1f%% %10.3f %9.0f %8.3f\n",
			trunc(a.Task, 18), trunc(a.Model, 14), a.Samples,
			a.SuccessRate*100, a.WilsonLower, a.P95LatencyMS, a.MeanQuality)
	}

	fmt.Fprintf(&sb, "\n%s\n PARETO FRONTIERS (non-dominated models per task)\n%s\n", line, line)
	for _, f := range r.Pareto {
		names := f.FrontierModels()
