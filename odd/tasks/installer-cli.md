# installer-cli — feature tracking

Branch: `feature/installer-cli` (base: `develop`)
Closes forge issue #11 (`feat(installer): add an installer CLI`): the chosen
option is the shell installer — `scripts/install.sh` for Linux/macOS and
`scripts/install.ps1` for Windows, shipped as standalone release assets. The
Go `./installer` package option was rejected: it would duplicate one binary
per platform per release to do what a script does. A follow-up issue records
the `dflow update` requirements (#25) instead of implementing self-update here.

## Why the installers look like this

- Both scripts resolve the latest tag from the `releases/latest` redirect
  (asset URLs need the tag; the checksum asset name embeds the bare version),
  then download the archive **and** `dflow_<version>_checksums.txt`, verify
  sha256 **before** touching the system, and only then install.
- Destinations are user-writable by design — `$HOME/.local/bin` (no sudo) and
  `%LOCALAPPDATA%\Programs\dflow` (no admin) — because a predictable,
  writable install location is also the anchor a future `dflow update` will
  need to replace the binary in place (see #25).
- The shell script never edits rc files; it prints the PATH line. The
  PowerShell script updates the user-scope PATH because Windows has no rc
  file convention.

## Tasks

1. [x] Branch, tracking and the #25 follow-up issue.
2. [x] WU1 — POSIX installer (`scripts/install.sh`)
   - Delegated to a bounded `gentle-ai-worker`; validated end-to-end against
     the **live v0.2.0 release**: resolved latest, verified checksum,
     installed, `dflow version` → `dflow 0.2.0`, rerun replace, error paths
     (bad pin, unknown option, unsupported OS/arch), trap cleanup confirmed.
3. [x] WU2 — Windows installer (`scripts/install.ps1`)
   - Delegated to a bounded `gentle-ai-worker` in parallel; verified by
     reading + structure (brace balance, single dynamic asset template).
     No `pwsh` runtime on this host, so the executable check is deferred;
     the risky part (PS 5.1 vs 7 redirect handling) is dual-path coded.
4. [x] WU3 — release packaging and documentation
   - `.goreleaser.yaml` `release.extra_files` publishes both scripts as
     standalone assets; `goreleaser check` output byte-identical to the
     pre-existing baseline (three unrelated deprecations, exit 2 before and
     after). README installation section rewritten to lead with the
     one-line installers; manual download kept as Option 2.
5. [x] Checks, identity and close-out
   - Controller-run checks on the integrated tree: `go build ./...`,
     `go vet ./...`, `gofmt -l .` clean; `go test -count=1 ./...` green
     (`cmd/gitutils 0.265s`, `cmd/tests 23.241s`, `cmd/utils 0.516s`).
   - `dflow finish --dry-run` evidence is recorded in the follow-up note
     below: the dry-run refuses a dirty tree, so it runs after this commit.

## Work-unit commit identity on `feature/installer-cli`

| Unit | Commit | Subject |
| --- | --- | --- |
| WU1 | `517017f` | `feat(installer): add a posix install script` |
| WU2 | `85a2d85` | `feat(installer): add a windows install script` |
| WU3 | `e8623f8` | `chore(release): publish installer scripts as release assets` |
| docs | `f9b905b` | `docs: lead installation with the new installer scripts` |

This document's own commit is the follow-up `docs(odd)` commit the ODD rule
allows: it carries the identity record above and closes this stage, and it is
the last tracked write before the freeze.

Declared expectation, not a verdict: a native review runs over this frozen
candidate against `develop` (`2ae5032`). Its outcome belongs to the native
receipt and to Engram. Issue #11 stays open until the delivering merge into
`develop` reports `manual_targets:[]`; the closing comment then names that
merge commit, per the issue-closure rule.
