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

