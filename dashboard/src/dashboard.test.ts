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
