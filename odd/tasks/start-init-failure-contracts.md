# Start and init failure contracts

Issue: #47
Branch: `feature/start-init-failure-contracts`

## Goal
Make `dflow start` and `dflow init` failure states deterministic and recoverable without hiding underlying Git diagnostics. Also add optional issue priority to the session-start backlog table.

## Problem and decision
`start` has partial recovery coverage but some checkout/pull/create errors discard Git's explanation. `init` writes config before creating/publishing branches and does not explain the resulting partial state on failure. Build and validate a draft in memory, then create/publish requested branches, and write `.dflow.yaml` only when the onboarding has completed successfully. On failure, preserve completed branches (including remote refs), stop at the first failure, report the exact recovery path, and allow a retry to reuse existing branches; do not roll back remote work automatically. Reject invalid primary Git refs and generated configuration before any Git/config mutation.

## Scope and constraints
- Work only on this dflow feature branch; do not publish, merge, close #47, or commit without separate authorization.
- Start: pre-creation errors restore the original branch where possible; restoration failure and current branch are visible; post-creation publication failure names the retained branch. Include original Git diagnostics.
- Init: validate the in-memory draft before mutation, gather all prompts before Git mutation, write `.dflow.yaml` only at successful onboarding closure, report local/remote branch progress on failure, reuse existing branches on retry, and give explicit recovery guidance. Avoid destructive rollback. Preserve an existing config when `--force` fails before final write.
- Session opening: show an issue's `priority:*` label in a separate column when present; leave it blank otherwise. Do not infer priority.
- TDD: use RED/GREEN for changed behaviors, based on gentle-ai skill; no repository-level TDD toggle identified. Runner: `go test ./...` (Makefile `make test`).
- Delivery strategy: user requires each work-unit commit to contain at most 500 authored changed diff lines (additions + deletions) and to be reviewed individually, rather than waiting until the branch accumulates a large candidate. Keep work units coherent and compiling; forecast/running PR-slice risk still uses ask-on-risk at ~400 lines. Record each commit and review boundary. Ask before staging and keep the stage checklist closed for any work included in that commit; commit identities may be recorded in a short follow-up `docs(odd)` commit before the review freeze.

## Tasks
- [x] T1 — Update `AGENTS.md` and `.agents/MEMORY.md` session-start rule with optional priority column; verify open-label source and readback. Route: inline, two mechanical documentation edits already understood.
- [x] T2 — Test and harden `start` checkout/pull/create/restore/push failures and invalid normalized names; preserve Git error details and assert HEAD, local/remote refs and recoverability. Route: delegated writer, multi-file tests and code. Check focused `go test ./cmd/tests -run 'TestStart|TestFailingGit'`.
- [x] T3a — Isolate `init` onboarding draft, in-memory config construction and pre-mutation validation in an independently passing ≤500-line candidate with its tests. Route: delegated writer; W2 prepared and verified.
- [x] T3b — Isolate `init` final-only config persistence, retry-safe local/remote branch work, fail-fast progress reports and `--force` guidance in a second independently passing ≤500-line candidate with tests. Route: delegated writer; W3 verified.
- [x] T4 — Run full checks (`go test ./...`, `go vet ./...`, `go build ./...`), inspect diff, and plan ≤500-line coherent commit boundaries. Route: delegated verifier; all work-unit boundaries verified.
- [ ] T5 — With standalone staging/commit authorization, produce ≤500-line work-unit commits and review each candidate without waiting until branch end; record commit identities in the short follow-up `docs(odd)` identity commit allowed by policy before the final freeze. W1/W2 done; W3 and identity record remain.

## Acceptance
- Invalid normalized `start` names and invalid init primary refs change no Git/config state.
- Failure to change/pull base, restore original branch, create new branch, or publish a new branch reports a useful error and the observed state.
- Init failures before onboarding completion leave `.dflow.yaml` absent (or the original config intact under `--force`), identify surviving branches, stop immediately, preserve underlying Git diagnostic, and explain how to retry. A successful retry reuses existing branches and writes the final config.
- A table at every future session start shows priority when `priority:*` exists, otherwise an empty cell.
- Focused and full Go checks pass; no unapproved delivery action.

## Progress and evidence
- Branch created from clean `develop` via `dflow start feat start-init-failure-contracts --no-push`.
- Issue #47 labels include `priority:medium`; labels are available from `gh issue list --state open --json number,title,labels`.
- Read-only map identified existing start tests and missing init wizard/partial-state coverage. User chose preserve-progress, fail-fast semantics, then clarified that `.dflow.yaml` must be written only after successful onboarding; partial Git progress remains retryable.
- Session-start rule now reads optional `priority:*` labels in `AGENTS.md` and `.agents/MEMORY.md`; both files read back and match the open-label source.
- T2 RED: focused tests failed for missing checkout/pull/create Git diagnostics and push recovery guidance. GREEN: `go test ./cmd/tests -run 'TestStart|TestFailingGit' -count=1` passed; `git diff --check` passed (worker). Changes: `cmd/commands/start.go`, `cmd/tests/start_cli_test.go`, 124 additions/4 deletions. Parent inspected diff. Full suite pending.
- T3 RED: focused tests failed for undefined init draft/apply and retry boundaries. GREEN: `go test ./cmd/commands ./cmd/tests -run 'TestInit' -count=1` passed; `git diff --check` passed (worker). Force-mode retry and divergent remote refs corrected; parent inspected implementation and tests. No real PTY wizard integration test yet. Current init code diff 215 additions/147 deletions plus 288 new test lines (~650), so T3 needs two independently passing candidates.
- Full checks on combined worktree: `go test ./...`, `go vet ./...`, `go build ./...`, `git diff --check` all PASS (independent verifier). Repeat checks for each staged candidate.
- Proposed boundaries: W1 T1+T2+ODD document (~180 lines), W2 T3a draft/preflight, W3 T3b persistence/retry. Review each work-unit commit before moving on. W1 review declaration: request native review of this work unit after commit, no verdict or approval is asserted here.
- W2 RED: focused tests exposed the early-save versus late-save boundary in the combined implementation; GREEN: focused init tests and `go test ./...`, `go vet ./...`, `go build ./...` all PASS after isolating draft/preflight while preserving legacy success ordering. W2 code/test diff totals 445 authored lines before this ODD update. W2 review declaration: request native review after this commit, without asserting a verdict. Final-only config persistence is T3b, not claimed for W2.
- W3 RED/GREEN: final-only save and retry tests were first run against W2's legacy early-save ordering; focused suite passed after restoring execution order and adding failure tests. Independent verification initially found raw Git remote inspection ignored `DFLOW_CWD`; a regression test with a decoy repository drove the correction to repository.Session commands. Focused `go test ./cmd/commands ./cmd/tests -run 'TestInit' -count=1`, full `go test ./...`, `go vet ./...`, `go build ./...`, and `git diff --check` PASS. Parent spot check ran `TestInitRemoteBranchInspectionUsesDflowCWD` uncached and observed PASS. W3 code/test diff: 353 authored lines before this ODD update. W3 review declaration: request native review of this work unit after commit; no verdict or approval asserted.
- Next: request standalone W3 staging authorization, then short `docs(odd)` identity record authorization before final freeze; review each committed candidate separately.
