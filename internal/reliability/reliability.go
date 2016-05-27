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

