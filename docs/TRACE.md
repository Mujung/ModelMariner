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
