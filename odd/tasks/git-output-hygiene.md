# git-output-hygiene — feature tracking

Branch: `feature/git-output-hygiene` (base: develop)
Issue: #18 — `fix(cli): stop git's own text from reaching the caller`
Status: authorized; scope is the two defects below and their regression coverage.

## Goal

Make the caller's stdout carry dflow's output and nothing else, and make a Git
failure explain itself instead of returning a bare exit status.

## Why this now

Issue #18 was left out of scope by the `core-hardening` branch, which established
the single output choke point. These two paths are the remaining holes in it: one
writes a foreign stream into stdout, and the other turns a diagnosable failure
into an uninformative one.

## Tasks

1. [x] WU1 — capture Git's own output at the five passthrough sites
   - `Checkout` and `CheckoutNew` in `cmd/gitutils/git.go`, plus
     `CheckoutTrackingBranch`, `MergeBranchIntoCurrent` and `AbortMerge` in
     `cmd/gitutils/finish.go`, all wire the child process's stdout and stderr to
     the CLI's own. Everything Git prints therefore lands on the caller's stdout,
     and a future machine-readable mode for those commands would be corrupted by it.
   - The same wiring also destroys diagnostics: because Git's output goes to the
     terminal, the returned error carries only `exit status 1`, so a failed merge
     says nothing about why it failed.
   - Fix both faces with one shared helper that captures both streams: on failure,
     attach the captured diagnostics to the returned error; on success, discard
     stderr (progress and advice noise) and report any captured stdout through the
     CLI output helper, so it stays inside the choke point and can be silenced by
     a machine-readable mode.
   - Evidence: see the WU1 OUTCOME under `## Evidence`.
2. [x] WU2 — refuse deleting the branch you are on, with a dflow message
   - `dflow delete <current-branch> --yes` reaches Git and surfaces its refusal
     verbatim, for a condition dflow can check without asking.
   - Refuse it in the deletion path itself, so every caller is protected, in the
     same spirit as the existing working-tree and merge-in-progress guards.
   - Deleting a branch checked out in another worktree stays Git's call: its
     explanation is now captured, which is the honest fix for that case.
   - Evidence: see the WU2 OUTCOME under `## Evidence`.
3. [x] WU3 — regression coverage and checks
   - A test that a failing Git operation still exits non-zero and its message
     carries Git's explanation rather than only an exit status.
   - A test that deleting the current branch is refused with the dflow message and
     that the branch survives.
   - `go build ./...`, `go vet ./...`, `gofmt -l .`, `go test -count=1 ./...`,
     `git diff --check`.
   - Evidence: see the WU3 OUTCOME under `## Evidence`.

## Out of scope

Forge integration, deploy targets, the TUI, and any change to what the checkout and
merge operations do. Only where their output goes changes.

## Evidence

### WU1 — capture Git's own output at the five passthrough sites

OUTCOME: done. Added one shared helper `runCapturingGit(cmd *exec.Cmd) error` in
`cmd/gitutils/git.go`. It captures both streams; on failure it returns
`fmt.Errorf("%s: %w", diagnostics, err)` with the trimmed stderr, falling back to
stdout when stderr is empty, so each caller's existing message and `%w` wrapping
keeps the reason; on success it drops stderr and reports any captured stdout
through `utils.Plain`, one line at a time. Routed `Checkout` and `CheckoutNew`
(`cmd/gitutils/git.go`) plus `CheckoutTrackingBranch`, `MergeBranchIntoCurrent`
and `AbortMerge` (`cmd/gitutils/finish.go`) through it, keeping every caller's
message and wrapping, and removed the now-unused `os` imports.

One deliberate refinement beyond stream wiring: the three checkout invocations now
pass `git checkout --quiet`. Git writes its branch-status advice
(`Your branch is ahead of 'origin/develop' by 1 commit.`) to stdout, which the
helper's stdout-reporting rule would otherwise re-emit — precisely the leak issue
#18 reports. `--quiet` empties checkout's success output at the source, while a
failed checkout still prints its explanation on stderr (verified). The merge path
is untouched, so `Merge made by ...` and its diffstat are preserved.

Files changed: `cmd/gitutils/git.go`, `cmd/gitutils/finish.go`,
`cmd/tests/git_output_hygiene_test.go` (`TestFailingGitOperationMessageCarriesGitExplanation`,
`TestMergeConflictErrorCarriesGitExplanation`, `TestFinishMergeFailureCarriesGitExplanation`,
`TestStartDoesNotLeakGitText`, `TestFinishSuccessPreservesGitMergeSummary`).
Checks: `go build ./...` ok, `go vet ./...` ok, `gofmt -l .` empty, `git diff --check`
empty, `go test -count=1 ./...` green (`ok .../cmd/tests 15.747s`). Real-binary
before/after on the same blocked-checkout fixture: before it printed Git's raw
stderr and ended with `failed to checkout branch "side": exit status 1`; after it
ends with `failed to checkout branch "side": error: Your local changes to the
following files would be overwritten by checkout: ... Aborting: exit status 1`, and
Git's raw text is gone. Issue reproduction `dflow start feat demo --no-push > out.log
2>&1; grep -c "Your branch is" out.log`: before 1, after 0.

### WU2 — refuse deleting the branch you are on

OUTCOME: done. `gitutils.Delete` checks `CurrentBranch()` before the spinner is
created and returns `cannot delete branch '<branch>' because it is the branch you
are currently on`; nothing is announced, the command exits non-zero, and the branch
survives. A branch checked out in another worktree stays Git's call, whose
explanation WU1 now captures. `README.md` documents the refusal.

Files changed: `cmd/gitutils/git.go`, `README.md`,
`cmd/tests/git_output_hygiene_test.go` (`TestDeleteCurrentBranchIsRefusedWithDflowMessage`).
Checks: real binary `dflow delete <current> --yes` exits 1 with dflow's own message
(no `used by worktree` wording), and `git branch --list` still shows the branch.

### WU3 — regression coverage and checks

OUTCOME: done, with one documented boundary. Added `cmd/tests/git_output_hygiene_test.go`
with six real-process tests:
`TestFailingGitOperationMessageCarriesGitExplanation` (a blocked checkout exits
non-zero and the message carries Git's explanation, not only `exit status 1`),
`TestMergeConflictErrorCarriesGitExplanation` (the merge error contains `CONFLICT`
and the conflicting path), `TestFinishMergeFailureCarriesGitExplanation` (a real
`dflow finish` merge failure exits non-zero and its message carries Git's own
explanation), `TestDeleteCurrentBranchIsRefusedWithDflowMessage`,
`TestStartDoesNotLeakGitText` (no `Switched to`, `Already on` or `Your branch is`
in dflow's output), and `TestFinishSuccessPreservesGitMergeSummary` (the diffstat
survives the helper).

Boundary: a *content conflict* reached through `dflow finish` is replaced by
dflow's own conflict message inside `cmd/commands/finish.go` whenever a merge is in
progress, so that one CLI message does not include Git's captured diagnostics. The
Git-exact conflict wording is asserted on the merge error (the layer WU1 owns), and
a merge failure that reaches the CLI unswallowed (unrelated histories) is asserted
end-to-end. `cmd/commands/finish.go` is outside this work unit's allowed edit
surfaces; wrapping the Git diagnostics into that message is the follow-up.

Checks: `go build ./...` ok, `go vet ./...` ok, `gofmt -l .` empty, `git diff --check`
empty, `go test -count=1 ./...` green. Existing contract suites re-run green:
`TestCommandLayerWritesGoThroughMessagesHelper`,
`TestFailurePathRendersOneErrorIconAndNoSuccessChrome`,
`TestShortValueArgumentRendersAsValue`, `TestStatusAndFinishJSONCLI`,
`TestJSONFlagParseFailureStaysJSON`. `dflow finish --dry-run --json` still emits
exactly one JSON document (manual run plus `TestStatusAndFinishJSONCLI/finish_--dry-run_--json`).

## Native review

The branch was reviewed as candidate `review-dd4f7244f0a3c9bc` over `develop..f038e5d`:
tier `high`, 5 changed paths, 502 changed lines, 4 of 4 lenses admitted, **approved**, and
the authority burned as `gentle-ai.review-acknowledged/v1`.

One non-blocking SUGGESTION was recorded: `R2-001`, readability, on the current-branch
guard in `Delete`. The guard reads `if current, err := CurrentBranch(); err == nil &&
current == branch`, so a failed lookup silently skips it.

Resolution: the behaviour is kept and the intent is now documented. A failed lookup must
not turn a deletion Git would have allowed into a refusal caused by an unrelated problem,
so the guard stands aside deliberately and the error is not propagated. The comment in
`Delete` states that. No behaviour changed, no correction was opened, and the approved
receipt stands.
