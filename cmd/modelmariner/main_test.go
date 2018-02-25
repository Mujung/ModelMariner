package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

const traceLines = `{"model":"cheap","task":"t","tokens":{"prompt":10,"completion":10},"cost_usd":0.10,"latency_ms":100,"quality":0.80,"error":false,"privacy":"public"}
{"model":"premium","task":"t","tokens":{"prompt":10,"completion":10},"cost_usd":0.60,"latency_ms":400,"quality":0.98,"error":false,"privacy":"public"}`

const policyDoc = `{"policies":[{"name":"q","preference":{"weights":[{"objective":"quality","weight":1}]}}]}`

func TestRunAnalyzeWritesArtifacts(t *testing.T) {
	tp := writeTemp(t, "traces.jsonl", traceLines)
	pp := writeTemp(t, "policy.json", policyDoc)
	out := t.TempDir()
	err := run([]string{"analyze", "--traces", tp, "--policy", pp, "--out", out, "--format", "json"})
	if err != nil {
		t.Fatalf("analyze failed: %v", err)
	}
	for _, name := range []string{"report.json", "report.txt", "policies.json"} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
