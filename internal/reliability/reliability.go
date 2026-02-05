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
			a.MeanCostUSD = b.costSum / float64(a.Successes)
			a.MeanQuality = b.qualSum / float64(a.Successes)
			a.MeanTokens = b.tokenSum / float64(a.Successes)
			sort.Float64s(b.latency)
			a.MeanLatencyMS = mean(b.latency)
			a.P50LatencyMS = percentile(b.latency, 0.50)
			a.P95LatencyMS = percentile(b.latency, 0.95)
			a.P99LatencyMS = percentile(b.latency, 0.99)
		}
		a.HighestPrivacy = b.privacy
		if len(b.errKinds) > 0 {
			a.ErrorKinds = b.errKinds
		}
		out.Aggregates = append(out.Aggregates, a)
	}

	sort.Slice(out.Aggregates, func(i, j int) bool {
		if out.Aggregates[i].Task != out.Aggregates[j].Task {
			return out.Aggregates[i].Task < out.Aggregates[j].Task
		}
		return out.Aggregates[i].Model < out.Aggregates[j].Model
	})

	out.Models = sortedKeys(modelSet)
	out.Tasks = sortedKeys(taskSet)
	out.TaskModels = map[string][]string{}
	for task, models := range taskModelSet {
		out.TaskModels[task] = sortedKeys(models)
	}
	return out
}

// ForTask returns aggregates belonging to a single task, in deterministic order.
func (s Summary) ForTask(task string) []Aggregate {
	var out []Aggregate
	for _, a := range s.Aggregates {
		if a.Task == task {
			out = append(out, a)
		}
	}
	return out
}

// mean returns the arithmetic mean of xs, or 0 for an empty slice.
func mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	var sum float64
	for _, x := range xs {
		sum += x
	}
	return sum / float64(len(xs))
}

// percentile returns the p-th percentile (0..1) of a sorted slice using linear
// interpolation between closest ranks. The slice MUST be sorted ascending.
func percentile(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n == 1 {
		return sorted[0]
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[n-1]
	}
	rank := p * float64(n-1)
	lo := int(math.Floor(rank))
	hi := int(math.Ceil(rank))
	if lo == hi {
		return sorted[lo]
	}
	frac := rank - float64(lo)
	return sorted[lo]*(1-frac) + sorted[hi]*frac
}

// wilsonLowerBound computes the lower bound of the Wilson score interval at a
// 95% confidence level for a binomial proportion. It rewards evidence: a model
// that succeeded 9/10 times scores lower than one that succeeded 90/100 times,
// which prevents thinly-sampled models from dominating routing decisions.
func wilsonLowerBound(successes, total int) float64 {
	if total == 0 {
		return 0
	}
	const z = 1.959963984540054 // 95% two-sided
	n := float64(total)
	phat := float64(successes) / n
	z2 := z * z
	denom := 1 + z2/n
	center := phat + z2/(2*n)
	margin := z * math.Sqrt((phat*(1-phat)+z2/(4*n))/n)
	lb := (center - margin) / denom
	if lb < 0 {
		return 0
	}
	if lb > 1 {
		return 1
	}
	return lb
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
