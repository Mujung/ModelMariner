package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Mujung/modelmariner/internal/pareto"
	"github.com/Mujung/modelmariner/internal/policy"
	"github.com/Mujung/modelmariner/internal/reliability"
	"github.com/Mujung/modelmariner/internal/routing"
	"github.com/Mujung/modelmariner/internal/trace"
)

const corpus = `{"model":"cheap","task":"t","tokens":{"prompt":10,"completion":10},"cost_usd":0.10,"latency_ms":100,"quality":0.80,"error":false,"privacy":"public"}
{"model":"cheap","task":"t","tokens":{"prompt":10,"completion":10},"cost_usd":0.10,"latency_ms":120,"quality":0.82,"error":false,"privacy":"public"}
{"model":"premium","task":"t","tokens":{"prompt":10,"completion":10},"cost_usd":0.60,"latency_ms":400,"quality":0.98,"error":false,"privacy":"public"}
{"model":"premium","task":"t","tokens":{"prompt":10,"completion":10},"cost_usd":0.60,"latency_ms":420,"quality":0.97,"error":false,"privacy":"public"}`

const policyJSON = `{"policies":[{"name":"quality","preference":{"weights":[{"objective":"quality","weight":1}]}}]}`

func buildReport(t *testing.T) Report {
	t.Helper()
	ing, err := trace.Ingest(strings.NewReader(corpus), trace.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	sum := reliability.Compute(ing.Traces)
	par := pareto.Compute(sum)
	set, err := policy.Load(strings.NewReader(policyJSON))
	if err != nil {
		t.Fatal(err)
	}
	sims := routing.SimulateAll(set, sum, ing.Traces)
	return Build(BuildInput{
		Ingest: ing, Traces: ing.Traces, Reliability: sum,
		Pareto: par, Policies: set, Simulations: sims, Deterministic: true,
	})
}

func TestReportJSONIsDeterministic(t *testing.T) {
	r1 := buildReport(t)
	r2 := buildReport(t)
	b1, err := r1.JSON()
	if err != nil {
		t.Fatal(err)
	}
	b2, err := r2.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b1, b2) {
		t.Fatal("report JSON is not deterministic across identical runs")
	}
}

func TestReportJSONIsValidAndSchema(t *testing.T) {
	r := buildReport(t)
	b, err := r.JSON()
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("report is not valid JSON: %v", err)
	}
	if decoded["schema"] != SchemaVersion {
		t.Errorf("bad schema: %v", decoded["schema"])
	}
	if r.Generated != "" {
		t.Error("deterministic report must omit generation timestamp")
	}
}

func TestReportTextContainsSelection(t *testing.T) {
	r := buildReport(t)
	txt := r.Text()
	if !strings.Contains(txt, "route \"t\" to premium") {
		t.Errorf("text report missing expected routing selection:\n%s", txt)
	}
	if !strings.Contains(txt, "PARETO FRONTIERS") {
		t.Error("text report missing pareto section")
	}
}

func TestCompilePolicyArtifacts(t *testing.T) {
	r := buildReport(t)
	// Rebuild simulations to compile artifacts.
	ing, _ := trace.Ingest(strings.NewReader(corpus), trace.DefaultOptions())
	sum := reliability.Compute(ing.Traces)
	set, _ := policy.Load(strings.NewReader(policyJSON))
	sims := routing.SimulateAll(set, sum, ing.Traces)
	arts := CompilePolicyArtifacts(sims)
	if len(arts) != 1 || len(arts[0].Routes) != 1 {
		t.Fatalf("unexpected artifacts: %+v", arts)
	}
	if arts[0].Routes[0].Model != "premium" {
		t.Errorf("compiled route should target premium, got %s", arts[0].Routes[0].Model)
	}
	b, err := ArtifactsJSON(arts)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(b) {
		t.Error("artifacts JSON invalid")
	}
	_ = r
}
