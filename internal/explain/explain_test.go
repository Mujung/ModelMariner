package explain

import (
	"strings"
	"testing"

	"github.com/Mujung/modelmariner/internal/policy"
	"github.com/Mujung/modelmariner/internal/routing"
)

func TestExplainWinner(t *testing.T) {
	d := routing.Decision{
		Task: "t", Policy: "p", Winner: "premium", Runnerup: "cheap",
		Score: 0.9, Margin: 0.2,
		Evaluations: []policy.Evaluation{
			{Model: "premium", Eligible: true, Score: 0.9, Components: map[string]float64{"quality": 0.9}},
