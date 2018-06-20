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
	}
	if len(res.Traces) != 1 {
		t.Fatalf("want 1 trace, got %d", len(res.Traces))
	}
	tr := res.Traces[0]
	if tr.Model != "clipper-pro" || tr.Task != "summarize" {
		t.Errorf("bad identity: %+v", tr)
	}
	if tr.Tokens() != 150 {
		t.Errorf("want 150 tokens, got %d", tr.Tokens())
	}
	if tr.Privacy != PrivacyInternal {
