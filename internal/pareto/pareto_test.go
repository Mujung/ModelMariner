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
