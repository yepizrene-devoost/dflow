# update-notification — feature tracking

Branch: `feature/update-notification` (base: `develop`)
Closes forge issue #27 (`feat(notify): surface available updates and release changes in the terminal`).

## Design decisions (user-confirmed in session)

- Update flow abort: `dflow update` shows the release-notes summary **before**
  downloading or replacing, then prompts for confirmation. The prompt only
  appears when stdin and stdout are both TTYs (`utils.IsInteractive`); a
  non-interactive run behaves exactly as before, and `--yes` skips the prompt
  from a TTY. Default answer is yes. No prompt in `--json`, `--check`, or the
  already-up-to-date path, and the prompt precedes `installLatest` so an abort
  costs zero downloaded bytes.
- JSON contract unchanged: `dflow update --json` keeps exactly 6 keys. The
  release body is prose truncated with a terminal-tuned rule, so publishing it
  as a machine-readable field would be a lossy artifact pretending to be data;
  consumers have `release_url` for the authoritative body. Machine-readable
  release notes, if ever wanted, belong in a `dflow changelog --json` follow-up
  (listed as an alternative in the issue), not in a truncated field.
- Release-notes source: the GitHub release `body` field. GoReleaser already
  publishes `CHANGELOG.md` as the release body (`--release-notes=CHANGELOG.md`,
  `release.mode: replace`), so no new artifact is needed and nothing is bundled
  into the binary.
- Update-check cache: `$XDG_CACHE_HOME/dflow/update-check.json`, fallback
  `$HOME/.cache/dflow/update-check.json` (exactly the issue's paths, not
  `os.UserCacheDir`, so behavior is identical across macOS/Linux/Windows HOME
  layouts and trivially testable). State: `last_checked`, `latest_version`,
  `release_url`. Writes are best-effort and silent on failure.
- Notification surface: one plain line on **stderr**, after command output,
  from `PersistentPostRun` on the root command (no subcommand defines its own).
  A process-level flag prevents the double render from the nested `Execute()`
  in the bare `dflow` path. Suppression: `--json` (raw-args signal),
  `completion`/`__complete`, `help`, `--version`/`-V`/`version`/`ver`, the
  `update` command itself, a non-empty `DFLOW_NO_UPDATE_CHECK`, and `dev`
  builds (no release provenance). Reuses the exclusion list the banner skip
  already implements instead of duplicating it.
- Background check: launched in `PersistentPreRun` on a buffered channel;
  `PersistentPostRun` waits at most a small grace budget (~150ms) and discards
  a result that is not ready, so a command never waits noticeably and a slow
  check simply skips the notice until the next run. The probe uses its own
  short (~2s) HTTP timeout, separate from the 30s `update` client.
- `dflow update` and `dflow update --check` refresh the cache with the release
  they just learned about, so the background check stays quiet for 24h after
  any explicit check.

## Why this shape

- The `releases/latest` client already exists with an injectable base URL; the
  only additive change is decoding the `body` field, which GitHub always
  returns for the endpoint dflow already queries.
- The notification must never disturb a stdout contract, hence stderr plus the
  raw-args `--json` signal that the banner skip and `jsonRequested` already
  established.
- Truncation is a pure function (lines + char cap) so the summary is
  deterministic and table-testable; the full body stays one link away.

## Tasks

1. [x] Branch, tracking, and the design decisions (this document; created with
   `dflow start feature update-notification --no-push` before any tracked
   write).
2. [x] WU1 — release body + notes truncation (`cmd/selfupdate`)
   - Delegated to a bounded `gentle-ai-worker` (parallel with WU2, disjoint
     surfaces). `release.go` now decodes the release body into `Release.Body`
     (whitespace-trimmed, empty stays empty) with a decoding test; new pure
     `notes.go` (`SummarizeReleaseNotes`: blank-line collapse, trailing trim
     with leading indentation preserved, 15-line cap, 1200-rune cumulative cap,
     single trailing `…` line when anything was dropped; `nil` reserved for
     empty/whitespace-only bodies). Controller-accepted deviations: character
     budget counted in runes (terminal display unit, pinned by test) and a
     single over-budget line returns `["…"]` rather than `nil`. Checks on the
     work tree: `go build ./...`, `go vet ./...`, `gofmt -l .` clean,
     `go test -count=1 ./cmd/selfupdate/...` green alongside WU2's in-flight
     untracked files.
3. [x] WU2 — update-check cache and probe (`cmd/selfupdate`)
   - Delegated to a bounded `gentle-ai-worker` (parallel with WU1, disjoint
     surfaces). New `check.go`: `UpdateCheckState` (`last_checked`,
     `latest_version`, `release_url`), `UpdateCheckCachePath`
     (`$XDG_CACHE_HOME` → `$HOME/.cache`, one injectable `lookupEnv` seam),
     atomic state write (temp file + rename, `ReadUpdateCheckState` treats a
     missing file as first run), `updateCheckTTL` 24h with strict freshness,
     `ShouldProbeUpdate`/`RecordUpdateCheck` orchestration,
     `ResolveBaseURL()` honoring `DFLOW_UPDATE_API_URL`, and
     `ProbeLatestRelease` on a dedicated 2s client reusing
     `NewReleaseClientWithBaseURL`. Controller-accepted deviations: a blank
     `baseURL` falls back to `ResolveBaseURL()` instead of requesting the
     empty string, and env indirection is a single `lookupEnv` var covering
     both cache path and override. Checks on the work tree: `go build ./...`,
     `go vet ./...`, `gofmt -l .` clean, `go test -count=1
     ./cmd/selfupdate/...` green.
4. [x] WU3 — startup notification wiring (`cmd/root`)
   - Delegated to a bounded `gentle-ai-worker` (parallel with W4, disjoint
     surfaces). New `cmd/root/updatenotify.go` owning the whole notification:
     eligibility gates (JSON, raw-args banner-skip list, `DFLOW_NO_UPDATE_CHECK`,
     the `update` command itself, dev provenance), a cap-1 buffered channel
     handoff, fresh-cache answers without network (synthetic release from
     state) and otherwise a background probe that best-effort records the
     check, and a `PersistentPostRun` renderer bounded by a 150ms grace that
     discards a late result until the next run. New `utils.Notice` writes one
     plain icon-free line to stderr, JSON-suppressed. Controller-accepted
     deviations: a second atomic flag (`updateProbeArmed`) so suppressed runs
     skip the grace wait; the probe arms outside the banner's interactive
     guard so piped runs (the audience for the notice) get it, preserving the
     exact banner condition; the once-guard is consumed on every result read,
     not only on render. E2E proves exact stderr line, stdout purity, ordering
     after output, cache reuse (second run: notice, zero server requests),
     env opt-out and dev-marker silence. Checks on the combined tree: `go
     build ./...`, `go vet ./...`, `gofmt -l .` clean, `go test -count=1
     ./cmd/...` green.
5. [x] WU4 — notes in `update --check` and the update flow (`cmd/commands`)
   - Delegated to a bounded `gentle-ai-worker` (parallel with WU3, disjoint
     surfaces). `update.go`: `--yes`/`-y` flag; human-only
     `reportNotesSummary` (summary lines only when the body is non-empty);
     `--check` human report appends the summary before the release-URL line;
     `confirmUpdateInstall` runs summary → guard → `--yes` → `survey.Confirm`
     (Default yes) strictly before `installLatest`, so an abort (printed as
     "update aborted", exit 0) costs zero downloaded bytes; non-interactive
     and JSON paths proceed with no prompt; post-swap order is binary line,
     summary, full URL; best-effort `RecordUpdateCheck` right after
     `LatestRelease()` keeps the startup probe quiet 24h; Long/Example
     updated. JSON contract untouched (6 keys, pinned by a new test with a
     body present). Known gap: the interactive accept/decline branches are
     not exercised (the harness never runs a PTY); deviation flagged for the
     maintainer instead of faked. Checks: `go build ./...`, `go vet ./...`,
     `gofmt -l .` clean, `go test -count=1 -run TestUpdateCLI ./cmd/tests/`
     green (full `./cmd/...` ran green on the worker's tree pre-WU3-merge).
6. [x] WU5 — docs and close-out
   - README `dflow update` section: `--yes`, the notes-before/after flow, the
     confirmation prompt (interactive-only, abort leaves the binary untouched)
     and the startup notification (stderr line, 24h cache paths,
     `DFLOW_NO_UPDATE_CHECK`, dev-marker silence). `.agents/workflows/
     dflow-workflow.md` command row updated. `CHANGELOG.md` Unreleased gains
     the three user-facing entries. Controller fix on the test harness:
     `setUpCLIEnv` now pins `XDG_CACHE_HOME` to a temp dir so the suite never
     writes the host's real cache (a gap the WU4 worker flagged, outside its
     surfaces).

## Work-unit commit identity on `feature/update-notification`

(recorded per work unit as commits land)

| Unit | Commit | Subject |
| --- | --- | --- |
| WU1 | `c6535bc` | `feat(update): surface the release body and a terminal notes summary` |
| WU2 | `3c5a956` | `feat(update): add the cached update check and background probe` |
| WU4 | `0b6deeb` | `feat(update): confirm the install after showing the release notes` |
| WU3 | `0d52b3e` | `feat(notify): announce newer releases on stderr after a command` |
| docs | `d338c27` | `docs: document the update notification and release notes surfacing` |
| docs(odd) | (this commit) | `docs(odd): record update-notification work-unit identity and close the feature` |

## Declared review expectation

The review candidate is the tip of `feature/update-notification` (base
`develop` at `57f0fb2`): every stage above is closed, `go build ./...`,
`go vet ./...`, `gofmt -l .` and `go test -count=1 ./...` ran green on this
exact tree. The review verdict is not recorded here; the native review receipt
and Engram are the record, and the tree stays untouched after approval.
| WU5 | — | — |

## Review history on this branch

(none yet)
