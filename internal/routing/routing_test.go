package routing

import (
	"testing"

	"github.com/Mujung/modelmariner/internal/policy"
	"github.com/Mujung/modelmariner/internal/reliability"
	"github.com/Mujung/modelmariner/internal/trace"
)

func tr(model, task string, cost, lat, qual float64, err bool) trace.Trace {
	return trace.Trace{Model: model, Task: task, CostUSD: cost, LatencyMS: lat, Quality: qual, Error: err, Prompt: 10, Completion: 10}
}

func buildTraces() []trace.Trace {
	var out []trace.Trace
	for i := 0; i < 10; i++ {
		out = append(out, tr("cheap", "t", 0.10, 100, 0.80, false))
		out = append(out, tr("premium", "t", 0.60, 400, 0.98, false))
	}
	return out
}

func TestSimulatePicksWinnerAndReplays(t *testing.T) {
	traces := buildTraces()
	sum := reliability.Compute(traces)
	p := policy.Policy{
		Name:       "quality",
		Preference: policy.Preference{Weights: []policy.Weight{{Objective: policy.ObjQuality, Weight: 1}}},
	}
	sim := Simulate(p, sum, traces)
	if len(sim.Decisions) != 1 {
		t.Fatalf("want 1 decision, got %d", len(sim.Decisions))
	}
	d := sim.Decisions[0]
	if d.Winner != "premium" {
		t.Errorf("quality-weighted policy should pick premium, got %s", d.Winner)
	}
	if d.Realized.Calls != 10 {
		t.Errorf("should replay 10 recorded calls, got %d", d.Realized.Calls)
	}
	// Baseline is the cheapest model.
	if d.Baseline.Model != "cheap" {
		t.Errorf("baseline should be cheapest model, got %s", d.Baseline.Model)
	}
	if d.Realized.TotalCostUSD <= d.Baseline.TotalCostUSD {
		t.Error("premium realized cost should exceed cheap baseline")
	}
}

func TestSimulateBudgetForcesCheaperWinner(t *testing.T) {
	traces := buildTraces()
	sum := reliability.Compute(traces)
	p := policy.Policy{
		Name:        "budget",
		Constraints: policy.Constraints{MaxCostUSD: 0.30},
		Preference:  policy.Preference{Weights: []policy.Weight{{Objective: policy.ObjQuality, Weight: 1}}},
	}
	sim := Simulate(p, sum, traces)
	d := sim.Decisions[0]
	if d.Winner != "cheap" {
		t.Errorf("budget should force cheap winner even though quality prefers premium, got %s", d.Winner)
	}
	// premium must be recorded as rejected.
	found := false
	for _, r := range d.Rejected {
		if r.Model == "premium" {
