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
		for _, r := range d.Rejected {
			e.Rejections = append(e.Rejections, rejectionLine(r))
		}
		return e
	}

	e.Headline = fmt.Sprintf("route %q to %s (score %.4f)", d.Task, d.Winner, d.Score)

	if d.Runnerup != "" {
		e.Reasons = append(e.Reasons, fmt.Sprintf(
			"%s beat runner-up %s by a margin of %.4f in weighted score",
			d.Winner, d.Runnerup, d.Margin))
	} else {
		e.Reasons = append(e.Reasons, fmt.Sprintf(
			"%s was the only model to satisfy every hard constraint", d.Winner))
	}

	// Surface the winner's score component breakdown.
	for _, ev := range d.Evaluations {
		if ev.Model == d.Winner && len(ev.Components) > 0 {
			e.Reasons = append(e.Reasons, "score components: "+componentBreakdown(ev.Components))
		}
	}

	r := d.Realized
	e.Evidence = append(e.Evidence, fmt.Sprintf(
		"replayed %d recorded call(s): %.1f%% success, mean quality %.3f, mean latency %.0f ms, total cost $%.4f",
		r.Calls, r.SuccessRate*100, r.MeanQuality, r.MeanLatencyMS, r.TotalCostUSD))

	b := d.Baseline
	if b.Model != "" && b.Model != r.Model {
		delta := r.TotalCostUSD - b.TotalCostUSD
		verb := "more"
		if delta < 0 {
			verb = "less"
			delta = -delta
		}
		e.Evidence = append(e.Evidence, fmt.Sprintf(
			"versus cheapest-model baseline %s: $%.4f %s cost, %.3f vs %.3f mean quality",
			b.Model, delta, verb, r.MeanQuality, b.MeanQuality))
	}

	for _, rej := range d.Rejected {
		e.Rejections = append(e.Rejections, rejectionLine(rej))
	}
	return e
}

// Text renders the explanation as a readable block.
func (e Explanation) Text() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "• %s\n", e.Headline)
	for _, r := range e.Reasons {
		fmt.Fprintf(&sb, "    - %s\n", r)
	}
	for _, ev := range e.Evidence {
		fmt.Fprintf(&sb, "    evidence: %s\n", ev)
	}
