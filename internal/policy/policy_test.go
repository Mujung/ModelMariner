package policy

import (
	"strings"
	"testing"

	"github.com/Mujung/modelmariner/internal/reliability"
	"github.com/Mujung/modelmariner/internal/trace"
)

func agg(model string, cost, p95, qual, rel float64, priv trace.PrivacyTier) reliability.Aggregate {
	return reliability.Aggregate{
		Model: model, Task: "t", Samples: 10, Successes: 10,
		MeanCostUSD: cost, P95LatencyMS: p95, MeanQuality: qual,
