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
