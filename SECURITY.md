# Security Policy

## Supported versions

| Version | Supported |
|---------|-----------|
| latest release on `main` | yes |
| older tags | best effort |

## Reporting a vulnerability

ModelMariner reads local trace files and writes local reports; it performs
no network I/O and executes no downloaded code. The realistic surface is
malformed traces (parser panics, extreme allocations) and path traversal
in report output paths.

**How to report:** email security concerns to the maintainer rather than
opening a public issue. Include a minimal crashing trace (see
`testdata/gen_traces.go` for generating synthetic ones) and the version
tag you tested against.

**What to expect:**

- acknowledgement within 7 days;
- a fix on `main` and a patch release within 30 days for confirmed
  vulnerabilities;
- credit in CHANGELOG.md and the release notes unless you prefer to stay
  anonymous.

## Scope notes

- The dashboard reads only report JSON produced by the core; it executes
  nothing from the trace data itself.
- Reports written with `--out` use the caller's filesystem permissions;
  run with a dedicated user when profiling sensitive fleets.