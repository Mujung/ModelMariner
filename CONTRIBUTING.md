# Contributing to ModelMariner

ModelMariner is deliberately local-first: it routes model versions, traces
decisions to their provenance, and renders reports from traces you already
own. Contributions that keep that scope are welcome.

## Development setup

```bash
git clone https://github.com/Mujung/ModelMariner.git
cd ModelMariner
go build ./...
go test ./...
```

Go 1.24+. Zero runtime dependencies; the dashboard in `dashboard/` is
plain TypeScript with no build step required to run the core.

## Ground rules

- **Provenance or it does not ship.** Every reported number must trace
  back to a recorded trace line. Reports without a provable source line
  are bugs, not features.
- **Deterministic reports.** Same traces + same policy = same bytes.
  No wall-clock in outputs, no map-iteration order in rendered reports.
- **Go stdlib only.** Runtime dependencies stay at zero. The dashboard
  may add dev dependencies but never runtime ones.
- **Tests on behaviour changes.** Each internal package has table tests;
  changes to routing, pareto, or reliability need boundary coverage.

## Commit style

Short imperative subjects (`feat: ...`, `fix: ...`, `docs: ...`). Body only
when the "why" is not obvious from the diff.

## Reporting issues

Include a sanitized trace sample, the policy pack involved, and the exact
report section that looks wrong. Synthetic traces (see `testdata/gen_traces.go`)
are preferred over real data.