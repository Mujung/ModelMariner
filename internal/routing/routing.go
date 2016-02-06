// Package routing simulates a compiled policy against the recorded traces. For
// each task the compiler picks the winning model (the highest-scoring eligible
// candidate) and then replays every recorded trace as if the router had sent
// it to that winner, measuring realized cost, latency, quality and reliability.
// Because we only ever replay observations we already hold, the simulation is
// fully deterministic and honest: it never invents a data point.
package routing

import (
	"fmt"
	"sort"

	"github.com/Mujung/modelmariner/internal/policy"
	"github.com/Mujung/modelmariner/internal/reliability"
	"github.com/Mujung/modelmariner/internal/trace"
)

// Decision is the routing choice for a single task under a policy.
type Decision struct {
	Task        string              `json:"task"`
	Policy      string              `json:"policy"`
	Winner      string              `json:"winner"`
	Runnerup    string              `json:"runner_up,omitempty"`
	Score       float64             `json:"score"`
	Margin      float64             `json:"margin"`
	Eligible    []string            `json:"eligible"`
	Rejected    []RejectedCandidate `json:"rejected,omitempty"`
	Evaluations []policy.Evaluation `json:"evaluations"`
	Realized    RealizedMetrics     `json:"realized"`
	Baseline    RealizedMetrics     `json:"baseline"`
	// NoEligible is true when the policy disqualified every candidate.
	NoEligible bool `json:"no_eligible"`
}

// RejectedCandidate summarizes why a model was excluded.
type RejectedCandidate struct {
	Model      string             `json:"model"`
	Violations []policy.Violation `json:"violations"`
}

// RealizedMetrics reports the outcome of replaying traces through a routing
// choice. Baseline uses the naive "cheapest model regardless of policy" pick so
// the report can quantify what the policy bought or cost.
type RealizedMetrics struct {
	Model         string  `json:"model"`
	Calls         int     `json:"calls"`
	Successes     int     `json:"successes"`
	SuccessRate   float64 `json:"success_rate"`
	TotalCostUSD  float64 `json:"total_cost_usd"`
	MeanLatencyMS float64 `json:"mean_latency_ms"`
	MeanQuality   float64 `json:"mean_quality"`
}

// Simulation is the full routing result across all tasks for one policy.
type Simulation struct {
	Policy    string     `json:"policy"`
	Decisions []Decision `json:"decisions"`
	Totals    Totals     `json:"totals"`
}

// Totals aggregate realized vs baseline economics across all decided tasks.
type Totals struct {
	TasksDecided    int     `json:"tasks_decided"`
	TasksUnrouted   int     `json:"tasks_unrouted"`
	RealizedCostUSD float64 `json:"realized_cost_usd"`
	BaselineCostUSD float64 `json:"baseline_cost_usd"`
	CostDeltaUSD    float64 `json:"cost_delta_usd"`
	RealizedQuality float64 `json:"realized_mean_quality"`
	BaselineQuality float64 `json:"baseline_mean_quality"`
}

// Simulate compiles a single policy into per-task routing decisions and replays
// the recorded traces to measure realized outcomes.
func Simulate(p policy.Policy, sum reliability.Summary, traces []trace.Trace) Simulation {
	byTaskModel := indexTraces(traces)
	sim := Simulation{Policy: p.Name}

	var realizedQualitySum, baselineQualitySum float64
	var qualityWeight int

	for _, task := range sum.Tasks {
		if !p.AppliesTo(task) {
			continue
		}
		aggs := sum.ForTask(task)
		evals := policy.Evaluate(p, aggs)

		d := Decision{
			Task:        task,
			Policy:      p.Name,
			Evaluations: evals,
		}

		var eligible []policy.Evaluation
		for _, e := range evals {
			if e.Eligible {
				eligible = append(eligible, e)
				d.Eligible = append(d.Eligible, e.Model)
			} else {
				d.Rejected = append(d.Rejected, RejectedCandidate{
					Model:      e.Model,
					Violations: e.Violations,
				})
			}
		}

		if len(eligible) == 0 {
			d.NoEligible = true
			sim.Decisions = append(sim.Decisions, d)
			sim.Totals.TasksUnrouted++
			continue
		}

		winner := eligible[0]
		d.Winner = winner.Model
		d.Score = winner.Score
		if len(eligible) > 1 {
			d.Runnerup = eligible[1].Model
			d.Margin = winner.Score - eligible[1].Score
		} else {
			d.Margin = winner.Score
		}

		d.Realized = replay(task, winner.Model, byTaskModel)
		d.Baseline = replay(task, cheapestModel(aggs), byTaskModel)

		sim.Totals.TasksDecided++
		sim.Totals.RealizedCostUSD += d.Realized.TotalCostUSD
		sim.Totals.BaselineCostUSD += d.Baseline.TotalCostUSD
		realizedQualitySum += d.Realized.MeanQuality
		baselineQualitySum += d.Baseline.MeanQuality
		qualityWeight++

		sim.Decisions = append(sim.Decisions, d)
	}

	sort.Slice(sim.Decisions, func(i, j int) bool { return sim.Decisions[i].Task < sim.Decisions[j].Task })
	sim.Totals.CostDeltaUSD = sim.Totals.RealizedCostUSD - sim.Totals.BaselineCostUSD
	if qualityWeight > 0 {
		sim.Totals.RealizedQuality = realizedQualitySum / float64(qualityWeight)
		sim.Totals.BaselineQuality = baselineQualitySum / float64(qualityWeight)
	}
	return sim
}

// SimulateAll runs every policy in a set and returns simulations in order.
func SimulateAll(set policy.Set, sum reliability.Summary, traces []trace.Trace) []Simulation {
	out := make([]Simulation, 0, len(set.Policies))
	for _, p := range set.Policies {
		out = append(out, Simulate(p, sum, traces))
	}
	return out
}

// indexTraces groups traces by task then model for O(1) replay lookups.
func indexTraces(traces []trace.Trace) map[string]map[string][]trace.Trace {
	idx := map[string]map[string][]trace.Trace{}
	for _, t := range traces {
		if idx[t.Task] == nil {
			idx[t.Task] = map[string][]trace.Trace{}
		}
		idx[t.Task][t.Model] = append(idx[t.Task][t.Model], t)
	}
	return idx
}

// replay computes realized metrics by replaying every recorded trace for the
// chosen model on the given task.
func replay(task, model string, idx map[string]map[string][]trace.Trace) RealizedMetrics {
	m := RealizedMetrics{Model: model}
	traces := idx[task][model]
	var latSum, qualSum float64
	var successCount int
	for _, t := range traces {
		m.Calls++
