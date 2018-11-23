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
