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
