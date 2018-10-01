# Changelog

All notable changes to modelmariner are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] — 2026-08-31

The first stable release. modelmariner compiles routing policies against
recorded LLM traces, entirely offline.

### Added

- **Trace ingestion** (`internal/trace`): provider-neutral JSONL reader with
  strict per-record validation, quality clamping, comment/blank-line skipping,
  deterministic sorting, and an ordered `PrivacyTier` enum
  (public < internal < confidential < restricted) that accepts either the name
  or the integer rank in JSON.
- **Reliability aggregation** (`internal/reliability`): per model/task success
  rate, Wilson score lower bound (95%) so thinly-sampled models cannot dominate,
  latency p50/p95/p99 via interpolation, mean cost/quality/tokens, and observed
  error-kind histograms.
- **Pareto frontiers** (`internal/pareto`): multi-objective dominance analysis
  over cost, p95 latency, quality, and reliability, reporting both the frontier
  and, for each dominated model, exactly which models dominate it.
- **Policy language & evaluator** (`internal/policy`): hard budget, latency,
  quality, reliability, and privacy constraints plus allow/deny and
  privacy-safe model lists; normalized weighted preferences for ranking
  survivors, with per-objective score components exposed for explainability.
- **Routing simulation** (`internal/routing`): compiles a winner per task and
  replays every recorded trace through the choice to measure realized cost,
  latency, quality, and success rate, compared against a cheapest-model
  baseline.
- **Explanations** (`internal/explain`): human-readable rationale for every
  selection, including margins over runners-up, score breakdowns, replayed
  evidence, and the specific constraint each rejected candidate violated.
