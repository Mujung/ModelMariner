// Package report assembles the outputs of every engine stage into a single
// deterministic document that both the text renderer and the TypeScript
// dashboard consume. Determinism is a first-class requirement: given identical
// inputs the JSON bytes are byte-for-byte identical, which makes reports safe to
// commit, diff, and gate CI on.
package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Mujung/modelmariner/internal/explain"
	"github.com/Mujung/modelmariner/internal/pareto"
	"github.com/Mujung/modelmariner/internal/policy"
	"github.com/Mujung/modelmariner/internal/reliability"
	"github.com/Mujung/modelmariner/internal/routing"
	"github.com/Mujung/modelmariner/internal/trace"
)

// SchemaVersion identifies the report format so the dashboard can guard against
// consuming an incompatible document.
const SchemaVersion = "modelmariner/v1"

// Report is the complete, self-describing analysis document.
type Report struct {
	Schema      string                  `json:"schema"`
	Generated   string                  `json:"generated,omitempty"`
	Input       InputSummary            `json:"input"`
	Reliability []reliability.Aggregate `json:"reliability"`
	Pareto      []pareto.Frontier       `json:"pareto"`
	Policies    []PolicyReport          `json:"policies"`
}

// InputSummary describes what was ingested.
type InputSummary struct {
	TotalLines int      `json:"total_lines"`
	Accepted   int      `json:"accepted"`
	Rejected   int      `json:"rejected"`
	Models     []string `json:"models"`
	Tasks      []string `json:"tasks"`
	Warnings   []string `json:"warnings,omitempty"`
}

// PolicyReport bundles a simulation with its explanations.
type PolicyReport struct {
