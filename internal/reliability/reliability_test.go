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
	if a.HighestPrivacy != trace.PrivacyInternal {
		t.Errorf("bad highest privacy: %v", a.HighestPrivacy)
	}
	if a.ErrorKinds["timeout"] != 1 {
		t.Errorf("error kind not counted: %+v", a.ErrorKinds)
	}
}

func TestWilsonLowerBoundRewardsEvidence(t *testing.T) {
	small := wilsonLowerBound(9, 10)
	large := wilsonLowerBound(90, 100)
	if !(large > small) {
		t.Errorf("more evidence should raise the lower bound: 9/10=%.4f 90/100=%.4f", small, large)
	}
	if wilsonLowerBound(0, 0) != 0 {
		t.Error("zero samples must yield 0")
	}
	if lb := wilsonLowerBound(10, 10); lb <= 0 || lb > 1 {
		t.Errorf("perfect record out of range: %v", lb)
	}
}

func TestPercentileInterpolation(t *testing.T) {
	xs := []float64{10, 20, 30, 40}
	if p := percentile(xs, 0.5); math.Abs(p-25) > 1e-9 {
		t.Errorf("p50 want 25, got %v", p)
	}
	if p := percentile(xs, 0); p != 10 {
		t.Errorf("p0 want 10, got %v", p)
	}
	if p := percentile(xs, 1); p != 40 {
		t.Errorf("p100 want 40, got %v", p)
	}
	if p := percentile(nil, 0.5); p != 0 {
		t.Errorf("empty want 0, got %v", p)
	}
}

func TestForTaskFiltersDeterministically(t *testing.T) {
	traces := []trace.Trace{
		mk("b", "t1", 0.1, 100, 0.9, false, 0),
		mk("a", "t1", 0.1, 100, 0.9, false, 0),
		mk("a", "t2", 0.1, 100, 0.9, false, 0),
	}
	sum := Compute(traces)
	t1 := sum.ForTask("t1")
	if len(t1) != 2 {
		t.Fatalf("want 2 aggregates for t1, got %d", len(t1))
	}
	if t1[0].Model != "a" || t1[1].Model != "b" {
		t.Errorf("aggregates not sorted by model: %v %v", t1[0].Model, t1[1].Model)
	}
}
