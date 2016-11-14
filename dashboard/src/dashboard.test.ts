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
