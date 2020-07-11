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
