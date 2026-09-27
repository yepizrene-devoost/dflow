# Finish failure recovery

Issue: #46
Branch: `feature/finish-failure-recovery`

## Goal
Make `dflow finish` failure states explicit and recoverable without pretending mutations are transactional.

## Tasks

- [x] Update project session-start instructions to show the previous-session status and an issue backlog table with number, title, and description.
- [x] Map the current finish orchestration and failure boundaries.
- [x] Define explicit execution states for target synchronization, merge, push, restoration, and merge cleanup.
- [x] Report completed targets and the exact recovery state on target failure.
- [x] Attempt safe restoration of the original branch and report restoration failures.
- [x] Distinguish active merges from ordinary command failures and clean up merge state where safe.
- [x] Add failure-state coverage for target pull/sync failure, target push failure, multiple auto-target partial completion, and merge-abort cleanup.
- [x] Run focused verification and record evidence.
- [ ] Record the completed work-unit commit identity in this checklist.

## Evidence

- Focused tests: `go test ./cmd/commands ./cmd/gitutils ./cmd/tests` — passed; `go test ./cmd/tests -run Finish -count=1` — passed; `go test ./cmd/...` — passed.
- Runtime harness: N/A — this change is covered by repository-level CLI/Git integration tests.
- Commit: pending work-unit commit identity.
