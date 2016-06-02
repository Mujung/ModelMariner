// Package explain turns routing decisions into human-readable rationale. Every
// selection the compiler makes can be traced back to concrete numbers: which
// candidates survived the hard constraints, why the winner outscored the field,
// and which recorded evidence backs the pick. Good explanations are what make a
// routing compiler trustworthy rather than a black box.
package explain

import (
	"fmt"
	"sort"
	"strings"

