# core-hardening — feature tracking

Branch: `feature/core-hardening` (base: develop)
Issues: #12 (WU1), #14 (WU2), #15 (WU3), #16 (WU4), #17 (WU5); #18 is a tracked follow-up
Status: authorized and implemented; seven work units below. Forge, deploy and TUI stay out.

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
| WU6 | none | repairs the two contracts independent verification refuted | no issue of its own: it corrects the WU1 and WU2 claims rather than tracking a new finding |
| follow-up | #18 | `fix(cli): stop git's own text from reaching the caller` | open; deliberately out of scope here |

Closure: `Closes #15`, `Closes #16` and `Closes #17` ride on their own
work-unit commits, and the closing `docs(odd)` commit carries `Closes #12` and
`Closes #14` because those two fixes were committed before their issues existed.
All five issues therefore close when this branch lands in `develop`, and none of
them needs a manual close afterwards. #18 is a separate follow-up and stays open.

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

6. [x] WU6 — close the contract holes found by verification
   - An independent verification of the committed `develop..HEAD` range refuted two
     claims this branch made. Finding A: a non-zero exit could still leave the
     repository mutated (a failed base pull abandoned the caller on the base branch; a
     failed push left a newly created branch behind without saying so). Finding B: the
     output guard was narrower than the single-stdout-document invariant it claimed.
     Finding C: a JSON invocation could still answer in human format on a flag error.
   - `start` now captures the caller's branch before any checkout and restores it on
     every failure before the new branch exists; a post-creation failure keeps the
     branch and says so. The guard now flags every stdout-targeted writer form while
     leaving `os.Stderr` allowed. `root.Execute` pre-selects the JSON format from the
     raw arguments so a flag-parse error is still JSON.
   - Evidence: real-binary before/after reproductions and planted-bypass guard failures;
     see the WU6 OUTCOME under `## Evidence`.

7. [x] WU7 — address the review advisories
   - The approved native review left four non-blocking advisories. The user chose
     to fix all four now as a separate work unit, accepting that a new commit
     creates a new candidate tree with its own review.
   - `R2-icon-arg-convention`: the output helpers stop guessing the icon from the
     last argument's length. `Error`/`Info`/`Success`/`Warn` always use their own
     default icon and a new `utils.Icon(icon, message, args...)` declares a
     non-default one, so a one-rune value can no longer be swallowed as an icon.
   - `R2-detect-as-predicate`: `flow.IsWorkBranch(cfg, branch)` states the yes/no
     question `status` was asking by discarding a detected branch type.
   - `R2-shared-type-placement`: `targetView`/`targetViews` move to
     `cmd/commands/report.go` as the shared report shape both commands emit.
   - `R2-dryrun-hardcoded`: `finishPlanReport.DryRun` derives from the actual
     `--dry-run` flag instead of a `true` literal.
   - Evidence: real-binary before/after reproduction of the icon defect, plus a
     byte-for-byte old/new comparison of every non-interactive call site; see the
     WU7 OUTCOME under `## Evidence`.

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

CORRECTION (WU6, after independent verification refuted the broader reading): the
"untouched repository" guarantee above holds only for the pre-checkout fail-fast guard. The
same verification showed that a later failure — a failed base-branch pull, or a failed push
after `CheckoutNew` — still exited non-zero with the repository mutated: the caller was
abandoned on the base branch, or the new branch was left behind silently. WU6 replaces the
over-broad claim with the honest contract: **a non-zero exit either leaves the repository as
it was, or states explicitly what it created and left behind.** See the WU6 OUTCOME.
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

### WU6 — verified contract holes

OUTCOME: done. An independent verification of the committed `develop..HEAD` range refuted the
two claims below; WU6 closes them. Two pre-existing limitations the same verification found
stay out of scope and are tracked separately: git's own output from the checkout passthrough
reaching the caller's stdout, and `dflow delete <current-branch> --yes` surfacing a raw git
refusal message. The passthrough plumbing was not touched.

Finding A — a non-zero exit could leave the repository mutated. `start` checked out the base
branch and pulled before its later failure returns, so a failed pull abandoned the caller on
the base branch, and a failed `PushBranch` left a freshly created branch behind without saying
so. The contract is now precise and honest: **a non-zero exit either leaves the repository as
it was, or states explicitly what it created and left behind.** `cmd/commands/start.go`
captures the caller's branch with `gitutils.CurrentBranch()` before any checkout and
centralizes the repair in a `restoreOriginalBranch` closure that reuses the existing
`gitutils.CheckoutExistingBranch`. Every failure in the base-checkout/pull window restores the
original branch before returning the unchanged error, and so does a `CheckoutNew` failure,
which means the new branch was not created. When the restore itself fails, the original error
is still the reported failure and a `utils.Warn` names the branch the caller is now on. The one
post-creation error, a `PushBranch` failure, never deletes the branch and now ends its message
with `; the branch '<name>' was created and remains`, so a caller that sees a non-zero exit
does not blindly retry into "already exists". The successful paths, the non-interactive
fail-fast guard (still first, before anything is touched) and the interactive skipped-push path
(still exit 0) are unchanged.

Finding B — the output guard was narrower than the invariant it claimed. `output_guard_test.go`
flagged only `fmt.Print`/`Printf`/`Println` and builtin `println`, so `fmt.Fprintf(os.Stdout,
...)`, `fmt.Fprintln(os.Stdout, ...)`, `os.Stdout.Write(...)` and
`fmt.Fprintf(cmd.OutOrStdout(), ...)` all passed. The guard now also flags the writer forms
`fmt.Fprint`/`Fprintf`/`Fprintln` when their first argument targets `os.Stdout` or a Cobra
out-writer (`cmd.OutOrStdout()`), and `<writer>.Write(...)` on those targets. Writes explicitly
targeting `os.Stderr` remain ALLOWED and the test documents why: stderr is outside the
single-document stdout contract, so it cannot corrupt machine-readable output. The two existing
`fmt.Fprintf(os.Stderr, ...)` call sites (`cmd/commands/config.go`, `cmd/commands/init.go`) are
unchanged.

Finding C — a JSON invocation could still answer in human format. Cobra parses flags and
validates args before any `PreRunE` runs, so a flag-parse or arity error on a `--json`
invocation fell through to the human renderer. `cmd/root/root.go` extracts the raw-argument
`--json` detection into one `jsonRequested` helper used by both `shouldSkipBanner` and a format
pre-selection at the top of `Execute()`, before `RootCmd.Execute()`. `--json`, `--json=true` and
the `--json=<bool>` form are honoured; an explicit `--json=false` is not a request, matching the
existing banner behaviour. The commands' `PreRunE` declarations are unchanged: both paths set
the same value, and the `PreRunE` remains the per-command declaration of intent.

Files changed: `cmd/commands/start.go`, `cmd/root/root.go`, `cmd/tests/output_guard_test.go`,
`cmd/tests/start_cli_test.go`, `cmd/tests/status_cli_test.go`, `odd/tasks/core-hardening.md`.
`cmd/gitutils/finish.go` was an allowed surface but needed no change.

Checks: `gofmt -l .` empty, `go build ./...` ok, `go vet ./...` ok, `git diff --check` empty,
`go test -count=1 ./...` green (`ok .../cmd/tests 11.750s`). New regression tests:
`TestStartCLI/pull_failure_restores_the_original_branch`,
`TestStartCLI/push_failure_names_the_created_branch`, and `TestJSONFlagParseFailureStaysJSON`
(three subtests).

Real-binary reproduction in scratch repositories under `$(mktemp -d)`, before (HEAD `74641ed`)
and after this change. Before, the pull failure exited 1 and left the caller on `develop`:

```
$ dflow start feat X --no-push < /dev/null      # before
Switched to branch 'develop'
Pulling latest changes from origin...
❌   Failed to pull latest changes from 'develop'
exit=1
$ git branch --show-current
develop
```

After, the same invocation exits 1 and the caller is back on `feature/parent`, with the new
branch absent:

```
$ dflow start feat X --no-push < /dev/null      # after
Switched to branch 'develop'
Pulling latest changes from origin...
Switched to branch 'feature/parent'
❌   Failed to pull latest changes from 'develop'
exit=1
$ git branch --show-current
feature/parent
$ git branch --list feature/X
```

Before, the push failure exited 1 without saying the branch existed; after, the message states
it and the branch remains:

```
$ dflow start feat post-push --from feature/parent --push   # before
Switched to a new branch 'feature/post-push'
✅   Created and switched to branch 'feature/post-push' from 'feature/parent'
Pushing branch 'feature/post-push' to origin...
❌   Failed to push branch 'feature/post-push': failed to push branch 'feature/post-push': exit status 128
exit=1

$ dflow start feat post-push --from feature/parent --push   # after
Switched to a new branch 'feature/post-push'
✅   Created and switched to branch 'feature/post-push' from 'feature/parent'
Pushing branch 'feature/post-push' to origin...
❌   Failed to push branch 'feature/post-push': failed to push branch 'feature/post-push': exit status 128; the branch 'feature/post-push' was created and remains
exit=1
$ git branch --show-current
feature/post-push
```

Guard verification: each bypass form was planted alone in a copy of the repository under
`$(mktemp -d)` and the guard failed naming it (the real tree passes, and
`fmt.Fprintf(os.Stderr, ...)` stays allowed):

```
../commands/zz_guard_probe.go:9:  fmt.Fprintf(...)      # fmt.Fprintf(os.Stdout, ...)
../commands/zz_guard_probe.go:9:  fmt.Fprintln(...)     # fmt.Fprintln(os.Stdout, ...)
../commands/zz_guard_probe.go:6:  Write(...)            # os.Stdout.Write(...)
../commands/zz_guard_probe.go:10: fmt.Fprintf(...)      # fmt.Fprintf(cmd.OutOrStdout(), ...)
fmt.Fprintf(os.Stderr, ...) -> guard PASSED (allowed)
```

JSON contract: every `--json` failure is one parseable document and exits 1, including the
flag-parse and arity errors that used to render as styled text:

```
$ dflow status --json --bogus-flag   # exit 1
{"error":"unknown flag: --bogus-flag"}

$ dflow status --json=true --bogus-flag   # exit 1
{"error":"unknown flag: --bogus-flag"}

$ dflow status --json unexpected-argument   # exit 1
{"error":"unknown command \"unexpected-argument\" for \"dflow status\""}

$ dflow status --json=false --bogus-flag   # exit 1, human by design
❌   unknown flag: --bogus-flag
```

WU6 closes the two refuted claims; the two pre-existing limitations remain tracked separately.

### WU7 — review advisories

OUTCOME: done. All four non-blocking advisories from the approved native review are fixed.
No behaviour, flag, message or icon changed; `go build ./...`, `go vet ./...`, empty
`gofmt -l .`, `git diff --check` and the full suite were green both before and after.

`R2-icon-arg-convention` (WARNING) — the substantive one. `cmd/utils/messages.go` guessed a
custom icon from the last argument's shape (`isCustomIcon`, 1–2 runes), so
`utils.Info("Branch '%s' exists", "x")` consumed the value as an icon and rendered
`%!s(MISSING)`. That heuristic and the last-argument branch in `printWithIcon` are gone.
`Error`, `Info`, `Success` and `Warn` keep their signatures and always use their own default
icon; the new `utils.Icon(icon, formattedMessage, args...)` declares a non-default icon as a
normal parameter, so it can never be inferred. `Spinner.Stop(message, icon ...string)` is
untouched on purpose: its icon is already a declared parameter, not an inference.

Real-binary reproduction of the defect, in scratch repositories under `$(mktemp -d)`, before
(the binary built from the committed `a34cff0` tree via a read-only `git archive` into the
scratch directory) and after (the binary built from this working tree). The command is
`dflow delete x --yes` in a repository with no `origin`, so the message's last argument is the
one-rune branch name:

```
$ dflow delete x --yes          # before
Deleting branch 'x' locally and remotely...
🗑️  Branch 'x' deleted locally.
x   Remote branch '%!s(MISSING)' does not exist. Skipping remote deletion.
exit=0

$ dflow delete x --yes          # after
Deleting branch 'x' locally and remotely...
🗑️  Branch 'x' deleted locally.
ℹ️  Remote branch 'x' does not exist. Skipping remote deletion.
exit=0
```
Before, the one-rune value became the icon and the format argument was dropped; after, it
renders as the value under the level's default `ℹ️` icon. A durable regression test covers the
same path: `TestShortValueArgumentRendersAsValue` in `cmd/tests/icon_chrome_test.go`.

Call sites updated: the two defaults drop the argument
(`utils.Info("Branch '%s' does not exist. Creating...", branch)` and
`utils.Success("Created branch '%s'", branch)`), and the six non-default ones become
`utils.Icon(...)`: the three `📁` messages (`cmd/gitutils/git.go`, `cmd/gitutils/finish.go`),
`✔` in `cmd/gitutils/git.go`, `🚫` in `cmd/commands/delete.go`, `🔧` and `🎉` in
`cmd/commands/init.go`. The `🎉` site at `cmd/commands/init.go:216` was not in the task's
enumeration but is the same positional-icon convention; leaving it would have printed
`%!(EXTRA string=🎉)`, so it was converted too.

Byte-identity proof for the existing call sites. Method: two binaries — the committed
`a34cff0` tree and this working tree — were run against freshly created, identical scratch
repositories for every non-interactive path that reaches a changed call site, and the combined
stdout+stderr and exit code were compared with `diff -u`. All byte-identical:
`delete feature/doomed --yes` (`ℹ️` `Info`), `delete no-such-branch-xyz --yes` (`❌` `Error`),
`start feat child --no-push` in a repository with no origin (`📁` `Icon` on the skipped pull),
`status`, `status --json`, `finish --dry-run`, `finish --dry-run --delete`,
`finish --dry-run --json` (unchanged document, `dry_run` still `true`) and a real `finish` in a
repository with no origin (the other two `📁` `Icon` sites plus `✅` `Success`). The four sites
that are only reachable through the interactive `init` flow (`ℹ️`/`✅`/`✔` in
`CheckOrCreateBranch`, `🔧` and `🎉` in `init`) were proved at the argument level against
`git show a34cff0:<file>`: the old form `Helper(format, args..., "icon")` resolved to exactly
`icon = "icon"` and `args` under the removed branch, which is precisely what the new form
declares, and `printWithIcon`'s render line `"%-3s %s\n"` is unchanged. No other positional
string argument remains anywhere: `grep` finds no `isCustomIcon` and no `Error`/`Info`/
`Success`/`Warn` call whose trailing argument is an icon literal.

`R2-detect-as-predicate` (SUGGESTION). `pkg/flow/workflow.go` gains
`IsWorkBranch(cfg *Config, branchName string) bool`, implemented as `DetectBranchType` plus an
`err == nil` check so the predicate and the detection cannot drift apart.
`cmd/commands/status.go` now asks `if !flow.IsWorkBranch(cfg, branch)` instead of discarding
the returned type. `TestIsWorkBranchMirrorsDetection` pins the agreement for matching and
non-matching branches.

`R2-shared-type-placement` (SUGGESTION). `targetView` and `targetViews` moved from
`cmd/commands/status.go` to the new `cmd/commands/report.go`, which documents them as the
shared machine-readable shape both `status --json` and `finish --dry-run --json` emit. JSON
tags, field order and the non-nil `[]` guarantee are unchanged, and the emitted documents are
byte-identical in the comparison above.

`R2-dryrun-hardcoded` (SUGGESTION). `finishPlanReport.DryRun` is now `dryRun`, the value the
`RunE` already read from the flag, instead of the literal `true`. The only reachable path still
requires `--dry-run` (enforced in `PreRunE`), so the value stays `true` there — confirmed by
the identical `finish --dry-run --json` document — but the field can no longer lie if that
reachability constraint changes.

Files changed: `cmd/utils/messages.go`, `cmd/commands/report.go` (new),
`cmd/commands/status.go`, `cmd/commands/finish.go`, `cmd/commands/delete.go`,
`cmd/commands/init.go`, `cmd/gitutils/git.go`, `cmd/gitutils/finish.go`,
`pkg/flow/workflow.go`, `cmd/tests/icon_chrome_test.go`, `cmd/tests/workflow_test.go`,
`odd/tasks/core-hardening.md`. `cmd/gitutils/finish.go` needed only the two `📁` sites. The
git passthrough plumbing, `cmd/utils/spinner.go`, `pkg/validators`, `README.md` and
`.agents/*` were not touched.

Checks: `go build ./...` ok, `go vet ./...` ok, `gofmt -l .` empty, `git diff --check` empty,
`go test -count=1 ./...` green (`ok .../cmd/tests 11.827s`),
`TestFailurePathRendersOneErrorIconAndNoSuccessChrome` (both subtests),
`TestCommandLayerWritesGoThroughMessagesHelper`,
`TestShortValueArgumentRendersAsValue` and `TestIsWorkBranchMirrorsDetection` all pass. The
existing icon test asserted chrome, not sniffing behaviour, so no assertion needed weakening.

## Independent verification

One read-only verification pass ran over the committed range `develop..HEAD` (then
at `74641ed`) and reported a verdict for each of twelve claimed contracts: ten
verified, two refuted, none left unverified. The two refutations are fixed by WU6:

- refuted: "a non-zero exit leaves no side effect behind" — a failure after the
  base checkout abandoned the caller on another branch, and a post-creation push
  failure did not say the branch had been created;
- refuted: the output guard was narrower than the invariant it claimed, missing
  `fmt.Fprintf(os.Stdout, ...)`, `fmt.Fprintln(os.Stdout, ...)` and
  `os.Stdout.Write(...)`.

Independently reproduced green at that revision: `go build ./...`, `go vet ./...`,
`gofmt -l .`, `go test -count=1 ./...` (`ok .../cmd/tests 10.147s`) and
`git diff --check`. Two pre-existing limitations were deliberately left out of
scope and are tracked as #18: git's own output reaching the caller's stdout through
the checkout passthrough, and `dflow delete <current-branch> --yes` surfacing a raw
Git refusal.

Scope of that verification, stated honestly: it covered `74641ed`. WU6 then changed
`cmd/commands/start.go`, `cmd/root/root.go` and the tests, and those changes carry
their own evidence (build, vet, gofmt, full suite, planted guard probes in throwaway
copies, and real-binary reproductions of both halves of the corrected contract) but
were not re-verified by a second independent pass. The native review over the final
candidate is the next independent check.

## Delivery

Work-unit commits on `feature/core-hardening`, based on `develop` at `af4dc14`:

| Unit | Commit | Subject |
| --- | --- | --- |
| WU1 | `d046078` | `fix(cli): propagate real exit codes` |
| WU2 | `0de2cce` | `feat(cli): detect non-interactive terminals` |
| WU3 | `c18c832` | `refactor: extract the branch planning core` |
| WU3b | `e0ae4ad` | `refactor(cli): route command output through one place` |
| WU5 | `45cd878` | `fix(cli): keep the status icon in the chrome` |
| WU4 | `74641ed` | `feat(cli): expose machine-readable state and validate merge modes` |
| WU6 | `00796e2` | `fix(start): leave the repository as found when a step fails` |

Eight commits, 31 files, roughly 2100 insertions. Each work unit is its own commit
so the branch can be reviewed unit by unit instead of as one change of that size.

Push and `dflow finish` were NOT performed: both remain the user's decisions. This
branch has never been pushed, so `origin` has no copy of it.

## Native review

Reviewed by the native four-lens review under this clone's RDD switch.

- lineage: `review-819ad26b9e0c322b`
- target: `sha256:849d95f8ebbd5d66025654affda3f4c2df64768367182fef31ba2b9426d74a5e`
- approved candidate: commit `8c856c4`, 32 changed paths, 3268 changed lines, tier `high`
- outcome: `approved`; authority burned as `gentle-ai.review-acknowledged/v1`, consuming
  revision `sha256:6a34e8b7582685fd7674b0016d81ef5da6e92704eee4258761940f6503f52e1d`
- lenses: `review-risk`, `review-resilience`, `review-readability`, `review-reliability`

One group attempt was refused at admission on the resilience lens: the reviewer emitted a
payload outside the schema (`json: unknown field "evidence"`), preserved under
`.git/gentle-ai/rejected-results/`. A fresh STATUS reoffered the exact slot and the rerun was
admitted, so the refusal was not deterministic. Every refused byte was discarded rather than
resubmitted.

Four non-blocking advisories were recorded. They opened no correction and are deliberately left
as later work, never as a reason to re-run the review on this candidate:

| id | lens | location | severity |
| --- | --- | --- | --- |
| `R2-detect-as-predicate` | readability | `cmd/commands/status.go:140-142` | SUGGESTION |
| `R2-dryrun-hardcoded` | readability | `cmd/commands/finish.go:95` | SUGGESTION |
| `R2-icon-arg-convention` | readability | `cmd/gitutils/git.go:33` | WARNING |
| `R2-shared-type-placement` | readability | `cmd/commands/finish.go:91` | SUGGESTION |

`R2-icon-arg-convention` concerns the positional icon-argument convention of the output helpers,
the same latent landmine WU3b already flagged: a short string argument sitting in the icon
position is read as an icon instead of as a value.

This section is post-review bookkeeping. At `a34cff0` it was the only file changed after
approval, so the reviewed code was byte-identical to the approved candidate.

WU7 changes the candidate tree after approval. The approved candidate at the time of that
delegation is HEAD `a34cff0` (the review itself bound the reviewed code at `8c856c4`, and
`a34cff0` only records this section), so the WU7 changes described under
`### WU7 — review advisories` form a new tree that carries its own review; the approval does
not extend to it.
