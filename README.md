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
========================================================================
task               model               n  success  reliab.LB    p95 ms  quality
classify-intent    clipper-pro        70    97.1%      0.902      1040    0.926
classify-intent    galleon-max        70    98.6%      0.923      1696    0.953
classify-intent    harbor-mini        70    97.1%      0.902       512    0.908
classify-intent    harbor-nano        70    95.7%      0.881       309    0.778
classify-intent    lighthouse-lo.     70    98.6%      0.923       584    0.784
draft-legal-clause clipper-pro        45   100.0%      0.921       925    0.939
draft-legal-clause galleon-max        45   100.0%      0.921      1634    0.964
draft-legal-clause harbor-mini        45    97.8%      0.884       417    0.819
draft-legal-clause harbor-nano        45    84.4%      0.712       302    0.589
draft-legal-clause lighthouse-lo.     45    97.8%      0.884       550    0.779
```

Notice `harbor-nano` on `draft-legal-clause`: only **84.4%** success and a
reliability lower bound of **0.712**. The dinghy is out of its depth on legal
drafting, and the numbers say so before any policy is applied.

### A compiled routing decision, fully explained

```
• route "classify-intent" to harbor-mini (score 0.8157)
    - harbor-mini beat runner-up lighthouse-local by a margin of 0.0698 in weighted score
    - score components: cost=0.5573, quality=0.1860, reliability=0.0724
    evidence: replayed 70 recorded call(s): 97.1% success, mean quality 0.908, mean latency 259 ms, total cost $12.9234
    evidence: versus cheapest-model baseline harbor-nano: $7.5293 more cost, 0.908 vs 0.778 mean quality
    rejected: clipper-pro — max_cost_usd (limit 0.35, observed 0.6692)
    rejected: galleon-max — max_cost_usd (limit 0.35, observed 1.612)
```

Every line is defensible. The winner, the runner-up and the margin between them,
the weighted breakdown, the replayed evidence, the baseline comparison, and the
precise reason each disqualified vessel was left at the dock.

### Privacy that actually holds the line

```
• route "draft-legal-clause" to lighthouse-local (score 0.6630)
    - lighthouse-local was the only model to satisfy every hard constraint
    rejected: clipper-pro — max_privacy (task data reaches "confidential"
              but policy caps at "internal" and model is not privacy-safe)
    rejected: galleon-max — max_privacy (task data reaches "confidential"
              but policy caps at "internal" and model is not privacy-safe)
```

Under the `privacy-lockdown` policy, confidential data cannot leave on-prem
infrastructure. The cloud vessels are disqualified outright — not down-weighted,
**disqualified** — and the only privacy-safe model wins by default.

---

## The Pareto compass

<p align="center">
  <img src="docs/assets/pareto-compass.svg" alt="Animated compass needle sweeping a cost-versus-quality Pareto frontier" width="520">
</p>

Before any policy is applied, ModelMariner draws the **Pareto frontier** for each
task. A model is *dominated* when some other model is at least as good on every
objective — cheaper-or-equal, faster-or-equal, higher-or-equal quality,
more-reliable-or-equal — and strictly better on at least one. Dominated models
are vessels no rational captain would ever choose, so they are flagged and
explained:

```
 PARETO FRONTIERS (non-dominated models per task)
classify-intent    frontier: harbor-nano, lighthouse-local, harbor-mini, clipper-pro, galleon-max
    harbor-nano    cost $0.0774  p95 309ms  q 0.778  rel 0.881  on-frontier
    harbor-mini    cost $0.1867  p95 512ms  q 0.908  rel 0.902  on-frontier
    galleon-max    cost $1.6117  p95 1696ms q 0.953  rel 0.923  on-frontier
```

The compass needle in the image above sweeps exactly this space: cyan vessels
ride the frontier; a dominated vessel drifts grey and is never chosen. Objectives
minimized (cost, latency) point one way; objectives maximized (quality,
reliability) point the other. The frontier is the coastline between them.

---

## Writing routing policies

A policy set is JSON. Constraints are hard; the preference ranks whoever
survives. Here is the bundled `budget-guard`:

```json
{
  "name": "budget-guard",
  "description": "Minimize spend while keeping quality acceptable across all tasks.",
  "constraints": {
    "max_cost_usd": 0.35,
    "min_quality": 0.7,
    "min_reliability": 0.8
  },
  "preference": {
    "weights": [
      { "objective": "cost", "weight": 0.6 },
      { "objective": "quality", "weight": 0.25 },
      { "objective": "reliability", "weight": 0.15 }
    ]
  }
}
```
