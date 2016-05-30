// Package reliability aggregates raw traces into per-model, per-task summaries
// with statistically meaningful reliability metrics: success rate, latency
// percentiles, cost distribution, and mean quality. These aggregates are the
// substrate the Pareto and routing engines reason over.
package reliability

import (
	"math"
	"sort"

	"github.com/Mujung/modelmariner/internal/trace"
)

// Aggregate summarizes every trace observed for a single (model, task) pair.
type Aggregate struct {
	Model string `json:"model"`
	Task  string `json:"task"`

	Samples     int     `json:"samples"`
	Successes   int     `json:"successes"`
	Failures    int     `json:"failures"`
	SuccessRate float64 `json:"success_rate"`

	// Wilson lower bound gives a conservative reliability estimate that does
	// not over-trust models with only a handful of samples.
	WilsonLower float64 `json:"reliability_lower"`

	MeanCostUSD   float64 `json:"mean_cost_usd"`
	MeanLatencyMS float64 `json:"mean_latency_ms"`
	P50LatencyMS  float64 `json:"p50_latency_ms"`
	P95LatencyMS  float64 `json:"p95_latency_ms"`
	P99LatencyMS  float64 `json:"p99_latency_ms"`
	MeanQuality   float64 `json:"mean_quality"`
	MeanTokens    float64 `json:"mean_tokens"`

	// HighestPrivacy is the strictest privacy tier ever observed for this task,
	// which routing uses to reason about what a candidate must be able to handle.
	HighestPrivacy trace.PrivacyTier `json:"highest_privacy"`

	// ErrorKinds counts each distinct failure category.
	ErrorKinds map[string]int `json:"error_kinds,omitempty"`
}

// Summary is the complete reliability view across all models and tasks.
type Summary struct {
	Aggregates []Aggregate         `json:"aggregates"`
	Models     []string            `json:"models"`
	Tasks      []string            `json:"tasks"`
	TaskModels map[string][]string `json:"-"`
}

// key uniquely identifies a model/task pairing.
type key struct {
	model string
	task  string
}

// Compute folds a slice of validated traces into a deterministic Summary.
func Compute(traces []trace.Trace) Summary {
	type bucket struct {
		agg      Aggregate
		latency  []float64
		costSum  float64
		qualSum  float64
		tokenSum float64
		privacy  trace.PrivacyTier
		errKinds map[string]int
	}
	buckets := map[key]*bucket{}
	modelSet := map[string]struct{}{}
	taskSet := map[string]struct{}{}
	taskModelSet := map[string]map[string]struct{}{}

	for _, t := range traces {
		k := key{t.Model, t.Task}
		b := buckets[k]
		if b == nil {
			b = &bucket{
				agg:      Aggregate{Model: t.Model, Task: t.Task},
				errKinds: map[string]int{},
			}
			buckets[k] = b
		}
		b.agg.Samples++
		if t.Error {
			b.agg.Failures++
			if t.ErrorKind != "" {
				b.errKinds[t.ErrorKind]++
			}
		} else {
			b.agg.Successes++
			// Only successful calls contribute to quality/latency/cost
			// distributions, because a failed call's metrics are not
			// representative of the service the model actually delivers.
			b.latency = append(b.latency, t.LatencyMS)
			b.costSum += t.CostUSD
			b.qualSum += t.Quality
			b.tokenSum += float64(t.Tokens())
		}
		if t.Privacy > b.privacy {
			b.privacy = t.Privacy
		}
		modelSet[t.Model] = struct{}{}
		taskSet[t.Task] = struct{}{}
		if taskModelSet[t.Task] == nil {
			taskModelSet[t.Task] = map[string]struct{}{}
		}
		taskModelSet[t.Task][t.Model] = struct{}{}
	}

	var out Summary
	for _, b := range buckets {
		a := b.agg
		if a.Samples > 0 {
			a.SuccessRate = float64(a.Successes) / float64(a.Samples)
			a.WilsonLower = wilsonLowerBound(a.Successes, a.Samples)
		}
		if a.Successes > 0 {
