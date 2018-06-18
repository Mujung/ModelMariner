package trace

import (
	"strings"
	"testing"
)

func TestIngestValidLine(t *testing.T) {
	in := `{"model":"clipper-pro","task":"summarize","tokens":{"prompt":100,"completion":50},"cost_usd":0.25,"latency_ms":420,"quality":0.9,"error":false,"privacy":"internal"}`
	res, err := Ingest(strings.NewReader(in), DefaultOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
