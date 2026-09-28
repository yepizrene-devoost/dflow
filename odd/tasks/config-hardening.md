# Configuration hardening (#44)

## Goal
Validate `.dflow.yaml` completely at load time, define legacy/nested precedence, and persist configuration atomically without silently losing malformed data.

## Tasks

- [x] Specify and implement config validation with path-specific errors
- [x] Harden legacy/nested YAML decoding and reject malformed or contradictory rules
- [x] Make config writes atomic and preserve wrapped filesystem errors
- [x] Add focused failure-path and compatibility tests
- [x] Run full verification and record ODD evidence

## Acceptance criteria

- `LoadConfig` validates parsed configuration before returning it.
- Legacy and nested formats have explicit deterministic precedence; contradictory or malformed values fail with useful paths.
- Invalid prefixes, empty bases, blank/duplicate targets, invalid merge modes, nil configs, and overlapping prefixes are rejected or handled explicitly.
- `SaveConfig` writes through a same-directory temporary file and atomic rename; failed writes do not replace the existing configuration.
- Tests cover malformed YAML, compatibility, validation failures, and write failure paths.

## Evidence

- Branch: `feature/config-hardening-44`
- Commits: `5dbeb29` (`fix(config): harden configuration validation and persistence`)
- Focused tests: `go test ./pkg/flow ./cmd/utils` — PASS
- Full verification: `go test ./...`, `go vet ./...`, `go build ./...`, `git diff --check` — PASS
- Runtime harness: N/A — behavior is covered by Go tests
