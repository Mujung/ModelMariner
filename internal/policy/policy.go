// Package policy defines the routing-policy language and its evaluator. A
// policy is a set of HARD constraints (budget, latency, privacy, reliability)
// plus a preference expressing how to rank the survivors. Constraints are hard:
// a candidate that violates any constraint is disqualified, never merely
// penalized. This makes the compiler's decisions auditable and safe for
// regulated workloads.
package policy

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/Mujung/modelmariner/internal/reliability"
	"github.com/Mujung/modelmariner/internal/trace"
)

// Objective names a rankable dimension used by preferences.
type Objective string

const (
	ObjCost        Objective = "cost"
	ObjLatency     Objective = "latency"
	ObjQuality     Objective = "quality"
	ObjReliability Objective = "reliability"
)

var validObjectives = map[Objective]bool{
	ObjCost: true, ObjLatency: true, ObjQuality: true, ObjReliability: true,
}

// Constraints are the hard limits a candidate must satisfy.
type Constraints struct {
	// MaxCostUSD caps mean cost per call. Zero means "no cap".
	MaxCostUSD float64 `json:"max_cost_usd,omitempty"`
	// MaxLatencyMS caps the P95 latency. Zero means "no cap".
	MaxLatencyMS float64 `json:"max_latency_ms,omitempty"`
	// MinQuality is the minimum acceptable mean quality. Zero means "no floor".
