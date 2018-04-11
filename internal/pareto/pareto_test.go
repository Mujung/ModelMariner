package pareto

import (
	"testing"

	"github.com/Mujung/modelmariner/internal/reliability"
)

func agg(model string, cost, p95, qual, rel float64) reliability.Aggregate {
	return reliability.Aggregate{
		Model: model, Task: "t", Successes: 10, Samples: 10,
		MeanCostUSD: cost, P95LatencyMS: p95, MeanQuality: qual, WilsonLower: rel,
	}
}

func TestDominatedPointRemoved(t *testing.T) {
	// "cheap" dominates "worse" on every axis; "premium" trades cost for quality.
	sum := reliability.Summary{
		Tasks: []string{"t"},
		Aggregates: []reliability.Aggregate{
			agg("cheap", 0.10, 100, 0.90, 0.95),
			agg("worse", 0.20, 200, 0.80, 0.90),   // dominated by cheap
			agg("premium", 0.50, 300, 0.99, 0.99), // higher quality, not dominated
		},
	}
	an := Compute(sum)
	f, ok := an.ForTask("t")
	if !ok {
		t.Fatal("no frontier for task t")
	}
	models := f.FrontierModels()
	if contains(models, "worse") {
		t.Errorf("dominated model should not be on frontier: %v", models)
	}
	if !contains(models, "cheap") || !contains(models, "premium") {
		t.Errorf("expected cheap and premium on frontier: %v", models)
	}
	// Verify DominatedBy is populated for the dominated point.
	for _, p := range f.Points {
		if p.Model == "worse" {
			if !p.Dominated || !contains(p.DominatedBy, "cheap") {
				t.Errorf("worse should be dominated by cheap: %+v", p)
			}
		}
	}
}

func TestSkipsZeroSuccessModels(t *testing.T) {
	a := agg("dead", 0.1, 100, 0.9, 0.9)
	a.Successes = 0
	sum := reliability.Summary{
		Tasks:      []string{"t"},
		Aggregates: []reliability.Aggregate{a, agg("live", 0.2, 200, 0.8, 0.8)},
	}
	an := Compute(sum)
	f, _ := an.ForTask("t")
	if len(f.Points) != 1 || f.Points[0].Model != "live" {
		t.Errorf("zero-success model should be excluded: %+v", f.Points)
	}
}

func TestNoModelsProducesNoFrontier(t *testing.T) {
	sum := reliability.Summary{Tasks: []string{"empty"}}
	an := Compute(sum)
	if _, ok := an.ForTask("empty"); ok {
