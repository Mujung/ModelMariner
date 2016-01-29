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
