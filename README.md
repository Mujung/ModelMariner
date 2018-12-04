<!-- ModelMariner — offline LLM trace evaluation & routing-policy compiler -->

<p align="center">
  <img src="docs/assets/model-fleet.svg" alt="ModelMariner fleet sailing an animated night sea" width="760">
</p>

<h1 align="center">⚓ ModelMariner</h1>

<p align="center">
  <em>Chart your model fleet. Compile the course. Let the recorded tides decide.</em>
</p>

<p align="center">
  <strong>An offline compiler that turns recorded LLM traces into auditable routing policies.</strong><br>
  No live API. No network. No guesswork — only the evidence already in your logs.
</p>

---

## Table of contents

- [Why a compiler, not a proxy](#why-a-compiler-not-a-proxy)
- [The voyage in one glance](#the-voyage-in-one-glance)
- [Chart room: core concepts](#chart-room-core-concepts)
- [Setting sail: quick start](#setting-sail-quick-start)
- [Reading the log: real output](#reading-the-log-real-output)
- [The Pareto compass](#the-pareto-compass)
- [Writing routing policies](#writing-routing-policies)
- [The navigation dashboard](#the-navigation-dashboard)
- [Deckhand's manual: CLI reference](#deckhands-manual-cli-reference)
- [How the hull is built](#how-the-hull-is-built)
- [Determinism: the ship's chronometer](#determinism-the-ships-chronometer)
- [Testing the rigging](#testing-the-rigging)
- [Project layout](#project-layout)
- [Design decisions worth defending](#design-decisions-worth-defending)
- [FAQ from the crow's nest](#faq-from-the-crows-nest)
- [License](#license)

---

## Why a compiler, not a proxy

Most "model routers" are **live** things: they sit in the request path, call
providers, and hope the decision they make right now is a good one. That is a
runtime gamble dressed up as infrastructure.

ModelMariner takes the opposite tack. It is an **offline compiler**. You feed it
the traces you already recorded — every model, every task, every token, dollar,
millisecond, quality score, error, and privacy tier — and it *compiles* a
routing policy the way a build tool compiles source: deterministically,
explainably, and with the whole picture in view.

The distinction matters for three reasons a captain cares about:

1. **Evidence over optimism.** Every decision is backed by observations you can
   point at. When ModelMariner routes `draft-legal-clause` to `lighthouse-local`,
   it can show you the 45 recorded calls that justify it.
2. **Hard constraints are actually hard.** A budget cap, a latency ceiling, a
   privacy tier — these *disqualify* candidates. They are never "soft penalties"
   that a high enough quality score can bribe past. Regulated workloads demand
   this, and so does anyone who has ever been paged at 3 a.m.
3. **Reproducibility.** The same traces and the same policy always compile to the
   same routing table, byte for byte. You can commit the output, diff it in code
   review, and gate CI on it.

ModelMariner never opens a socket to a provider. It cannot. That is the point.

---

## The voyage in one glance

```
   traces.jsonl ──▶ ingest & validate ──▶ reliability aggregation
                                                    │
                                                    ▼
                                          Pareto frontier per task
                                                    │
        policy.json ──▶ hard-constraint evaluation ─┤
                                                    ▼
                                    routing simulation (replay traces)
                                                    │
                                                    ▼
                              explanations  ◀──  deterministic report
                                                    │
                        report.json + report.txt + policies.json
                                                    │
                                                    ▼
                              TypeScript navigation dashboard
```

Every arrow is a package with tests. Every box is a decision you can inspect.

---

## Chart room: core concepts

| Term | What it means aboard ModelMariner |
|------|------------------------------------|
| **Trace** | One recorded observation: a model handling a task, with its cost/latency/quality/error/privacy. The atom of everything. |
| **Reliability** | Per model/task success rate plus a **Wilson score lower bound** so a lucky 9/10 never outranks a proven 90/100. |
| **Pareto frontier** | The set of *non-dominated* models for a task — the only candidates a rational router should even consider. |
| **Policy** | A bundle of **hard constraints** (budget, latency, quality, reliability, privacy) plus a weighted **preference** for ranking survivors. |
| **Compilation** | Applying a policy to the frontier to pick one winner per task, then replaying recorded traces to measure what that choice actually delivers. |
| **Explanation** | The human-readable "why": margins, score components, replayed evidence, and the exact constraint each rejected model tripped over. |

### The fleet

The bundled synthetic corpus sails five vessels, each a caricature of a real
trade-off you will recognize:

- **`harbor-nano`** — the dinghy. Cheapest and fastest, but its quality founders
  on hard generative work.
- **`harbor-mini`** — the sloop. A superb all-rounder that wins most
  cost-sensitive routes.
- **`clipper-pro`** — the clipper. Fast *and* accurate, at a real price.
- **`galleon-max`** — the galleon. Top quality, heaviest cost and latency.
- **`lighthouse-local`** — the on-prem lighthouse. Modest metrics, but the *only*
  vessel cleared to carry restricted cargo without leaving harbor.

No single vessel wins everywhere. That is by design — a router with an obvious
answer is not worth compiling.

---

## Setting sail: quick start

You need **Go 1.24+** and (for the dashboard) **Node 20+**. Nothing else.

```bash
# 1. Build the compiler
make build            # produces ./bin/modelmariner

# 2. Compile a report + routing tables from the sample fleet
make report           # writes testdata/output/{report.json,report.txt,policies.json}

# 3. Explore it in the dashboard
make demo             # builds the TS dashboard and prints the fleet overview
```

Or drive the binary directly:

```bash
./bin/modelmariner analyze \
  --traces testdata/fleet.jsonl \
  --policy testdata/policies.json \
  --out    testdata/output \
  --format both
```

---

## Reading the log: real output

This is **actual output** from `modelmariner analyze` over the 1,400-line sample
corpus — not a mock-up.

### Reliability, per model and task

```
========================================================================
 RELIABILITY (per model / task)
