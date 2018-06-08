package routing

import (
	"testing"

	"github.com/Mujung/modelmariner/internal/policy"
	"github.com/Mujung/modelmariner/internal/reliability"
	"github.com/Mujung/modelmariner/internal/trace"
)

func tr(model, task string, cost, lat, qual float64, err bool) trace.Trace {
	return trace.Trace{Model: model, Task: task, CostUSD: cost, LatencyMS: lat, Quality: qual, Error: err, Prompt: 10, Completion: 10}
}

func buildTraces() []trace.Trace {
	var out []trace.Trace
	for i := 0; i < 10; i++ {
		out = append(out, tr("cheap", "t", 0.10, 100, 0.80, false))
		out = append(out, tr("premium", "t", 0.60, 400, 0.98, false))
	}
	return out
}

func TestSimulatePicksWinnerAndReplays(t *testing.T) {
