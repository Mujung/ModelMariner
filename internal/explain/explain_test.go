package explain

import (
	"strings"
	"testing"

	"github.com/Mujung/modelmariner/internal/policy"
	"github.com/Mujung/modelmariner/internal/routing"
)

func TestExplainWinner(t *testing.T) {
	d := routing.Decision{
		Task: "t", Policy: "p", Winner: "premium", Runnerup: "cheap",
		Score: 0.9, Margin: 0.2,
		Evaluations: []policy.Evaluation{
			{Model: "premium", Eligible: true, Score: 0.9, Components: map[string]float64{"quality": 0.9}},
		},
		Realized: routing.RealizedMetrics{Model: "premium", Calls: 10, Successes: 10, SuccessRate: 1, MeanQuality: 0.98, MeanLatencyMS: 400, TotalCostUSD: 6.0},
		Baseline: routing.RealizedMetrics{Model: "cheap", Calls: 10, MeanQuality: 0.80, TotalCostUSD: 1.0},
	}
	e := For(d)
	if !strings.Contains(e.Headline, "premium") {
		t.Errorf("headline missing winner: %q", e.Headline)
	}
	if len(e.Reasons) == 0 || !strings.Contains(strings.Join(e.Reasons, " "), "margin") {
		t.Errorf("reasons missing margin: %+v", e.Reasons)
	}
	if len(e.Evidence) < 2 {
		t.Errorf("expected replay + baseline evidence: %+v", e.Evidence)
	}
	txt := e.Text()
	if !strings.Contains(txt, "evidence:") {
		t.Errorf("text missing evidence marker:\n%s", txt)
	}
}

func TestExplainNoEligible(t *testing.T) {
	d := routing.Decision{
		Task: "t", Policy: "p", NoEligible: true,
		Rejected: []routing.RejectedCandidate{
			{Model: "m", Violations: []policy.Violation{{Constraint: "min_quality", Detail: "too low", Limit: 0.9, Observed: 0.5}}},
		},
	}
	e := For(d)
	if !strings.Contains(e.Headline, "NO eligible") {
		t.Errorf("headline should flag no eligible: %q", e.Headline)
	}
	if len(e.Rejections) != 1 || !strings.Contains(e.Rejections[0], "min_quality") {
		t.Errorf("rejection detail missing: %+v", e.Rejections)
	}
}

func TestExplainSoleSurvivor(t *testing.T) {
	d := routing.Decision{
		Task: "t", Policy: "p", Winner: "only", Score: 0.5,
		Evaluations: []policy.Evaluation{{Model: "only", Eligible: true, Score: 0.5}},
		Realized:    routing.RealizedMetrics{Model: "only", Calls: 5, MeanQuality: 0.7},
	}
	e := For(d)
	joined := strings.Join(e.Reasons, " ")
	if !strings.Contains(joined, "only model") {
		t.Errorf("expected sole-survivor reasoning: %+v", e.Reasons)
	}
}
