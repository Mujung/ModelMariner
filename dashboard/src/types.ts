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
