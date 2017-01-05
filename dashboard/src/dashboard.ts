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
