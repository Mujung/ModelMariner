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
