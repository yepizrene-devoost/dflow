# finish-confirmation-contract — feature tracking

Issue: #38 — `feat(finish): give the finish flow the CLI's confirmation contract`
Status: designed; awaiting maintainer authorization (`status:approved`)

## Goal

Make the intentionally conscious `dflow finish` contract explicit: the complete flow, including the explicit `--delete` option, remains deterministic and non-interactive.

## Decision

`dflow finish` does not prompt and does not require `--yes`.

The primary flow remains:

1. Push the work branch.
2. Check out each configured `auto` target when available.
3. Merge the work branch into the `auto` target.
4. Push the target.

`dflow finish --delete` also does not prompt and does not require an additional confirmation flag. Passing `--delete` is the explicit caller intent to remove the work branch. The operation remains safe through its existing ordering and guards:

- Deletion happens only after all automatic target merges and pushes succeed.
- Deletion is skipped when manual targets remain.
- If a merge, conflict, checkout, pull, or target push fails, the work branch is not deleted.
- There is no automatic rollback or automatic merge abort; a failed merge remains available for manual reconciliation or abort.

This differs from standalone `dflow delete`, where deletion is the primary action and the existing interactive confirmation contract remains appropriate.

## Tasks

1. [x] **WU1**: Document the deterministic, non-interactive finish contract
   - State in `finish --help`, README, and the agent workflow documentation that `finish` never prompts.
   - State that `--delete` is explicit intent and does not require `--yes`.
   - Preserve the existing publish-before-merge ordering and the post-success deletion guard.
2. [x] **WU2**: Add regression coverage for the contract
   - `finish --delete` succeeds without a prompt in the non-interactive test harness.
   - A finish without `--delete` remains prompt-free.
   - A merge conflict or non-fast-forward failure leaves the work branch available.
   - A later auto-target failure leaves the work branch available even if an earlier target succeeded.
   - Manual targets continue to prevent deletion.
3. [x] **WU3**: Run focused and full verification
   - `gofmt -l .`, `go build ./...`, `go vet ./...`, `go test -count=1 ./...`, and `git diff --check`.
   - Record evidence before staging a work-unit commit.
4. [x] **WU4**: Review and delivery bookkeeping
   - Close all implementation stages before the work-unit commit, following ODD/RDD ordering.
   - Record the work-unit commit identity and review evidence in this file.

## Out of scope

- No confirmation around pushing the work branch, checking out targets, merging, or pushing auto targets.
- No confirmation around `--delete`.
- No `--yes` flag for `finish`.
- No change to standalone `dflow delete` behavior.
- No automatic rollback or merge-abort behavior.
- No configuration key for confirmation policy.

## Evidence

### Design evidence

Read-only exploration for #38 covered `cmd/commands/{delete,update,finish}.go`, `cmd/utils/tty.go`, the `cmd/tests` harness, and workflow/template references. The maintainer decided that `finish` is inherently a conscious operation and that the explicit `--delete` flag is sufficient intent for branch removal. The existing implementation already places deletion after successful automatic merges and pushes; implementation work must preserve and test that invariant.

### WU1 — document the deterministic, non-interactive finish contract

OUTCOME: done. `cmd/commands/finish.go` now states that `finish` and `finish --delete` are fully non-interactive, that `--delete` is explicit intent, and that deletion happens only after every automatic target merge and push succeeds. README, `.agents/workflows/dflow-workflow.md`, `pkg/agent/generate.go`, and `pkg/agent/testdata/agent_doc.md` carry the same contract. Standalone `dflow delete` documentation was not changed.

### WU2 — regression coverage for the contract

OUTCOME: done. `cmd/tests/finish_cmd_test.go` proves `finish --delete` runs through the existing non-interactive in-process path and adds coverage for a first auto-target merge conflict and a later auto-target conflict. Both cases assert that the work branch remains locally and on origin; the later-target case also asserts the process remains on the failed target for manual reconciliation.

### WU3 — run focused and full verification

OUTCOME: done. Focused verification: `gofmt -l cmd/commands/finish.go cmd/tests/finish_cmd_test.go pkg/agent/generate.go` produced no output; `go test -count=1 ./cmd/tests ./pkg/agent` passed. Full verification: `go test -count=1 ./...`, `go build ./...`, `go vet ./...`, and `git diff --check` all passed.

### WU4 — review and delivery bookkeeping

OUTCOME: done for the implementation candidate. The implementation and verification evidence are complete, and all checklist stages are closed before staging. Recording the work-unit commit identity remains the permitted commit-identity follow-up; native review evidence is derived from the committed candidate and is not written into it.
