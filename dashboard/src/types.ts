/**
 * Type definitions for the modelmariner report document (schema
 * "modelmariner/v1"). These mirror the Go structs in internal/report exactly so
 * the dashboard consumes reports with full type safety and no `any` leakage.
 */

export type PrivacyTier =
  | "public"
  | "internal"
  | "confidential"
  | "restricted";

export interface InputSummary {
  total_lines: number;
  accepted: number;
  rejected: number;
  models: string[];
  tasks: string[];
  warnings?: string[];
}

export interface ReliabilityAggregate {
  model: string;
  task: string;
  samples: number;
  successes: number;
  failures: number;
  success_rate: number;
  reliability_lower: number;
  mean_cost_usd: number;
  mean_latency_ms: number;
  p50_latency_ms: number;
  p95_latency_ms: number;
  p99_latency_ms: number;
  mean_quality: number;
  mean_tokens: number;
  highest_privacy: PrivacyTier;
  error_kinds?: Record<string, number>;
}

export interface ParetoPoint {
  model: string;
  cost_usd: number;
  latency_ms: number;
  quality: number;
  reliability: number;
  dominated: boolean;
  dominated_by?: string[];
}

export interface ParetoFrontier {
  task: string;
  points: ParetoPoint[];
  frontier: ParetoPoint[];
}

export interface Violation {
  constraint: string;
  detail: string;
  limit?: number;
  observed?: number;
}

export interface Evaluation {
  model: string;
  eligible: boolean;
  score: number;
  violations?: Violation[];
  components?: Record<string, number>;
}

export interface RejectedCandidate {
