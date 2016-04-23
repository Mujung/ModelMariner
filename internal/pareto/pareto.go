// Package pareto computes multi-objective Pareto frontiers over model
// aggregates. For each task we want to know which models are "non-dominated":
// a model is dominated if some other model is at least as good on every axis
// (cheaper-or-equal, faster-or-equal, higher-or-equal quality, more-reliable-or-equal)
// and strictly better on at least one. The surviving set is the frontier — the
// only candidates a rational router should ever consider before applying policy.
package pareto

import (
	"math"
	"sort"

	"github.com/Mujung/modelmariner/internal/reliability"
)

// Point is one model's position in objective space for a task.
type Point struct {
	Model       string  `json:"model"`
	CostUSD     float64 `json:"cost_usd"`
	LatencyMS   float64 `json:"latency_ms"`
	Quality     float64 `json:"quality"`
	Reliability float64 `json:"reliability"`
	Dominated   bool    `json:"dominated"`
	// DominatedBy lists models that dominate this point (empty if on frontier).
	DominatedBy []string `json:"dominated_by,omitempty"`
}

// Frontier is the Pareto analysis for a single task.
type Frontier struct {
	Task     string  `json:"task"`
	Points   []Point `json:"points"`
	Frontier []Point `json:"frontier"`
}

// FrontierModels returns just the model names on the frontier.
