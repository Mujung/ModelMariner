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
