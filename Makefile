# Makefile for modelmariner — offline LLM trace evaluation & routing-policy compiler.
# All targets are self-contained and run fully offline.

BINARY      := modelmariner
CMD         := ./cmd/modelmariner
BIN_DIR     := bin
TRACES      := testdata/fleet.jsonl
