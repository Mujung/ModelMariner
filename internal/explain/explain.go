// Package explain turns routing decisions into human-readable rationale. Every
// selection the compiler makes can be traced back to concrete numbers: which
// candidates survived the hard constraints, why the winner outscored the field,
// and which recorded evidence backs the pick. Good explanations are what make a
// routing compiler trustworthy rather than a black box.
package explain

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Mujung/modelmariner/internal/routing"
)

// Explanation is a structured rationale for a single decision.
type Explanation struct {
	Task       string   `json:"task"`
	Policy     string   `json:"policy"`
	Headline   string   `json:"headline"`
	Reasons    []string `json:"reasons"`
	Evidence   []string `json:"evidence"`
	Rejections []string `json:"rejections,omitempty"`
}

// For builds a structured explanation for one decision.
func For(d routing.Decision) Explanation {
	e := Explanation{Task: d.Task, Policy: d.Policy}
	if d.NoEligible {
		e.Headline = fmt.Sprintf("task %q has NO eligible model under policy %q", d.Task, d.Policy)
		e.Reasons = append(e.Reasons, "every candidate violated at least one hard constraint")
