/**
 * Unit tests for the dashboard navigator and renderer using Node's built-in
 * test runner (node:test). No external test dependencies are required.
 */

import { test } from "node:test";
import assert from "node:assert/strict";
import { ReportNavigator, SchemaError } from "./navigator.js";
import { render } from "./dashboard.js";
import { Report } from "./types.js";

function sampleReport(): Report {
  return {
    schema: "modelmariner/v1",
    input: {
      total_lines: 4,
      accepted: 4,
      rejected: 0,
      models: ["cheap", "premium"],
      tasks: ["t"],
    },
    reliability: [
      {
        model: "cheap", task: "t", samples: 2, successes: 2, failures: 0,
        success_rate: 1, reliability_lower: 0.34, mean_cost_usd: 0.1,
        mean_latency_ms: 110, p50_latency_ms: 110, p95_latency_ms: 120,
        p99_latency_ms: 120, mean_quality: 0.81, mean_tokens: 20,
        highest_privacy: "public",
      },
      {
        model: "premium", task: "t", samples: 2, successes: 2, failures: 0,
        success_rate: 1, reliability_lower: 0.34, mean_cost_usd: 0.6,
        mean_latency_ms: 410, p50_latency_ms: 410, p95_latency_ms: 420,
        p99_latency_ms: 420, mean_quality: 0.975, mean_tokens: 20,
        highest_privacy: "public",
      },
    ],
    pareto: [
      {
        task: "t",
        points: [
          { model: "cheap", cost_usd: 0.1, latency_ms: 120, quality: 0.81, reliability: 0.34, dominated: false },
          { model: "premium", cost_usd: 0.6, latency_ms: 420, quality: 0.975, reliability: 0.34, dominated: false },
        ],
        frontier: [
          { model: "cheap", cost_usd: 0.1, latency_ms: 120, quality: 0.81, reliability: 0.34, dominated: false },
          { model: "premium", cost_usd: 0.6, latency_ms: 420, quality: 0.975, reliability: 0.34, dominated: false },
        ],
      },
    ],
    policies: [
      {
        name: "quality",
        description: "prefer quality",
        simulation: {
          policy: "quality",
          decisions: [
            {
              task: "t", policy: "quality", winner: "premium", runner_up: "cheap",
              score: 1, margin: 0.5, eligible: ["premium", "cheap"],
              evaluations: [
                { model: "premium", eligible: true, score: 1 },
                { model: "cheap", eligible: true, score: 0.5 },
              ],
              realized: { model: "premium", calls: 2, successes: 2, success_rate: 1, total_cost_usd: 1.2, mean_latency_ms: 415, mean_quality: 0.975 },
              baseline: { model: "cheap", calls: 2, successes: 2, success_rate: 1, total_cost_usd: 0.2, mean_latency_ms: 110, mean_quality: 0.81 },
              no_eligible: false,
            },
          ],
          totals: {
            tasks_decided: 1, tasks_unrouted: 0,
            realized_cost_usd: 1.2, baseline_cost_usd: 0.2, cost_delta_usd: 1.0,
            realized_mean_quality: 0.975, baseline_mean_quality: 0.81,
          },
        },
        explanations: [
          {
            task: "t", policy: "quality",
            headline: 'route "t" to premium (score 1.0000)',
            reasons: ["premium beat runner-up cheap"],
            evidence: ["replayed 2 recorded call(s)"],
          },
        ],
      },
    ],
  };
}

test("navigator validates schema", () => {
  const nav = new ReportNavigator(sampleReport());
  assert.deepEqual(nav.tasks(), ["t"]);
  assert.deepEqual(nav.models(), ["cheap", "premium"]);
  assert.deepEqual(nav.policies(), ["quality"]);
});

test("navigator rejects wrong schema", () => {
  const bad = { ...sampleReport(), schema: "bogus/v9" };
  assert.throws(() => new ReportNavigator(bad as Report), SchemaError);
});

test("fromJSON rejects invalid json", () => {
  assert.throws(() => ReportNavigator.fromJSON("{not json"), SchemaError);
});

test("routing table compiles winners", () => {
  const nav = new ReportNavigator(sampleReport());
  const table = nav.routingTable("quality");
  assert.equal(table.length, 1);
  assert.equal(table[0].model, "premium");
  assert.equal(table[0].task, "t");
});

test("cost delta reads totals", () => {
  const nav = new ReportNavigator(sampleReport());
  assert.equal(nav.policyCostDelta("quality"), 1.0);
});

test("quality ranking orders descending", () => {
  const nav = new ReportNavigator(sampleReport());
  const ranking = nav.qualityRanking("t");
  assert.equal(ranking[0].model, "premium");
  assert.ok(ranking[0].quality > ranking[1].quality);
});

test("task overview aggregates winners across policies", () => {
  const nav = new ReportNavigator(sampleReport());
  const ov = nav.taskOverview("t");
  assert.deepEqual(ov.frontierModels, ["cheap", "premium"]);
  assert.equal(ov.winnersByPolicy[0].winner, "premium");
});

test("render overview contains headline and winners", () => {
  const nav = new ReportNavigator(sampleReport());
  const out = render(nav, ["overview"]);
  assert.match(out, /fleet overview/);
  assert.match(out, /premium/);
});

test("render task shows frontier and selections", () => {
  const nav = new ReportNavigator(sampleReport());
  const out = render(nav, ["task", "t"]);
  assert.match(out, /Pareto frontier/);
  assert.match(out, /quality.*premium/s);
});

test("render routes shows compiled table", () => {
  const nav = new ReportNavigator(sampleReport());
  const out = render(nav, ["routes", "quality"]);
  assert.match(out, /COMPILED ROUTING TABLE/);
  assert.match(out, /premium/);
});

test("render model reports cross-task behavior", () => {
  const nav = new ReportNavigator(sampleReport());
  const out = render(nav, ["model", "premium"]);
  assert.match(out, /MODEL: premium/);
  assert.match(out, /Selected as winner in 1/);
});

test("render unknown command", () => {
  const nav = new ReportNavigator(sampleReport());
  assert.match(render(nav, ["bogus"]), /unknown command/);
});
