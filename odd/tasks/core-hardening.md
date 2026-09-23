# core-hardening — feature tracking

Branch: `feature/core-hardening` (base: develop)
Issues: #12 (real exit codes), plus the non-TTY defect found during exploration
Status: authorized; scope is the four work units below. Forge, deploy and TUI stay out.

## Goal

Make the dflow core mature enough to be driven by agents and by any non-TTY
caller, before adding forge, deploy and TUI layers on top of it.

## Authorization

Read-only exploration (2026-09-23) recommended a foundation-first sequence and
the user authorized it: delete `feature/dflow-implement-tui` (local + remote; it
was identical to `develop` at `af4dc14`), create this branch from `develop`, and
implement the four work units. Push and `dflow finish` remain separate user
decisions.

## Why this order

The domain model is already data-driven (`BranchType` -> `FlowRule{Base,
FinishTargets}` -> `FinishPlan.AutoTargets()/ManualTargets()`, resolved by
`ResolveFinishPlan`), but it is buried inside `cmd/commands/*` next to inline
prompts and direct `fmt.Printf` output. Widening that seam first is what allows
the forge, deploy and TUI layers to be added later without duplicating
decisions in each client.

## Tasks

1. [x] WU1 — propagate real exit codes (issue #12)
   - Root cause: `validators.WithChecks` catches the handler error and returns
     `nil`, so every command reports failure in prose and still exits 0.
   - Add `SilenceErrors`/`SilenceUsage` on the root command so Cobra does not
     double-print, and have every command return its error after `utils.Error`.
   - Evidence: exit-code assertions in `cmd/tests`; see the WU1 OUTCOME under `## Evidence`.

2. [ ] WU2 — TTY awareness
   - `cmd/utils/spinner.go` writes `\r<frame> <message>` every 100ms
     unconditionally, so a captured stdout prints every frame on its own line;
     `Stop` additionally emits ANSI `\r\033[K`. The banner and the prompts make
     the same terminal assumption.
   - Add a single `utils.IsInteractive()` helper; when not interactive the
     spinner emits one plain line, the banner is suppressed, and prompts fail
     fast with an actionable message and a non-zero exit instead of hanging or
     degrading silently.
   - Evidence pending: non-TTY CLI test asserting single-line output.

3. [ ] WU3 — extract the pure core into `pkg/flow`
   - Move the planning and decision logic out of `cmd/utils` and replace direct
     `fmt.Printf` output with an output sink, so CLI, future TUI and future
     forge/deploy clients share one source of truth.
   - Evidence pending: core package builds without depending on stdout; existing
     tests stay green.

4. [ ] WU4 — machine-readable state and validated merge modes
   - `dflow status --json`, `finish --dry-run --json`, and validation of
     `merge_mode` so an unknown value fails loudly instead of silently matching
     neither `auto` nor `manual`.
   - Evidence pending: JSON output assertions.

## Out of scope

- Forge integration behind a `gh`-backed provider and a third `merge_mode: pr`.
- Envoyer deploy targets and the finish-then-deploy flow.
- Any TUI: it is a later client of the core extracted in WU3.

## Evidence

### WU1 — propagate real exit codes (issue #12)

OUTCOME: done. `validators.WithChecks` no longer swallows the handler error, every command
failure path returns an error instead of `utils.Error(...); return nil`, and `RootCmd` sets
`SilenceErrors`/`SilenceUsage` so `root.Execute()` renders the single `❌` line and exits 1.
Files changed: `pkg/validators/validators.go`, `cmd/root/root.go`,
`cmd/commands/{start,finish,init,config}.go`, `cmd/tests/{start_cmd,start_cli}_test.go`.
Checks: `go build ./...` ok, `gofmt -l .` empty, `go test -count=1 ./...` green
(`ok .../cmd/tests 7.233s`), `git diff --check` empty; new CLI assertions cover exit 1 on
failure (single render) and exit 0 on success. WU2–WU4 remain unchecked.
