package policy

import (
	"strings"
	"testing"

	"github.com/Mujung/modelmariner/internal/reliability"
	"github.com/Mujung/modelmariner/internal/trace"
)

func agg(model string, cost, p95, qual, rel float64, priv trace.PrivacyTier) reliability.Aggregate {
	return reliability.Aggregate{
		Model: model, Task: "t", Samples: 10, Successes: 10,
		MeanCostUSD: cost, P95LatencyMS: p95, MeanQuality: qual,
		WilsonLower: rel, HighestPrivacy: priv,
	}
}

func TestLoadValidPolicy(t *testing.T) {
	js := `{"policies":[{"name":"p","constraints":{"max_cost_usd":0.5},"preference":{"weights":[{"objective":"cost","weight":1}]}}]}`
	set, err := Load(strings.NewReader(js))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(set.Policies) != 1 || set.Version != 1 {
		t.Fatalf("bad set: %+v", set)
	}
}

func TestLoadRejectsInvalid(t *testing.T) {
	cases := map[string]string{
		"no policies":         `{"policies":[]}`,
		"empty name":          `{"policies":[{"name":"","preference":{"weights":[{"objective":"cost","weight":1}]}}]}`,
		"bad objective":       `{"policies":[{"name":"p","preference":{"weights":[{"objective":"speed","weight":1}]}}]}`,
		"zero weights":        `{"policies":[{"name":"p","preference":{"weights":[{"objective":"cost","weight":0}]}}]}`,
		"quality range":       `{"policies":[{"name":"p","constraints":{"min_quality":2},"preference":{"weights":[{"objective":"cost","weight":1}]}}]}`,
		"duplicate name":      `{"policies":[{"name":"p","preference":{"weights":[{"objective":"cost","weight":1}]}},{"name":"p","preference":{"weights":[{"objective":"cost","weight":1}]}}]}`,
		"duplicate objective": `{"policies":[{"name":"p","preference":{"weights":[{"objective":"cost","weight":1},{"objective":"cost","weight":1}]}}]}`,
	}
	for name, js := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Load(strings.NewReader(js)); err == nil {
				t.Errorf("expected load error for %q", name)
			}
		})
	}
}

func TestBudgetConstraintDisqualifies(t *testing.T) {
	p := Policy{
		Name:        "budget",
		Constraints: Constraints{MaxCostUSD: 0.30},
		Preference:  Preference{Weights: []Weight{{ObjCost, 1}}},
	}
	aggs := []reliability.Aggregate{
		agg("cheap", 0.10, 100, 0.9, 0.95, 0),
		agg("pricey", 0.50, 100, 0.9, 0.95, 0),
	}
	evals := Evaluate(p, aggs)
	byModel := map[string]Evaluation{}
	for _, e := range evals {
		byModel[e.Model] = e
	}
	if !byModel["cheap"].Eligible {
		t.Error("cheap should be eligible")
	}
	if byModel["pricey"].Eligible {
		t.Error("pricey should be disqualified by budget")
	}
	if len(byModel["pricey"].Violations) == 0 || byModel["pricey"].Violations[0].Constraint != "max_cost_usd" {
		t.Errorf("expected max_cost_usd violation: %+v", byModel["pricey"].Violations)
	}
}

func TestPrivacyConstraintAndSafeList(t *testing.T) {
	restricted := trace.PrivacyRestricted
	internal := trace.PrivacyInternal
	p := Policy{
		Name: "privacy",
		Constraints: Constraints{
			MaxPrivacy:        &internal,
			PrivacySafeModels: []string{"onprem"},
		},
		Preference: Preference{Weights: []Weight{{ObjQuality, 1}}},
	}
	aggs := []reliability.Aggregate{
		agg("cloud", 0.1, 100, 0.9, 0.9, restricted),  // exceeds cap, not safe
		agg("onprem", 0.2, 200, 0.8, 0.9, restricted), // exceeds cap but cleared
	}
	evals := Evaluate(p, aggs)
