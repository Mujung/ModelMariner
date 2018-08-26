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
