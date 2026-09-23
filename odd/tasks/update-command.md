# update-command — feature tracking

Branch: `feature/update-command` (base: `develop`)
Closes forge issue #25 (`feat(update): add dflow update self-update command`).

## Design decisions (user-confirmed in session)

- `--check` flag is in scope: read-only report (current vs latest, no download,
  no replacement), with the same `--json` document contract as `status`.
- Semver comparison: hand-rolled comparator (~40 lines, table-tested). The repo
  only ever publishes clean `vMAJOR.MINOR.PATCH` tags via GoReleaser; adding
  `golang.org/x/mod` for three numbers is not worth the dependency. Revisit if
  pre-release ordering is ever needed.
- No-release-provenance binaries (version marker `dev`: `go build` / `go
  install`): warn with a clear message and continue, matching the issue's
  "warn instead of silently replacing" wording. A release-built binary carries
  the ldflags-injected marker; a `dev` marker means the version cannot be
  compared and the binary is likely managed by `go install`.
- Follow-up issue #27 records the startup update notification + changelog
  surfacing requirements (out of scope here).

## Why this shape

- Asset naming and checksum conventions are already fixed by
  `scripts/install.sh`: assets `dflow_<OS>_<Arch>.tar.gz` (Windows → `.zip`),
  arch `amd64` → `x86_64`, checksums file `dflow_<version>_checksums.txt`.
- The GitHub API `releases/latest` endpoint needs no token (60 req/h per IP),
  same policy the installer documents.
- Atomic replacement mirrors the installer: stage next to the target, chmod,
  rename. Windows additionally renames the running executable aside first,
  because a running `.exe` cannot be replaced in place.
- All network logic takes an injectable base URL so tests run against
  `httptest` servers: zero real network in the test suite.

## Tasks

1. [x] Branch, tracking, and the #27 follow-up issue.
2. [x] WU1 — selfupdate core package (`cmd/selfupdate`)
   - Delegated to a bounded `gentle-ai-worker`. `release.go` (injectable
     ReleaseClient, 30s timeout, user-oriented 404/403+429/5xx errors),
     `platform.go` (GoReleaser asset naming, switch tables), `compare.go`
     (two-tier comparator, documented fallback), `checksum.go` (sha256sum-format
     parsing, case-insensitive, expected/actual in the mismatch message).
     65 subtests against httptest/temp files, zero real network. Controller-run
     checks on the work tree: `go build ./...`, `go vet ./...`, `gofmt -l .`
     clean, `go test -count=1 ./cmd/selfupdate/...` ok.
3. [x] WU2 — replacement logic (`cmd/selfupdate`)
   - Delegated to a bounded `gentle-ai-worker`; then one controller correction.
     `fetch.go` (streaming download, injectable client, partial-file cleanup),
     `archive.go` (tar.gz/zip extraction by goos, zip-slip guard on every
     entry, `dflow.exe` accepted per install.ps1), `replace.go`
     (`EnsureWritable` + staged atomic `InstallBinary`, Windows rename-aside
     with restore-on-failure), `provenance.go` (`HasReleaseProvenance`).
     Accepted worker deviation: no post-swap `.old` removal — a running Windows
     process keeps it locked anyway, and a stale aside is cleared best-effort
     before each rename, so nothing accumulates. Controller correction: the
     O_WRONLY probe in `EnsureWritable` now treats ETXTBSY as a pass (it is the
     signature of probing the running binary itself, and the rename strategy
     never needs a write-open) — without it, `dflow update` would have refused
     on every Unix host. Checks: `go build ./...`, `go vet ./...`,
     `gofmt -l .`, `GOOS=windows go vet`, `go test -count=1 ./cmd/selfupdate/...` all green.
4. [x] WU3 — command wiring (`cmd/commands/update.go`)
   - Delegated to a bounded `gentle-ai-worker`; then one controller semantics
     correction. `UpdateCmd` with `--force`/`--check`/`--json`, six-key JSON
     report (`current_version`, `latest_version`, `update_available`,
     `updated`, `path`, `release_url`), `--check`+`--force` conflict guard,
     `os.Executable()`+`EvalSymlinks` target resolution, `DFLOW_UPDATE_API_URL`
     override, provenance warning, `EnsureWritable` before any download, temp
     dir cleaned on all paths, spinner skipped in JSON mode. Controller
     correction: no-provenance binaries always report an update available —
     the comparator's string fallback ranked `"dev"` below every digit, which
     would have shown "already up to date" and stranded the user behind a
     `--force` (contradicting the confirmed warn-and-continue decision);
     up-to-date test re-pinned with a stamped equal-version binary. CLI tests
     run a copy of the built binary in a temp install dir against an httptest
     release server (zero real network). Full `go test -count=1 ./...` green
     (`cmd/tests` 32.3s).
5. [x] WU4 — docs and close-out
   - README: new `dflow update` section (flow, flags, provenance warning,
     writability remedy, `DFLOW_UPDATE_API_URL`); `.agents/workflows/
     dflow-workflow.md`: command table row. Controller-run checks on the
     integrated tree: `go build ./...`, `go vet ./...`, `gofmt -l .` clean,
     `go test -count=1 ./...` green (`cmd/selfupdate 0.5s`, `cmd/tests 32.9s`,
     `cmd/utils 0.8s`).

## Work-unit commit identity on `feature/update-command`

(recorded per work unit as commits land)

| Unit | Commit | Subject |
| --- | --- | --- |
| WU1 | `e2cfe29` | `feat(update): add release discovery and checksum verification` |
| WU2 | `d4f0094` | `feat(update): add atomic binary replacement` |
| WU3 | `dd2f27b` | `feat(update): add the dflow update command` |
| docs | (this commit) | `docs: document the update command` |

The `docs(odd)` commit carrying this identity record and closing the last
stage is the follow-up commit the ODD rule allows, and it is the last tracked
write before the freeze.

Declared expectation, not a verdict: a native review runs over this frozen
candidate against `develop` (`7fafc4c`). Its outcome belongs to the native
receipt and to Engram. Issue #25 stays open until the delivering merge into
`develop` reports `manual_targets:[]`; the closing comment then names that
merge commit, per the issue-closure rule.
