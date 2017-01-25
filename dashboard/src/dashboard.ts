/**
 * A terminal navigation dashboard for modelmariner reports. It renders a
 * report into browsable views without any remote assets or network access.
 *
 * Usage:
 *   node dist/dashboard.js <report.json> [command]
 *
 * Commands:
 *   overview            Fleet summary and per-task routing winners (default).
 *   task <name>         Deep-dive one task: reliability, frontier, policy picks.
 *   model <name>        One model's behavior across every task.
 *   policy <name>       A policy's compiled routing table and economics.
 *   routes <name>       Just the compiled winner-per-task routing table.
 */

import { readFileSync } from "node:fs";
import { ReportNavigator, SchemaError } from "./navigator.js";

const BAR = "─".repeat(70);

function pad(s: string, n: number): string {
  return s.length >= n ? s.slice(0, n) : s + " ".repeat(n - s.length);
}

function money(n: number): string {
  return `$${n.toFixed(4)}`;
}

function pct(n: number): string {
  return `${(n * 100).toFixed(1)}%`;
}

function renderOverview(nav: ReportNavigator): string {
  const out: string[] = [];
  const r = nav.report;
  out.push(BAR);
  out.push("  MODELMARINER DASHBOARD — fleet overview");
  out.push(BAR);
  out.push(
    `  Traces: ${r.input.accepted} accepted / ${r.input.total_lines} lines  |  ` +
      `${r.input.models.length} models  |  ${r.input.tasks.length} tasks  |  ` +
      `${r.policies.length} policies`,
  );
  out.push(`  Models: ${r.input.models.join(", ")}`);
  out.push("");
  out.push("  ROUTING WINNERS BY TASK");
  out.push(`  ${pad("task", 26)}${r.policies.map((p) => pad(p.name, 18)).join("")}`);
  for (const task of nav.tasks()) {
    const ov = nav.taskOverview(task);
    const cells = r.policies.map((p) => {
      const w = ov.winnersByPolicy.find((x) => x.policy === p.name);
      if (!w) return pad("—", 18);
      return pad(w.winner ?? "UNROUTED", 18);
    });
    out.push(`  ${pad(task, 26)}${cells.join("")}`);
  }
  out.push("");
  out.push("  POLICY ECONOMICS (realized vs cheapest-model baseline)");
  for (const p of r.policies) {
    const t = p.simulation.totals;
    const verb = t.cost_delta_usd >= 0 ? "spent" : "saved";
    out.push(
      `  ${pad(p.name, 20)} ${verb} ${money(Math.abs(t.cost_delta_usd))} ` +
        `for +${(t.realized_mean_quality - t.baseline_mean_quality).toFixed(3)} quality ` +
        `(${t.tasks_decided} routed, ${t.tasks_unrouted} unrouted)`,
    );
  }
  return out.join("\n");
}

function renderTask(nav: ReportNavigator, task: string): string {
  const out: string[] = [];
  out.push(BAR);
  out.push(`  TASK: ${task}`);
  out.push(BAR);
  const rel = nav.reliabilityForTask(task);
  if (rel.length === 0) {
    return `no such task: ${task}`;
  }
  out.push(`  ${pad("model", 18)}${pad("success", 10)}${pad("reliab.LB", 12)}${pad("p95 ms", 10)}${pad("quality", 10)}${pad("cost", 10)}`);
  for (const a of rel) {
    out.push(
      `  ${pad(a.model, 18)}${pad(pct(a.success_rate), 10)}` +
        `${pad(a.reliability_lower.toFixed(3), 12)}${pad(a.p95_latency_ms.toFixed(0), 10)}` +
        `${pad(a.mean_quality.toFixed(3), 10)}${pad(money(a.mean_cost_usd), 10)}`,
    );
  }
  const f = nav.frontier(task);
  if (f) {
    out.push("");
    out.push(`  Pareto frontier: ${f.frontier.map((p) => p.model).join(", ")}`);
    const dominated = f.points.filter((p) => p.dominated);
    for (const p of dominated) {
      out.push(`    ${p.model} dominated by ${(p.dominated_by ?? []).join("/")}`);
    }
  }
  out.push("");
  out.push("  POLICY SELECTIONS");
  for (const policy of nav.policies()) {
    const d = nav.decision(policy, task);
    if (!d) continue;
    if (d.no_eligible) {
      out.push(`    ${pad(policy, 20)} UNROUTED — no eligible model`);
    } else {
      out.push(`    ${pad(policy, 20)} → ${d.winner} (score ${d.score.toFixed(4)}, margin ${d.margin.toFixed(4)})`);
    }
  }
  return out.join("\n");
}

function renderModel(nav: ReportNavigator, model: string): string {
  const out: string[] = [];
  out.push(BAR);
  out.push(`  MODEL: ${model}`);
  out.push(BAR);
  const rows = nav.reliabilityForModel(model);
  if (rows.length === 0) {
    return `no such model: ${model}`;
  }
  out.push(`  ${pad("task", 26)}${pad("success", 10)}${pad("p95 ms", 10)}${pad("quality", 10)}${pad("cost", 10)}`);
  for (const a of rows) {
    out.push(
      `  ${pad(a.task, 26)}${pad(pct(a.success_rate), 10)}` +
        `${pad(a.p95_latency_ms.toFixed(0), 10)}${pad(a.mean_quality.toFixed(3), 10)}${pad(money(a.mean_cost_usd), 10)}`,
    );
  }
  // Count how many task/policy combinations pick this model.
  let wins = 0;
