# TRACE.md — the modelmariner trace format

modelmariner reasons over **recorded observations**, never live calls. This
document is the authoritative reference for the input it consumes (JSONL traces)
and the output it produces (report and policy artifacts).

---

## 1. Trace input: provider-neutral JSONL

The compiler ingests newline-delimited JSON. Each non-blank, non-comment line is
exactly one **trace**: a single recorded observation of one model handling one
task. Lines beginning with `#` are treated as comments and skipped, as are blank
lines. Field order does not matter and unknown fields are rejected, which keeps
the format honest as it evolves.

### 1.1 Record schema

| Field         | JSON type        | Required | Meaning                                                        |
|---------------|------------------|----------|----------------------------------------------------------------|
| `model`       | string           | yes      | Model identifier (e.g. `clipper-pro`). Non-empty.              |
| `task`        | string           | yes      | Logical task the call served (e.g. `classify-intent`).        |
| `tokens`      | object           | yes      | `{ "prompt": int>=0, "completion": int>=0 }`.                 |
| `cost_usd`    | number           | yes      | Cost of the call in USD. Finite, `>= 0`.                      |
| `latency_ms`  | number           | yes      | Wall-clock latency in milliseconds. Finite, `>= 0`.          |
| `quality`     | number           | yes      | Recorded quality score. Clamped into `[0, 1]` on ingest.      |
| `privacy`     | string or int    | yes      | Privacy tier of the data (see §1.2).                          |
| `error`       | bool             | no       | `true` if the call failed. Defaults to `false`.              |
| `error_kind`  | string           | cond.    | Failure category. **Required when `error` is `true`.**        |
| `provider`    | string           | no       | Optional provider/label (e.g. `openseas`, `onprem`).         |
| `region`      | string           | no       | Optional region (e.g. `eu-west`).                            |
| `timestamp`   | string           | no       | Optional RFC 3339 timestamp; used only for deterministic sort.|

### 1.2 Privacy tiers

Privacy is an **ordered** enumeration. Higher tiers demand stricter routing.

| Name           | Rank | Use                                                     |
|----------------|------|---------------------------------------------------------|
| `public`       | 0    | Safe for any destination.                               |
| `internal`     | 1    | Must stay within the organization.                      |
| `confidential` | 2    | Sensitive business data.                                |
| `restricted`   | 3    | PII, secrets, regulated data.                           |

A trace may carry the tier either as its name (`"restricted"`) or its integer
rank (`3`). Reports always emit the name.

### 1.3 Example lines

```jsonl
# a successful call
{"model":"clipper-pro","task":"classify-intent","provider":"openseas","region":"us-east","tokens":{"prompt":420,"completion":88},"cost_usd":0.2794,"latency_ms":612.4,"quality":0.93,"error":false,"privacy":"public","timestamp":"2026-08-25T08:00:00Z"}
# a failed call — error_kind is mandatory here
{"model":"harbor-nano","task":"draft-legal-clause","tokens":{"prompt":900,"completion":0},"cost_usd":0.054,"latency_ms":301.2,"quality":0,"error":true,"error_kind":"content_filter","privacy":"confidential"}
```

### 1.4 Validation & normalization

Ingestion is **strict per record but lenient per file** by default: an invalid
line is collected as a `ValidationError` (with line number, field, and reason)
and skipped, while valid lines still load. Pass `--strict` to make any invalid
line abort the run. Normalization performs exactly one transform beyond
validation: `quality` is clamped into `[0, 1]`. Traces are then sorted
deterministically by `(model, task, timestamp, line)` so output never depends on
input order.

Only **successful** calls contribute to cost, latency, and quality
distributions; a failed call's metrics are not representative of the service a
model delivers. Failures still count toward the success rate and reliability
bound.

---

## 2. Report output: `report.json`

`analyze --out DIR` writes three files. `report.json` is the complete document
(schema `modelmariner/v1`) with these top-level sections:

- `input` — line counts, discovered models/tasks, and any rejection warnings.
- `reliability` — one entry per model/task with success rate, `reliability_lower`
  (Wilson 95% lower bound), latency `p50/p95/p99`, mean cost/quality/tokens,
  the `highest_privacy` tier observed, and an `error_kinds` histogram.
- `pareto` — per task, every point in objective space plus the non-dominated
  `frontier`; each dominated point lists the models that dominate it.
- `policies` — per policy, the full `simulation` (decisions + realized/baseline
