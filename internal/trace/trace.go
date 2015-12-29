// Package trace defines the provider-neutral trace record model and the
// ingestion pipeline that reads JSONL logs, validates every field, and
// normalizes values into a canonical form suitable for evaluation.
//
// A "trace" is a single recorded observation of one model handling one task:
// how many tokens it consumed, what it cost, how long it took, how good the
// answer was, whether it errored, and what privacy tier the data belonged to.
// modelmariner never contacts a provider; it reasons entirely over these
// recorded observations, which makes every run deterministic and replayable.
package trace

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
)

// PrivacyTier classifies the sensitivity of the data a task carried. Higher
// tiers demand stricter routing (for example, forbidding models that ship data
// off-premises). The ordering is meaningful: Public < Internal < Confidential < Restricted.
type PrivacyTier int

const (
	// PrivacyPublic marks data safe for any destination.
	PrivacyPublic PrivacyTier = iota
	// PrivacyInternal marks data that must stay within the organization.
	PrivacyInternal
	// PrivacyConfidential marks sensitive business data.
	PrivacyConfidential
	// PrivacyRestricted marks the most sensitive data (PII, secrets, regulated).
	PrivacyRestricted
)

var privacyNames = [...]string{"public", "internal", "confidential", "restricted"}

// String renders the tier as its canonical lowercase name.
func (p PrivacyTier) String() string {
	if p < 0 || int(p) >= len(privacyNames) {
		return "unknown"
	}
	return privacyNames[p]
}

// ParsePrivacyTier converts a case-insensitive string into a PrivacyTier.
func ParsePrivacyTier(s string) (PrivacyTier, error) {
	norm := strings.ToLower(strings.TrimSpace(s))
	for i, name := range privacyNames {
		if name == norm {
			return PrivacyTier(i), nil
		}
	}
	return 0, fmt.Errorf("unknown privacy tier %q (want one of public, internal, confidential, restricted)", s)
}

// MarshalJSON encodes the tier as its name so reports read cleanly.
func (p PrivacyTier) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

// UnmarshalJSON accepts either the string name or the integer rank.
func (p *PrivacyTier) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		tier, err := ParsePrivacyTier(s)
		if err != nil {
			return err
		}
		*p = tier
		return nil
	}
	var n int
	if err := json.Unmarshal(data, &n); err != nil {
		return fmt.Errorf("privacy tier must be a string or integer: %w", err)
	}
	if n < 0 || n >= len(privacyNames) {
		return fmt.Errorf("privacy tier rank %d out of range", n)
	}
	*p = PrivacyTier(n)
	return nil
}

// Record is the raw shape read from a JSONL line. Fields use pointers where a
// missing value is semantically different from a zero value so validation can
// distinguish "absent" from "explicitly zero".
type Record struct {
	Model     string      `json:"model"`
	Task      string      `json:"task"`
	Provider  string      `json:"provider,omitempty"`
	Region    string      `json:"region,omitempty"`
	Tokens    *TokenUsage `json:"tokens"`
	CostUSD   *float64    `json:"cost_usd"`
	LatencyMS *float64    `json:"latency_ms"`
	Quality   *float64    `json:"quality"`
	Error     bool        `json:"error"`
	ErrorKind string      `json:"error_kind,omitempty"`
	Privacy   PrivacyTier `json:"privacy"`
	Timestamp string      `json:"timestamp,omitempty"`
}

// TokenUsage captures prompt and completion token counts.
type TokenUsage struct {
	Prompt     int `json:"prompt"`
	Completion int `json:"completion"`
}

// Total returns the sum of prompt and completion tokens.
func (t TokenUsage) Total() int { return t.Prompt + t.Completion }

// Trace is the validated, normalized observation used throughout the engine.
// Unlike Record, every field here is guaranteed valid and in canonical units.
type Trace struct {
	Model      string
	Task       string
	Provider   string
	Region     string
	Prompt     int
	Completion int
	CostUSD    float64
	LatencyMS  float64
	Quality    float64 // clamped to [0,1]
	Error      bool
	ErrorKind  string
	Privacy    PrivacyTier
	Timestamp  string
	// Line records the 1-based source line for diagnostics.
	Line int
}

// Tokens returns total token usage for the trace.
func (t Trace) Tokens() int { return t.Prompt + t.Completion }

// ValidationError describes exactly why a single source line was rejected.
