# core-hardening — feature tracking

Branch: `feature/core-hardening` (base: develop)
Issues: #12 (WU1), #14 (WU2), #15 (WU3), #16 (WU4)
Status: authorized; scope is the four work units below. Forge, deploy and TUI stay out.

## Issues

Every finding has a tracked issue so it can be closed explicitly once its work
lands in `develop`.

| Unit | Issue | Title | State |
| --- | --- | --- | --- |
| WU1 | #12 | `fix(cli): return a non-zero exit code on failure` | open; fix committed, keyword predates the issue-independent work |
| WU2 | #14 | `fix(cli): make progress output and prompts terminal-aware` | open; fix committed on this branch |
| WU3 | #15 | `refactor: extract the branch planning core from the command layer` | open; split into extraction + output routing; extraction implemented on this branch |
| WU4 | #16 | `feat(cli): expose machine-readable state and validate merge modes` | open; pending |

Closure: the WU3 and WU4 work-unit commits carry `Closes #15` and `Closes #16`.
The WU1 and WU2 fixes were committed before #14 existed, so the follow-up
`docs(odd)` commit on this branch carries `Closes #12` and `Closes #14`; all four
issues therefore close when this branch lands in `develop`, and none of them
requires a manual close afterwards.

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

2. [x] WU2 — TTY awareness
   - `cmd/utils/spinner.go` writes `\r<frame> <message>` every 100ms
     unconditionally, so a captured stdout prints every frame on its own line;
     `Stop` additionally emits ANSI `\r\033[K`. The banner and the prompts make
     the same terminal assumption.
   - Add a single `utils.IsInteractive()` helper; when not interactive the
     spinner emits one plain line, the banner is suppressed, and prompts fail
     fast with an actionable message and a non-zero exit instead of hanging or
     degrading silently.
   - `delete` gains a `--yes`/`-y` flag so a fail-fast confirmation message can
     name a flag that exists; `start` fails fast naming `--push`/`--no-push`.
   - Evidence: non-TTY CLI tests assert no `\r`, no ANSI escape, no banner and a
     single pull status line, plus non-zero exits naming `--yes`, both push
     flags and the interactive `init` requirement; see the WU2 OUTCOME under
     `## Evidence`.

3. [x] WU3 — extract the pure core into `pkg/flow`
   - Move the planning and decision logic (config schema and YAML compatibility,
     branch types, finish planning) out of `cmd/utils` into a pure package, so
     CLI, future TUI and future forge/deploy clients share one source of truth.
     The unit ships as two commits for reviewability: this extraction commit, and
     a following output-routing commit under the same issue that replaces direct
     `fmt.Printf` output with an output sink.
   - Evidence: `pkg/flow` builds with no I/O imports, no `cmd/*` dependency and no
     cycle; every moved body is byte-identical to its original; the existing suite
     stays green with import/qualifier changes only; see the WU3 OUTCOME under
     `## Evidence`.

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

### WU2 — TTY awareness

OUTCOME: done. Added `utils.IsInteractive()` (both stdin and stdout must be a terminal,
via `golang.org/x/term.IsTerminal`) and gated every terminal assumption on it. When
non-interactive the spinner prints the status message once on `Start` and a final icon
plus message on `Stop` with no frames, no `\r` and no ANSI escape (and no double-close,
since `Stop` is idempotent); the banner is skipped; and prompts fail fast with an
actionable error that the WU1 single render point turns into exit 1. `delete` gained a
`--yes`/`-y` flag and its prompt error path now returns the error instead of reporting
success; `start` names `--push`/`--no-push`; `config set-author` names the argument or
`--email`; `init` states it requires a terminal.
Files changed: `cmd/utils/tty.go` (new), `cmd/utils/spinner.go`, `cmd/root/root.go`,
`cmd/commands/{start,delete,config,init}.go`, `cmd/tests/start_cli_test.go`, `go.mod`
(`golang.org/x/term` promoted to a direct requirement; `go.sum` unchanged).
Checks: `go build ./...` ok, `gofmt -l .` empty, `go test -count=1 ./...` green
(`ok .../cmd/tests 8.449s`), `git diff --check` empty. New subprocess assertions cover
clean non-interactive output (no `\r`, no `\x1b[`, no banner, one pull status line) and
fail-fast non-zero exits naming `--yes`, `--push`/`--no-push` and `init`'s terminal
requirement. WU3–WU4 remain unchecked.

FOLLOW-UP (review defects, same work unit): fixed two WU2 defects and synced the docs.
(1) `start`'s non-interactive guard now runs at the top of `RunE`, next to the
`--push`/`--no-push` conflict check and before `utils.LoadConfig`, every checkout, the
pull and `gitutils.CheckoutNew`, so a refused invocation leaves the repository untouched
instead of creating the branch and then exiting non-zero (which made a retry fail with
"branch already exists"). (2) The interactive publish-prompt failure path no longer
returns an error: it prints a warning that the branch WAS created but the push was
skipped plus the exact `git push -u origin <branch>` command, and returns `nil`, so a
command whose primary side effect succeeded exits 0. Synced `README.md` (start fail-fast
paragraph, `delete` `--yes`/`-y`, the non-TTY progress/banner note) and
`.agents/workflows/dflow-workflow.md` (`delete [--yes]` row and the chained-branch
paragraph). Added the Defect 1 regression test in `cmd/tests/start_cli_test.go`:
`dflow start feat no-flags` with no push flags exits non-zero and `feature/no-flags` is
absent (`git branch --list`, `git rev-parse --verify`), with the fixture branch and HEAD
unchanged.
Files changed: `cmd/commands/start.go`, `cmd/tests/start_cli_test.go`, `README.md`,
`.agents/workflows/dflow-workflow.md`.
Checks: `go build ./...` ok, `gofmt -l .` empty, `go test -count=1 ./...` green
(`ok .../cmd/tests 7.914s`), `git diff --check` empty. Manual real-binary run in a scratch
clone outside the repository: `dflow start feat scratch-no-flags` exited 1 with `cannot
prompt to publish the new branch without an interactive terminal; pass --push to publish
or --no-push to keep it local`, branch absent, `develop` still checked out, nothing
published. No PTY test for the interactive skipped-push path: it is not testable with the
current harness. WU3–WU4 remain unchecked.

### WU3 — extract the pure core

OUTCOME: done (the extraction half; the output sink is the follow-up half). Created the
pure `pkg/flow` package and moved the branching domain into it: the config schema and its
YAML compatibility rules (`Config`, `BranchFlowRule`, `FlowConfig`, `WorkflowConfig`,
`WorkflowBranchRule`, their `UnmarshalYAML`/`MarshalYAML` methods and the `uniqueStrings`
helper, legacy flat-format support included), the branch model (`BranchType` plus its four
constants, `ParseBranchType`, `DetectBranchType`, `GetBranchPrefix`, `GetFlowRule`) and
finish planning (`FinishTarget`, `FinishPlan`, `ResolveFinishPlan`, `AutoTargets`,
`ManualTargets`, `GetMergeModeForBranch`). `cmd/utils/workflow.go` is deleted and
`cmd/utils/config.go` keeps only `LoadConfig`/`SaveConfig` (now `*flow.Config`, banner
comment and error strings byte-identical) next to the existing message, spinner and TTY
helpers. `cmd/commands/*` and `cmd/tests/*` changed by import path and package qualifier
only, with no assertion touched; `start_cli_test.go` needed no edit because it reaches the
moved types through the `startTestConfig` helper. `pkg/flow` imports only `fmt`, `slices`,
`strings` and `gopkg.in/yaml.v3`: no I/O, no printing, no `cmd/*` dependency and therefore
no cycle.
Files changed: `pkg/flow/{doc,config,workflow}.go` (new), `cmd/utils/config.go`,
`cmd/utils/workflow.go` (deleted), `cmd/commands/{init,start,finish}.go`,
`cmd/tests/{finish_cmd,start_cmd,utils,workflow}_test.go`, `.agents/MEMORY.md`.
Checks: `go build ./...` ok, `go vet ./...` ok, `gofmt -l .` empty, `git diff --check`
empty, `go test -count=1 ./...` green (`ok .../cmd/tests 8.022s`). No-behaviour-change
proof: diffing the moved line ranges against their originals is empty (byte-identical
bodies), and the whole-tree multiset of Go string literals is unchanged apart from import
paths — the single removed literal is the `cmd/utils` import and the added ones are eight
`pkg/flow` imports plus `gopkg.in/yaml.v3`, which moved with the YAML methods. No
user-facing message, error string or generated YAML changed. WU4 remains unchecked.
