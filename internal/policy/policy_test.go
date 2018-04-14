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
