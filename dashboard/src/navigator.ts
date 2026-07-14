/**
 * The navigator is the read model the dashboard renders. It ingests a raw
 * report document, validates its schema, and exposes ergonomic navigation
 * queries — walk tasks, inspect a model across tasks, drill into a policy's
 * decisions, and surface the compiled routing table. It performs no I/O so it
 * is trivially unit-testable and reusable in any front-end shell.
 */

import {
  Decision,
  ParetoFrontier,
  PolicyReport,
  ReliabilityAggregate,
  Report,
  SCHEMA_VERSION,
} from "./types.js";

/** Raised when a document does not conform to the expected schema. */
export class SchemaError extends Error {}

/** A concise per-task overview combining routing winners across policies. */
export interface TaskOverview {
  task: string;
  frontierModels: string[];
  winnersByPolicy: { policy: string; winner: string | null; score: number }[];
}

/** A compiled route entry for the routing-table view. */
export interface RouteEntry {
  task: string;
  model: string;
  score: number;
  margin: number;
}

export class ReportNavigator {
  readonly report: Report;

  constructor(report: Report) {
    if (!report || typeof report !== "object") {
      throw new SchemaError("report is not an object");
    }
    if (report.schema !== SCHEMA_VERSION) {
      throw new SchemaError(
        `unsupported schema ${JSON.stringify(report.schema)} (expected ${SCHEMA_VERSION})`,
      );
    }
    if (!Array.isArray(report.reliability) || !Array.isArray(report.pareto)) {
      throw new SchemaError("report is missing reliability or pareto sections");
    }
    this.report = report;
  }

  /** Parse a JSON string into a validated navigator. */
  static fromJSON(text: string): ReportNavigator {
    let parsed: unknown;
    try {
      parsed = JSON.parse(text);
    } catch (e) {
      throw new SchemaError(`report is not valid JSON: ${(e as Error).message}`);
    }
    return new ReportNavigator(parsed as Report);
  }

  /** All task names, sorted. */
  tasks(): string[] {
    return [...this.report.input.tasks];
  }

  /** All model names, sorted. */
  models(): string[] {
    return [...this.report.input.models];
  }

  /** All policy names in the report. */
  policies(): string[] {
    return this.report.policies.map((p) => p.name);
  }

  /** Reliability aggregates for a single task. */
  reliabilityForTask(task: string): ReliabilityAggregate[] {
    return this.report.reliability.filter((a) => a.task === task);
  }

  /** Reliability aggregates for a single model across every task. */
  reliabilityForModel(model: string): ReliabilityAggregate[] {
    return this.report.reliability.filter((a) => a.model === model);
  }

  /** The Pareto frontier for a task, if present. */
  frontier(task: string): ParetoFrontier | undefined {
    return this.report.pareto.find((f) => f.task === task);
  }

  /** A named policy report, if present. */
  policy(name: string): PolicyReport | undefined {
    return this.report.policies.find((p) => p.name === name);
  }

  /** The routing decision for a task under a policy, if present. */
  decision(policy: string, task: string): Decision | undefined {
    return this.policy(policy)?.simulation.decisions.find(
      (d) => d.task === task,
    );
  }

  /** Compile the winner-per-task routing table for a policy. */
  routingTable(policy: string): RouteEntry[] {
    const pr = this.policy(policy);
    if (!pr) return [];
    return pr.simulation.decisions
      .filter((d) => !d.no_eligible)
      .map((d) => ({
        task: d.task,
        model: d.winner,
        score: d.score,
        margin: d.margin,
      }))
      .sort((a, b) => a.task.localeCompare(b.task));
  }

  /** Tasks a policy could not route (no eligible model). */
  unroutedTasks(policy: string): string[] {
    const pr = this.policy(policy);
    if (!pr) return [];
    return pr.simulation.decisions
      .filter((d) => d.no_eligible)
      .map((d) => d.task)
      .sort((a, b) => a.localeCompare(b));
  }

  /** A cross-policy overview for a single task. */
  taskOverview(task: string): TaskOverview {
    const frontier = this.frontier(task);
    const winnersByPolicy = this.report.policies
      .map((p) => {
        const d = p.simulation.decisions.find((x) => x.task === task);
        if (!d) return null;
        return {
          policy: p.name,
          winner: d.no_eligible ? null : d.winner,
          score: d.score,
        };
      })
      .filter((x): x is NonNullable<typeof x> => x !== null);
    return {
      task,
      frontierModels: frontier ? frontier.frontier.map((p) => p.model) : [],
      winnersByPolicy,
    };
  }

  /**
   * Total net savings (or cost) a policy realized versus the naive baseline,
   * summed across all decided tasks. Negative means the policy saved money.
   */
  policyCostDelta(policy: string): number {
    const pr = this.policy(policy);
    return pr ? pr.simulation.totals.cost_delta_usd : 0;
  }

  /** Rank models for a task by their mean quality (descending). */
  qualityRanking(task: string): { model: string; quality: number }[] {
    return this.reliabilityForTask(task)
      .filter((a) => a.successes > 0)
      .map((a) => ({ model: a.model, quality: a.mean_quality }))
      .sort((a, b) => b.quality - a.quality);
  }
}
