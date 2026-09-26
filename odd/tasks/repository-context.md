# Repository context (#43)

## Goal
Centralize Git repository/worktree and `.dflow.yaml` discovery so commands invoked from subdirectories and worktrees resolve one canonical context.

## Tasks

- [x] Define canonical repository context and `DFLOW_CWD` semantics
- [x] Integrate config loading and saving with repository context
- [x] Migrate validators and command checks to canonical context
- [x] Migrate Git utilities and raw Git calls to canonical context
- [ ] Run full verification and record ODD evidence

## Acceptance criteria

- Git repositories are discovered from the invocation directory, including Git worktrees and repository subdirectories.
- `.dflow.yaml` is resolved relative to the canonical repository root.
- `DFLOW_CWD` has documented, tested behavior and cannot make validation and config I/O disagree.
- Permission, I/O, and malformed-path errors are returned rather than treated as absence.
- Direct tests cover `EnsureGitRepo`, `EnsureDflowInitialized`, and `EnsureDflowNotInitialized`.

## Evidence

- Branch: `feature/repository-context`
- Commits: `52487fb` (context), `dc77286` (config), `5dfbf07` (validators); Git migration pending commit
- Focused tests: `go test ./pkg/repository` — PASS; `go test ./cmd/utils ./cmd/tests` — PASS; `go test ./pkg/validators ./cmd/tests -run 'Test(Ensure|Dflow|Init)'` — PASS; `go test ./cmd/gitutils ./cmd/utils ./pkg/validators ./cmd/tests` — PASS
- Runtime harness: N/A — repository/config/validator/Git behavior is covered by focused Go tests
