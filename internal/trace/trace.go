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
type ValidationError struct {
	Line   int
	Field  string
	Reason string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("line %d: field %q: %s", e.Line, e.Field, e.Reason)
}

// IngestResult bundles the successfully parsed traces with any per-line errors.
// Ingestion is intentionally lenient at the collection level (it keeps going
// past bad lines) but strict per record, so callers can decide whether a run
// with rejected lines should fail.
type IngestResult struct {
	Traces   []Trace
	Errors   []ValidationError
	Rejected int
	Total    int
}

// Options tunes the ingestion pipeline.
type Options struct {
	// SkipInvalid keeps ingestion going past invalid lines instead of stopping.
	SkipInvalid bool
	// MaxQuality clamps quality to this ceiling (defaults to 1.0).
	MaxQuality float64
}

// DefaultOptions returns the standard ingestion configuration.
func DefaultOptions() Options {
	return Options{SkipInvalid: true, MaxQuality: 1.0}
}

// Ingest reads JSONL from r, validating and normalizing each line. Blank lines
// and lines beginning with '#' (comments) are ignored. The returned traces are
// sorted deterministically by (model, task, timestamp, line) so downstream
// output never depends on input ordering.
func Ingest(r io.Reader, opts Options) (IngestResult, error) {
	if opts.MaxQuality <= 0 {
		opts.MaxQuality = 1.0
	}
	var res IngestResult
	scanner := bufio.NewScanner(r)
	// Allow long lines; traces can carry large token counts and metadata.
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		res.Total++
		var rec Record
		dec := json.NewDecoder(strings.NewReader(raw))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&rec); err != nil {
			ve := ValidationError{Line: lineNo, Field: "<json>", Reason: err.Error()}
			res.Errors = append(res.Errors, ve)
			res.Rejected++
			if !opts.SkipInvalid {
				return res, ve
			}
			continue
		}
		tr, verrs := normalize(rec, lineNo, opts)
		if len(verrs) > 0 {
			res.Errors = append(res.Errors, verrs...)
			res.Rejected++
			if !opts.SkipInvalid {
				return res, verrs[0]
			}
			continue
		}
		res.Traces = append(res.Traces, tr)
	}
	if err := scanner.Err(); err != nil {
		return res, fmt.Errorf("reading traces: %w", err)
	}
	sort.SliceStable(res.Traces, func(i, j int) bool {
		a, b := res.Traces[i], res.Traces[j]
		if a.Model != b.Model {
			return a.Model < b.Model
		}
		if a.Task != b.Task {
			return a.Task < b.Task
		}
		if a.Timestamp != b.Timestamp {
			return a.Timestamp < b.Timestamp
		}
		return a.Line < b.Line
	})
	return res, nil
}

// normalize validates a raw record and converts it into a canonical Trace.
func normalize(rec Record, line int, opts Options) (Trace, []ValidationError) {
	var errs []ValidationError
	add := func(field, reason string) {
		errs = append(errs, ValidationError{Line: line, Field: field, Reason: reason})
	}

	model := strings.TrimSpace(rec.Model)
	if model == "" {
		add("model", "must be a non-empty identifier")
	}
	task := strings.TrimSpace(rec.Task)
	if task == "" {
		add("task", "must be a non-empty identifier")
	}
	if rec.Tokens == nil {
		add("tokens", "required object with prompt and completion counts")
	} else {
		if rec.Tokens.Prompt < 0 {
			add("tokens.prompt", "must be >= 0")
		}
		if rec.Tokens.Completion < 0 {
			add("tokens.completion", "must be >= 0")
		}
	}
	if rec.CostUSD == nil {
		add("cost_usd", "required numeric field")
	} else if *rec.CostUSD < 0 {
		add("cost_usd", "must be >= 0")
	} else if math.IsNaN(*rec.CostUSD) || math.IsInf(*rec.CostUSD, 0) {
		add("cost_usd", "must be a finite number")
	}
	if rec.LatencyMS == nil {
		add("latency_ms", "required numeric field")
	} else if *rec.LatencyMS < 0 {
		add("latency_ms", "must be >= 0")
	} else if math.IsNaN(*rec.LatencyMS) || math.IsInf(*rec.LatencyMS, 0) {
		add("latency_ms", "must be a finite number")
	}
	if rec.Quality == nil {
		add("quality", "required numeric field in [0,1]")
	} else if math.IsNaN(*rec.Quality) || math.IsInf(*rec.Quality, 0) {
		add("quality", "must be a finite number")
	}
	if rec.Error && strings.TrimSpace(rec.ErrorKind) == "" {
		add("error_kind", "must be set when error is true")
	}

	if len(errs) > 0 {
		return Trace{}, errs
	}

	quality := *rec.Quality
	// Normalize quality into [0, MaxQuality] by clamping. Recorded scorers can
	// occasionally overshoot; clamping keeps Pareto math well-behaved.
	if quality < 0 {
		quality = 0
	}
	if quality > opts.MaxQuality {
		quality = opts.MaxQuality
	}

	return Trace{
		Model:      model,
		Task:       task,
		Provider:   strings.TrimSpace(rec.Provider),
		Region:     strings.TrimSpace(rec.Region),
		Prompt:     rec.Tokens.Prompt,
		Completion: rec.Tokens.Completion,
		CostUSD:    *rec.CostUSD,
		LatencyMS:  *rec.LatencyMS,
		Quality:    quality,
		Error:      rec.Error,
		ErrorKind:  strings.TrimSpace(rec.ErrorKind),
		Privacy:    rec.Privacy,
		Timestamp:  strings.TrimSpace(rec.Timestamp),
		Line:       line,
	}, nil
// review note
