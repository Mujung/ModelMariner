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
