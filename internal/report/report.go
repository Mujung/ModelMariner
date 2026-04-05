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
		fmt.Fprintf(&sb, "%-18s frontier: %s\n", trunc(f.Task, 18), strings.Join(names, ", "))
		for _, p := range f.Points {
			flag := "  on-frontier"
			if p.Dominated {
				flag = "  dominated by " + strings.Join(p.DominatedBy, "/")
			}
			fmt.Fprintf(&sb, "    %-14s cost $%.4f  p95 %.0fms  q %.3f  rel %.3f%s\n",
				trunc(p.Model, 14), p.CostUSD, p.LatencyMS, p.Quality, p.Reliability, flag)
		}
	}

	for _, pr := range r.Policies {
		fmt.Fprintf(&sb, "\n%s\n POLICY: %s\n%s\n", line, pr.Name, line)
		if pr.Description != "" {
			fmt.Fprintf(&sb, "%s\n\n", pr.Description)
		}
		t := pr.Simulation.Totals
		fmt.Fprintf(&sb, "Decided %d task(s), %d unrouted. ",
			t.TasksDecided, t.TasksUnrouted)
		fmt.Fprintf(&sb, "Realized cost $%.4f vs baseline $%.4f (delta $%+.4f). ",
			t.RealizedCostUSD, t.BaselineCostUSD, t.CostDeltaUSD)
		fmt.Fprintf(&sb, "Quality %.3f vs %.3f.\n\n", t.RealizedQuality, t.BaselineQuality)
		for _, e := range pr.Explanations {
			sb.WriteString(e.Text())
		}
	}
	return sb.String()
}

// PolicyArtifact is the compiled, standalone routing table that a runtime could
// load to make decisions. It contains only the essential winner-per-task map
// plus provenance so it can be regenerated and verified.
type PolicyArtifact struct {
	Schema   string            `json:"schema"`
	Policy   string            `json:"policy"`
	Routes   []Route           `json:"routes"`
	Fallback map[string]string `json:"unrouted,omitempty"`
}

// Route is a single compiled winner for a task.
type Route struct {
	Task   string  `json:"task"`
	Model  string  `json:"model"`
	Score  float64 `json:"score"`
	Margin float64 `json:"margin"`
}

// CompilePolicyArtifacts extracts standalone routing tables from simulations.
func CompilePolicyArtifacts(sims []routing.Simulation) []PolicyArtifact {
	out := make([]PolicyArtifact, 0, len(sims))
	for _, sim := range sims {
		art := PolicyArtifact{Schema: SchemaVersion, Policy: sim.Policy}
		for _, d := range sim.Decisions {
			if d.NoEligible {
				if art.Fallback == nil {
					art.Fallback = map[string]string{}
				}
				art.Fallback[d.Task] = "no eligible model under policy constraints"
				continue
			}
			art.Routes = append(art.Routes, Route{
				Task:   d.Task,
				Model:  d.Winner,
				Score:  d.Score,
				Margin: d.Margin,
			})
		}
		sort.Slice(art.Routes, func(i, j int) bool { return art.Routes[i].Task < art.Routes[j].Task })
		out = append(out, art)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Policy < out[j].Policy })
	return out
}

// ArtifactsJSON renders compiled policy artifacts as deterministic JSON.
func ArtifactsJSON(arts []PolicyArtifact) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(arts); err != nil {
		return nil, fmt.Errorf("encoding artifacts: %w", err)
	}
	return buf.Bytes(), nil
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}
