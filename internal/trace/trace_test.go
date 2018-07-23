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
	}
	for name, line := range cases {
		t.Run(name, func(t *testing.T) {
			res, err := Ingest(strings.NewReader(line), DefaultOptions())
			if err != nil {
				t.Fatalf("ingest should not hard-fail in skip mode: %v", err)
			}
			if len(res.Traces) != 0 {
				t.Fatalf("expected rejection, got trace: %+v", res.Traces)
			}
			if res.Rejected != 1 {
				t.Fatalf("expected 1 rejection, got %d", res.Rejected)
			}
		})
	}
}

func TestStrictModeAborts(t *testing.T) {
	in := `{"model":"","task":"t","tokens":{"prompt":1,"completion":1},"cost_usd":0,"latency_ms":1,"quality":0.5,"privacy":"public"}`
	opts := DefaultOptions()
	opts.SkipInvalid = false
	_, err := Ingest(strings.NewReader(in), opts)
	if err == nil {
		t.Fatal("expected error in strict mode")
	}
}

func TestQualityClamp(t *testing.T) {
	in := `{"model":"m","task":"t","tokens":{"prompt":1,"completion":1},"cost_usd":0,"latency_ms":1,"quality":1.5,"privacy":"public"}`
	res, err := Ingest(strings.NewReader(in), DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Traces) != 1 || res.Traces[0].Quality != 1.0 {
		t.Fatalf("quality not clamped to 1.0: %+v", res.Traces)
	}
}

func TestPrivacyTierParsing(t *testing.T) {
	for _, name := range []string{"public", "internal", "confidential", "restricted"} {
		tier, err := ParsePrivacyTier(strings.ToUpper(name))
		if err != nil {
			t.Fatalf("parse %q: %v", name, err)
		}
		if tier.String() != name {
			t.Errorf("round trip failed: %q -> %v -> %q", name, tier, tier.String())
		}
	}
	if _, err := ParsePrivacyTier("nonsense"); err == nil {
		t.Error("expected error for unknown tier")
