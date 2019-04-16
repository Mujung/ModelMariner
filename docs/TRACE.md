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
