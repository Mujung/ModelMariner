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
in report output paths. Email security concerns rather than opening a
public issue; include a minimal crashing trace and the version tag.
Fixes land in the next patch release and are credited in CHANGELOG.md.