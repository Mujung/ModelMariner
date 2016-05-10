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
func (f Frontier) FrontierModels() []string {
	out := make([]string, 0, len(f.Frontier))
	for _, p := range f.Frontier {
		out = append(out, p.Model)
	}
	return out
}

// Analysis holds frontiers for every task.
type Analysis struct {
	Frontiers []Frontier `json:"frontiers"`
}

// ForTask returns the frontier for a named task and whether it exists.
func (a Analysis) ForTask(task string) (Frontier, bool) {
	for _, f := range a.Frontiers {
		if f.Task == task {
			return f, true
		}
	}
	return Frontier{}, false
}

// Compute derives Pareto frontiers per task from a reliability summary. Only
// models with at least one successful sample participate; a model that never
// succeeded has no meaningful cost/latency/quality position.
func Compute(sum reliability.Summary) Analysis {
	var an Analysis
	for _, task := range sum.Tasks {
		aggs := sum.ForTask(task)
		points := make([]Point, 0, len(aggs))
		for _, a := range aggs {
			if a.Successes == 0 {
				continue
			}
			points = append(points, Point{
				Model:       a.Model,
				CostUSD:     a.MeanCostUSD,
				LatencyMS:   a.P95LatencyMS,
				Quality:     a.MeanQuality,
				Reliability: a.WilsonLower,
			})
		}
		if len(points) == 0 {
			continue
		}
		markDominance(points)
		frontier := make([]Point, 0, len(points))
		for _, p := range points {
			if !p.Dominated {
				frontier = append(frontier, p)
			}
		}
		// Sort frontier by ascending cost then descending quality for stable,
		// human-friendly presentation.
		sort.Slice(frontier, func(i, j int) bool {
			if !almostEqual(frontier[i].CostUSD, frontier[j].CostUSD) {
				return frontier[i].CostUSD < frontier[j].CostUSD
			}
			return frontier[i].Quality > frontier[j].Quality
		})
		sort.Slice(points, func(i, j int) bool { return points[i].Model < points[j].Model })
		an.Frontiers = append(an.Frontiers, Frontier{
