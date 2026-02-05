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
	MinQuality float64 `json:"min_quality,omitempty"`
	// MinReliability is the minimum acceptable Wilson lower bound. Zero means none.
	MinReliability float64 `json:"min_reliability,omitempty"`
	// MaxPrivacy is the strictest privacy tier a candidate may handle. When set,
	// a candidate is disqualified if the task's observed privacy tier exceeds it
	// unless the model is on AllowPrivacy. Empty means "no privacy constraint".
	MaxPrivacy *trace.PrivacyTier `json:"max_privacy,omitempty"`
	// DenyModels lists models forbidden regardless of metrics (e.g. deprecated).
	DenyModels []string `json:"deny_models,omitempty"`
	// AllowModels, when non-empty, restricts routing to exactly these models.
	AllowModels []string `json:"allow_models,omitempty"`
	// PrivacySafeModels lists models cleared to handle data above MaxPrivacy
	// (for example on-premise deployments that never egress data).
	PrivacySafeModels []string `json:"privacy_safe_models,omitempty"`
}

// Weight assigns relative importance to an objective for tie-breaking / scoring.
type Weight struct {
	Objective Objective `json:"objective"`
	Weight    float64   `json:"weight"`
}

// Preference declares how survivors are ranked. Weights are normalized before
// scoring, so their absolute magnitudes do not matter — only their ratios.
type Preference struct {
	Weights []Weight `json:"weights"`
}

// Policy is a named bundle of constraints and a preference, optionally scoped
// to specific tasks. A policy with no Tasks applies to every task.
type Policy struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Tasks       []string    `json:"tasks,omitempty"`
	Constraints Constraints `json:"constraints"`
	Preference  Preference  `json:"preference"`
}

// Set is a collection of policies loaded from a policy file.
type Set struct {
	Version  int      `json:"version"`
	Policies []Policy `json:"policies"`
}

// AppliesTo reports whether the policy governs the given task.
func (p Policy) AppliesTo(task string) bool {
	if len(p.Tasks) == 0 {
		return true
	}
	for _, t := range p.Tasks {
		if t == task {
			return true
		}
	}
	return false
}

// Load parses and validates a policy set from JSON.
func Load(r io.Reader) (Set, error) {
	var set Set
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&set); err != nil {
		return set, fmt.Errorf("parsing policy file: %w", err)
	}
	if set.Version == 0 {
		set.Version = 1
	}
	if len(set.Policies) == 0 {
		return set, fmt.Errorf("policy file defines no policies")
	}
	names := map[string]bool{}
	for i := range set.Policies {
		p := &set.Policies[i]
		p.Name = strings.TrimSpace(p.Name)
		if p.Name == "" {
			return set, fmt.Errorf("policy #%d has an empty name", i+1)
		}
		if names[p.Name] {
			return set, fmt.Errorf("duplicate policy name %q", p.Name)
		}
		names[p.Name] = true
		if err := validatePreference(p.Preference); err != nil {
			return set, fmt.Errorf("policy %q: %w", p.Name, err)
		}
		if err := validateConstraints(p.Constraints); err != nil {
			return set, fmt.Errorf("policy %q: %w", p.Name, err)
		}
	}
	return set, nil
}

func validatePreference(pref Preference) error {
	if len(pref.Weights) == 0 {
		return fmt.Errorf("preference must define at least one weight")
	}
	seen := map[Objective]bool{}
	var total float64
	for _, w := range pref.Weights {
		if !validObjectives[w.Objective] {
			return fmt.Errorf("unknown objective %q (want cost, latency, quality, reliability)", w.Objective)
		}
		if seen[w.Objective] {
			return fmt.Errorf("objective %q weighted more than once", w.Objective)
		}
		seen[w.Objective] = true
		if w.Weight < 0 {
			return fmt.Errorf("objective %q has negative weight", w.Objective)
		}
		total += w.Weight
	}
	if total <= 0 {
		return fmt.Errorf("preference weights sum to zero")
	}
	return nil
}

func validateConstraints(c Constraints) error {
	if c.MaxCostUSD < 0 {
		return fmt.Errorf("max_cost_usd must be >= 0")
	}
	if c.MaxLatencyMS < 0 {
		return fmt.Errorf("max_latency_ms must be >= 0")
	}
	if c.MinQuality < 0 || c.MinQuality > 1 {
		return fmt.Errorf("min_quality must be in [0,1]")
	}
	if c.MinReliability < 0 || c.MinReliability > 1 {
		return fmt.Errorf("min_reliability must be in [0,1]")
	}
	return nil
}

// Violation records a single reason a candidate failed a constraint.
type Violation struct {
	Constraint string  `json:"constraint"`
	Detail     string  `json:"detail"`
	Limit      float64 `json:"limit,omitempty"`
	Observed   float64 `json:"observed,omitempty"`
}

// Evaluation is the verdict for one candidate model under one policy.
type Evaluation struct {
	Model      string      `json:"model"`
	Eligible   bool        `json:"eligible"`
	Score      float64     `json:"score"`
	Violations []Violation `json:"violations,omitempty"`
	// Components exposes the normalized per-objective contributions to Score.
	Components map[string]float64 `json:"components,omitempty"`
}

// NormalizedWeights returns the preference weights scaled to sum to 1.
func (p Preference) NormalizedWeights() map[Objective]float64 {
	total := 0.0
	for _, w := range p.Weights {
		total += w.Weight
	}
	out := map[Objective]float64{}
	if total <= 0 {
		return out
	}
	for _, w := range p.Weights {
		out[w.Objective] = w.Weight / total
	}
	return out
}

// Evaluate applies a policy to every candidate aggregate for a task and returns
// per-model verdicts sorted by score (descending) then model name. Candidates
// with no successful samples are always ineligible.
func Evaluate(p Policy, aggs []reliability.Aggregate) []Evaluation {
	deny := toSet(p.Constraints.DenyModels)
	var allow map[string]bool
	if len(p.Constraints.AllowModels) > 0 {
		allow = toSet(p.Constraints.AllowModels)
	}
	safe := toSet(p.Constraints.PrivacySafeModels)

	// Compute scaling extents for score normalization across the candidate set.
	ranges := computeRanges(aggs)

	out := make([]Evaluation, 0, len(aggs))
	for _, a := range aggs {
		ev := Evaluation{Model: a.Model, Eligible: true}
		c := p.Constraints

		if a.Successes == 0 {
			ev.Eligible = false
			ev.Violations = append(ev.Violations, Violation{
				Constraint: "samples",
				Detail:     "no successful samples recorded for this model/task",
			})
		}
		if deny[a.Model] {
			ev.Eligible = false
			ev.Violations = append(ev.Violations, Violation{
				Constraint: "deny_models",
				Detail:     fmt.Sprintf("model %q is explicitly denied", a.Model),
			})
		}
		if allow != nil && !allow[a.Model] {
			ev.Eligible = false
			ev.Violations = append(ev.Violations, Violation{
				Constraint: "allow_models",
				Detail:     fmt.Sprintf("model %q is not in the allow list", a.Model),
			})
		}
		if c.MaxCostUSD > 0 && a.MeanCostUSD > c.MaxCostUSD+1e-9 {
			ev.Eligible = false
			ev.Violations = append(ev.Violations, Violation{
				Constraint: "max_cost_usd",
				Detail:     "mean cost exceeds budget",
				Limit:      c.MaxCostUSD,
				Observed:   a.MeanCostUSD,
			})
		}
		if c.MaxLatencyMS > 0 && a.P95LatencyMS > c.MaxLatencyMS+1e-9 {
			ev.Eligible = false
			ev.Violations = append(ev.Violations, Violation{
				Constraint: "max_latency_ms",
				Detail:     "p95 latency exceeds ceiling",
				Limit:      c.MaxLatencyMS,
				Observed:   a.P95LatencyMS,
			})
		}
		if c.MinQuality > 0 && a.MeanQuality < c.MinQuality-1e-9 {
			ev.Eligible = false
			ev.Violations = append(ev.Violations, Violation{
				Constraint: "min_quality",
				Detail:     "mean quality below floor",
				Limit:      c.MinQuality,
				Observed:   a.MeanQuality,
			})
		}
		if c.MinReliability > 0 && a.WilsonLower < c.MinReliability-1e-9 {
			ev.Eligible = false
			ev.Violations = append(ev.Violations, Violation{
				Constraint: "min_reliability",
				Detail:     "reliability lower bound below floor",
				Limit:      c.MinReliability,
				Observed:   a.WilsonLower,
			})
		}
		if c.MaxPrivacy != nil && a.HighestPrivacy > *c.MaxPrivacy && !safe[a.Model] {
			ev.Eligible = false
			ev.Violations = append(ev.Violations, Violation{
				Constraint: "max_privacy",
				Detail: fmt.Sprintf("task data reaches %q but policy caps at %q and model is not privacy-safe",
					a.HighestPrivacy, c.MaxPrivacy.String()),
			})
		}

		if ev.Eligible {
			ev.Score, ev.Components = score(a, p.Preference.NormalizedWeights(), ranges)
		}
		out = append(out, ev)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Eligible != out[j].Eligible {
			return out[i].Eligible // eligible first
		}
		if out[i].Eligible && !almostEqual(out[i].Score, out[j].Score) {
			return out[i].Score > out[j].Score
		}
		return out[i].Model < out[j].Model
	})
	return out
}

// objectiveRange records the min/max of each objective across candidates so
// scores can be normalized to [0,1] regardless of absolute units.
type objectiveRange struct {
	minCost, maxCost float64
	minLat, maxLat   float64
	minQual, maxQual float64
	minRel, maxRel   float64
}

func computeRanges(aggs []reliability.Aggregate) objectiveRange {
	r := objectiveRange{
		minCost: pos, maxCost: 0,
		minLat: pos, maxLat: 0,
		minQual: pos, maxQual: 0,
		minRel: pos, maxRel: 0,
	}
	found := false
	for _, a := range aggs {
		if a.Successes == 0 {
			continue
		}
		found = true
		r.minCost = min(r.minCost, a.MeanCostUSD)
		r.maxCost = max(r.maxCost, a.MeanCostUSD)
		r.minLat = min(r.minLat, a.P95LatencyMS)
		r.maxLat = max(r.maxLat, a.P95LatencyMS)
		r.minQual = min(r.minQual, a.MeanQuality)
		r.maxQual = max(r.maxQual, a.MeanQuality)
		r.minRel = min(r.minRel, a.WilsonLower)
		r.maxRel = max(r.maxRel, a.WilsonLower)
	}
	if !found {
		return objectiveRange{}
	}
	return r
}

const pos = 1e18

// score produces a normalized [0,1] weighted score. Cost and latency are
// inverted (lower is better) via normalization; quality and reliability are
// used directly. Components are exposed so explanations can show the breakdown.
func score(a reliability.Aggregate, w map[Objective]float64, r objectiveRange) (float64, map[string]float64) {
	comp := map[string]float64{}
	total := 0.0

	normLowerBetter := func(v, lo, hi float64) float64 {
		if hi <= lo {
			return 1.0
		}
		return 1.0 - (v-lo)/(hi-lo)
	}
	normHigherBetter := func(v, lo, hi float64) float64 {
		if hi <= lo {
			return 1.0
		}
		return (v - lo) / (hi - lo)
	}

	if wc, ok := w[ObjCost]; ok {
		s := normLowerBetter(a.MeanCostUSD, r.minCost, r.maxCost) * wc
		comp["cost"] = s
		total += s
	}
	if wc, ok := w[ObjLatency]; ok {
		s := normLowerBetter(a.P95LatencyMS, r.minLat, r.maxLat) * wc
		comp["latency"] = s
		total += s
	}
	if wc, ok := w[ObjQuality]; ok {
		s := normHigherBetter(a.MeanQuality, r.minQual, r.maxQual) * wc
		comp["quality"] = s
		total += s
	}
	if wc, ok := w[ObjReliability]; ok {
		s := normHigherBetter(a.WilsonLower, r.minRel, r.maxRel) * wc
		comp["reliability"] = s
		total += s
	}
	return total, comp
}

func toSet(xs []string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[strings.TrimSpace(x)] = true
	}
	return m
}

func almostEqual(a, b float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= 1e-9
}
