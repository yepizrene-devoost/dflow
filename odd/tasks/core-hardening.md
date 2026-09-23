# core-hardening — feature tracking

Branch: `feature/core-hardening` (base: develop)
Issues: #12 (WU1), #14 (WU2), #15 (WU3), #16 (WU4), #17 (WU5)
Status: authorized; scope is the five work units below. Forge, deploy and TUI stay out.

## Issues

Every finding has a tracked issue so it can be closed explicitly once its work
lands in `develop`.

| Unit | Issue | Title | State |
| --- | --- | --- | --- |
| WU1 | #12 | `fix(cli): return a non-zero exit code on failure` | open; keyword carried by the closing `docs(odd)` commit |
| WU2 | #14 | `fix(cli): make progress output and prompts terminal-aware` | open; keyword carried by the closing `docs(odd)` commit |
| WU3 | #15 | `refactor: extract the branch planning core from the command layer` | open; extraction and output routing committed on this branch |
| WU4 | #16 | `feat(cli): expose machine-readable state and validate merge modes` | open; fix committed on this branch |
| WU5 | #17 | `fix(cli): keep the status icon in the chrome instead of the message` | open; fix committed on this branch |

Closure: `Closes #15`, `Closes #16` and `Closes #17` ride on their own
work-unit commits, and the closing `docs(odd)` commit carries `Closes #12` and
`Closes #14` because those two fixes were committed before their issues existed.
All five issues therefore close when this branch lands in `develop`, and none of
them needs a manual close afterwards.

## Goal

Make the dflow core mature enough to be driven by agents and by any non-TTY
caller, before adding forge, deploy and TUI layers on top of it.

## Authorization

Read-only exploration (2026-09-23) recommended a foundation-first sequence and
the user authorized it: delete `feature/dflow-implement-tui` (local + remote; it
was identical to `develop` at `af4dc14`), create this branch from `develop`, and
implement the four work units. Push and `dflow finish` remain separate user
decisions.

Scope extension, recorded rather than assumed: while verifying the branch, the
`delete` failure path was found to render both a duplicated error icon and a
success icon over a failure message, and the first defect was a regression
introduced by WU1 itself. That finding is tracked as issue #17 and fixed as WU5
before WU4, because the machine-readable output mode in WU4 would otherwise have
inherited the malformed error strings.

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

4. [x] WU4 — machine-readable state and validated merge modes (issue #16)
   - `dflow status --json`, `finish --dry-run --json`, and validation of
     `merge_mode` so an unknown value fails loudly instead of silently matching
     neither `auto` nor `manual`.
   - Evidence: JSON output assertions in `cmd/tests`; see the WU4 OUTCOME under `## Evidence`.

5. [x] WU5 — keep the status icon in the chrome (issue #17)
   - Found while verifying the branch, and ordered before WU4 because the JSON
     output mode would otherwise have inherited the malformed error strings.
   - The icon is chrome, never message content: four error strings dropped their
     embedded `❌`, and the failure paths stop the spinner with `Clear()` instead
     of announcing a failure with the default success icon.
   - Evidence: single-`❌` failure output, success chrome preserved, and a real
     binary regression test; see the WU5 OUTCOME under `## Evidence`.

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

WU3b OUTCOME: done (the output-routing half of the same issue). Extended the `cmd/utils`
choke point with the two shapes the command layer still needed. `Plain(formattedMessage,
args...)` prints a formatted line with no icon and a single trailing newline, for output that
is the requested result rather than commentary (the `config get-author` author/email lines,
the raw `config list` listing). `Prompt(label, args...)` prints an inline input label with no
trailing newline, so the cursor stays on the label's line. Both are documented with that
rationale; `Error`/`Info`/`Success`/`Warn` are untouched. Routed every remaining direct
write: `cmd/gitutils/git.go` `CheckOrCreateBranch`'s three messages, `cmd/commands/config.go`
the two prompt labels, the `get-author` author/email lines and the `config list` output (the
raw `git config` output's own trailing newline is trimmed with `strings.TrimSuffix` so `Plain`
owns the terminator, keeping the bytes identical), `cmd/commands/delete.go` the declined
confirmation, and `cmd/commands/init.go` the merge-mode explanation and merge-behaviour
summary blocks plus their blank separator lines. The icon is passed explicitly in
`CheckOrCreateBranch` so a short branch name cannot be read as a custom icon by the shared
chrome and drop the `%s` argument.
Guard: `cmd/tests/output_guard_test.go` parses the Go sources of `../commands` and
`../gitutils` (relative to the package dir) with `go/parser` and fails on any
`fmt.Print`/`fmt.Printf`/`fmt.Println` or builtin `println` call, naming the file:line and
telling the reader to route the write through `cmd/utils`; `cmd/root` and `cmd/utils` are
excluded on purpose. Verified fail-then-pass: adding `fmt.Println("temporary guard probe")` to
`CheckOrCreateBranch` failed the test with `../gitutils/git.go:22: fmt.Println(...)`; removing
it made the test pass.
Chrome normalisations (the shared chrome pads a 1-rune icon to three spaces and a 2-rune icon
to two; message text is unchanged): `✅ Created branch 'x'`, `✔ Branch 'x' exists`,
`🚫 Operation aborted by user.`, `🔧 Dflow supports two types of merge modes:` and
`✅ Merge behavior summary:` each move from one leading space after the icon to three. The
`ℹ️  Branch 'x' does not exist. Creating...` line (two spaces), both prompt labels, the
`get-author` and `config list` output, the explanation continuation lines, the summary detail
lines and every blank separator are byte-identical.
Files changed: `cmd/utils/messages.go`, `cmd/gitutils/git.go`,
`cmd/commands/{config,delete,init}.go`, `cmd/tests/output_guard_test.go` (new).
Checks: `go build ./...` ok, `go vet ./...` ok, `gofmt -l .` empty, `git diff --check` empty,
`go test -count=1 ./...` green (`ok .../cmd/tests 7.855s`). Read-only real-binary spot checks:
`dflow config get-author` and `dflow config list` print the same bytes as before, and the
`utils.Error` render shows the expected three-space 1-rune chrome (1-rune vs 2-rune padding
confirmed). The git passthrough (`cmd.Stdout = os.Stdout` in the checkout helpers),
`cmd/root/completion.go` and `cmd/utils/interrupt.go` were deliberately left untouched. WU4
remains unchecked.

### WU5 — icon is chrome, not message (issue #17)

OUTCOME: done. A failure now renders exactly one `❌` line with no success chrome, and a
success path keeps its own icon.

Before, at `e0ae4ad`:

```
Deleting branch 'no-such-branch-xyz' locally and remotely...
✅   Failed to delete local branch.
❌   ❌ failed to delete local branch 'no-such-branch-xyz': error: branch 'no-such-branch-xyz' not found
```

After, with the real binary built from this working tree:

```
Deleting branch 'no-such-branch-xyz' locally and remotely...
❌   failed to delete local branch 'no-such-branch-xyz': error: branch 'no-such-branch-xyz' not found
```

exit 1. The success path still renders its chrome: `🗑️  Branch 'feature/doomed' deleted
locally.` and `ℹ️  Remote branch 'feature/doomed' does not exist. Skipping remote deletion.`,
exit 0.

Changes: `Spinner.Clear()` added, sharing one `terminate` guard with `Stop` so both terminators
are idempotent, safe to combine, and safe without `Start`; `cmd/utils/spinner.go`. The four
error strings in `cmd/gitutils/git.go` lost their embedded `❌`, as did one stray trailing
newline; its three failure `Stop("Failed to ...")` calls and the three in
`cmd/gitutils/finish.go` became `Clear()`, and the three informational `📁` messages moved the
icon into the helper's icon argument. Files changed: `cmd/utils/spinner.go`,
`cmd/gitutils/git.go`, `cmd/gitutils/finish.go`, `cmd/tests/icon_chrome_test.go` (new).
Checks: `go build ./...` ok, `go vet ./...` ok, `gofmt -l .` empty, `go test -count=1 ./...`
green (`ok .../cmd/tests 8.687s`), and both subtests of
`TestFailurePathRendersOneErrorIconAndNoSuccessChrome` passing.

Process note: the delegated writer stalled on a permission prompt for a guarded `git branch -D`
inside a scratch clone after already writing the three source files and the test, so the
orchestrator completed the verification, the reproduction and this record. HEAD, every local
branch and the reflog were inspected and found intact before resuming; the forced deletion the
prompt referred to was inside a temporary clone outside the repository. WU4 remains unchecked.

### WU4 — machine-readable state and validated modes (issue #16)

OUTCOME: done. The contract a non-interactive caller depends on is now complete: a
deterministic exit code (WU1), terminal-aware output (WU2), a machine-readable mode, and a
`merge_mode` that fails loudly instead of silently skipping a target.

Added `cmd/utils/output.go`: `Format` with `FormatHuman` (the zero value) and `FormatJSON`,
`SetFormat`, `CurrentFormat`, and `EmitJSON`, which writes one compact newline-terminated
JSON document to stdout. `cmd/utils/messages.go` is the single suppression point: `Plain`,
`Prompt` and `printWithIcon` (and therefore `Error`, `Info`, `Success` and `Warn`) return
before writing when the format is `FormatJSON`, so stdout carries the document and nothing
else. Human mode is byte-identical to before. `cmd/root/root.go` renders a failure in JSON
mode as one `{"error": ...}` document while still exiting 1, registers `StatusCmd`, and
`shouldSkipBanner` now skips the banner when `--json` appears in the arguments, including
`--json=true` (parsed with `strconv.ParseBool`, so an explicit `--json=false` still gets the
banner).

New `cmd/commands/status.go`: a config-dependent, `Args: cobra.NoArgs` command whose
`PreRunE` sets the format early enough that a pre-handler failure such as a missing
`.dflow.yaml` is also rendered as JSON (Cobra runs `PersistentPreRun`, then `PreRunE`, then
the `WithChecks` wrapper inside `RunE`). It reports the current branch, the detected type, the
resolved base, every target with its effective merge mode, whether the tree is clean, whether
a merge is in progress, and whether an origin remote exists. A branch that matches no
configured prefix is a successful query: empty `branch_type`, empty `base`, empty `targets`
array, exit 0. Human mode prints the same seven keys as greppable `key: value` lines through
`utils.Plain`, with `targets` rendered as `branch (merge_mode)` pairs and no icons.

`cmd/commands/finish.go` gained `--json`. Its `PreRunE` sets the format first and then rejects
`--json` without `--dry-run` with `--json requires --dry-run: a mutating finish has no JSON
report, so --json would only hide the plan. Run \`dflow finish --dry-run --json\``, so the
rejection is itself a JSON error document and exit 1. With `--dry-run`, JSON mode
short-circuits before any human line and emits `finishPlanReport`; the human dry-run path is
unchanged because the JSON branch returns first. `targetView` is shared by both commands, and
`targetViews`/`nonNilStrings` keep empty lists as `[]` rather than `null`.

`pkg/flow/workflow.go` added `MergeModeAuto`, `MergeModeManual` and `IsValidMergeMode`, replaced
the two comparison literals in `AutoTargets`/`ManualTargets`, and made `ResolveFinishPlan`
return a hard error for an unrecognized effective merge mode: `branch "develop" has invalid
merge mode "pr": use "auto" or "manual"`. An empty value gets the config-aware message
`branch "develop" has merge mode "", which is unset: set workflow.default_merge_mode or the
workflow.branch_rules entry for "develop" to "auto" or "manual"`. `GetMergeModeForBranch`
still returns the raw string so the error can name it. This is the unit's one intentional
behaviour change and it is stated in the README: a config that previously did nothing silently
now fails loudly.

Files changed: `cmd/utils/output.go` (new), `cmd/utils/messages.go`, `cmd/commands/status.go`
(new), `cmd/commands/finish.go`, `cmd/root/root.go`, `pkg/flow/workflow.go`,
`cmd/tests/status_cli_test.go` (new), `README.md`, `.agents/workflows/dflow-workflow.md`.
`cmd/gitutils/finish.go` was an allowed surface but needed no change: no merge-mode literal is
compared there.

Checks: `gofmt -l .` empty, `go build ./...` ok, `go vet ./...` ok, `git diff --check` empty,
`go test -count=1 ./...` green (`ok .../cmd/tests 10.037s`). The new real-binary suite
(`TestStatusAndFinishJSONCLI`, seven subtests) parses stdout with `encoding/json` and a
`json.Decoder` that rejects trailing content, asserts no banner, no carriage return, no ANSI
escape and no icon, and covers the invalid and unset merge modes, human `status` and the
unchanged human `finish --dry-run`.

Manual real-binary run in scratch repositories created under `$(mktemp -d)`, every JSON
document piped through `python3 -m json.tool`:

```
$ dflow status                                              # exit 0
branch: feature/example
branch_type: feature
base: develop
targets: develop (auto)
working_tree_clean: true
merge_in_progress: false
has_origin: true

$ dflow status --json                                       # exit 0
{"branch":"feature/example","branch_type":"feature","base":"develop","targets":[{"branch":"develop","merge_mode":"auto"}],"working_tree_clean":true,"merge_in_progress":false,"has_origin":true}

$ dflow finish --dry-run --json                             # exit 0
{"current_branch":"feature/example","branch_type":"feature","base":"develop","return_branch":"develop","targets":[{"branch":"develop","merge_mode":"auto"}],"auto_targets":["develop"],"manual_targets":[],"delete_requested":false,"dry_run":true}

$ dflow finish --json                                       # exit 1
{"error":"--json requires --dry-run: a mutating finish has no JSON report, so --json would only hide the plan. Run `dflow finish --dry-run --json`"}

$ dflow status --json   # non-work branch main            # exit 0
{"branch":"main","branch_type":"","base":"","targets":[],"working_tree_clean":true,"merge_in_progress":false,"has_origin":true}

$ dflow finish --dry-run   # merge_mode: pr               # exit 1
❌   branch "develop" has invalid merge mode "pr": use "auto" or "manual"

$ dflow finish --dry-run   # merge_mode unset             # exit 1
❌   branch "develop" has merge mode "", which is unset: set workflow.default_merge_mode or the workflow.branch_rules entry for "develop" to "auto" or "manual"
```

No branch was deleted and no destructive Git command was run in the scratch repositories. WU4
is the last unit: every task in this file is now closed.
