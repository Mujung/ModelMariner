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

