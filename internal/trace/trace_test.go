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
		t.Errorf("want internal privacy, got %v", tr.Privacy)
	}
}

func TestIngestSkipsCommentsAndBlanks(t *testing.T) {
	in := "# a comment\n\n" +
		`{"model":"m","task":"t","tokens":{"prompt":1,"completion":1},"cost_usd":0,"latency_ms":1,"quality":0.5,"error":false,"privacy":"public"}` + "\n"
	res, err := Ingest(strings.NewReader(in), DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 1 || len(res.Traces) != 1 {
		t.Fatalf("comments/blanks not skipped: total=%d traces=%d", res.Total, len(res.Traces))
	}
}

func TestValidationRejectsMissingFields(t *testing.T) {
	cases := map[string]string{
		"missing model":   `{"task":"t","tokens":{"prompt":1,"completion":1},"cost_usd":0,"latency_ms":1,"quality":0.5,"privacy":"public"}`,
		"missing tokens":  `{"model":"m","task":"t","cost_usd":0,"latency_ms":1,"quality":0.5,"privacy":"public"}`,
		"negative cost":   `{"model":"m","task":"t","tokens":{"prompt":1,"completion":1},"cost_usd":-1,"latency_ms":1,"quality":0.5,"privacy":"public"}`,
		"missing latency": `{"model":"m","task":"t","tokens":{"prompt":1,"completion":1},"cost_usd":0,"quality":0.5,"privacy":"public"}`,
		"error no kind":   `{"model":"m","task":"t","tokens":{"prompt":1,"completion":1},"cost_usd":0,"latency_ms":1,"quality":0.5,"error":true,"privacy":"public"}`,
		"unknown field":   `{"model":"m","task":"t","tokens":{"prompt":1,"completion":1},"cost_usd":0,"latency_ms":1,"quality":0.5,"privacy":"public","bogus":1}`,
