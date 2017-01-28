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
