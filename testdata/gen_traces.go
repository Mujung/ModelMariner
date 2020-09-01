//go:build ignore

// gen_traces.go produces the realistic synthetic trace corpus used by the
// examples and tests. Run with: go run gen_traces.go > testdata/fleet.jsonl
// It is deterministic (seeded PRNG) so the committed corpus is reproducible.
package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"time"
)
