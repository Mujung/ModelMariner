package reliability

import (
	"math"
	"testing"

	"github.com/Mujung/modelmariner/internal/trace"
)

func mk(model, task string, cost, lat, qual float64, err bool, priv trace.PrivacyTier) trace.Trace {
	return trace.Trace{
		Model: model, Task: task, CostUSD: cost, LatencyMS: lat,
		Quality: qual, Error: err, Privacy: priv, Completion: 10, Prompt: 10,
	}
}

func TestComputeBasic(t *testing.T) {
	traces := []trace.Trace{
		mk("a", "t1", 0.10, 100, 0.9, false, trace.PrivacyPublic),
		mk("a", "t1", 0.20, 200, 0.8, false, trace.PrivacyInternal),
		mk("a", "t1", 0.00, 0, 0, true, trace.PrivacyPublic),
	}
	traces[2].ErrorKind = "timeout"
	sum := Compute(traces)
	if len(sum.Aggregates) != 1 {
		t.Fatalf("want 1 aggregate, got %d", len(sum.Aggregates))
	}
	a := sum.Aggregates[0]
	if a.Samples != 3 || a.Successes != 2 || a.Failures != 1 {
		t.Errorf("bad counts: %+v", a)
	}
	if math.Abs(a.SuccessRate-2.0/3.0) > 1e-9 {
		t.Errorf("bad success rate: %v", a.SuccessRate)
	}
	// Cost/quality means use only successes.
	if math.Abs(a.MeanCostUSD-0.15) > 1e-9 {
		t.Errorf("bad mean cost: %v", a.MeanCostUSD)
	}
