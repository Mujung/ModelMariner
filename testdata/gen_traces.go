//go:build ignore

// gen_traces.go produces the realistic synthetic trace corpus used by the
// examples and tests. Run with: go run gen_traces.go > testdata/fleet.jsonl
// It is deterministic (seeded PRNG) so the committed corpus is reproducible.
package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"time"
)

type profile struct {
	model    string
	provider string
	region   string
	// per-task behavior: cost per 1k tokens, base latency, latency jitter,
	// base quality, error rate.
	costPer1k float64
	baseLatMS float64
	jitterMS  float64
	quality   float64
	errRate   float64
	privacy   string
}

func main() {
	rng := rand.New(rand.NewSource(20260831))

	tasks := []struct {
		name    string
		privacy string
		volume  int
	}{
		{"summarize-support-ticket", "internal", 60},
		{"classify-intent", "public", 70},
		{"extract-pii-redaction", "restricted", 50},
		{"draft-legal-clause", "confidential", 45},
		{"code-review-comment", "internal", 55},
	}

	// Each model behaves differently per task class. We describe base profiles
	// then perturb per task to create realistic, non-trivial trade-offs.
	models := []profile{
		{"harbor-nano", "coastal", "eu-west", 0.06, 180, 90, 0.71, 0.03, ""},
		{"harbor-mini", "coastal", "eu-west", 0.15, 240, 120, 0.82, 0.02, ""},
		{"clipper-pro", "openseas", "us-east", 0.55, 620, 260, 0.93, 0.015, ""},
		{"galleon-max", "openseas", "us-east", 1.20, 1150, 400, 0.965, 0.01, ""},
		{"lighthouse-local", "onprem", "on-prem", 0.09, 320, 140, 0.78, 0.04, ""},
	}

	baseTime := time.Date(2026, 8, 25, 8, 0, 0, 0, time.UTC)
	var seq int

	emit := func(rec string) { fmt.Fprintln(os.Stdout, rec) }
	emit("# modelmariner synthetic trace corpus — provider-neutral JSONL")
	emit("# deterministic seed 20260831; each line is one recorded observation")

	for _, task := range tasks {
		for _, m := range models {
			// Task-specific perturbations so no single model wins everywhere.
			p := m
			switch task.name {
			case "draft-legal-clause":
				// Bigger models pull ahead on hard generative tasks.
				if m.model == "harbor-nano" {
					p.quality -= 0.12
					p.errRate += 0.05
				}
				if m.model == "galleon-max" {
					p.quality += 0.02
				}
			case "classify-intent":
				// Small models are plenty good and dominate on cost/latency.
				if m.model == "harbor-nano" || m.model == "harbor-mini" {
					p.quality += 0.08
				}
			case "extract-pii-redaction":
				// Only the on-prem model is privacy-safe; cloud models still
				// appear in traces (they were tried) but score lower on quality.
				if m.model == "lighthouse-local" {
					p.quality += 0.06
				}
			case "code-review-comment":
