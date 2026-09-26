# Repository context (#43)

## Goal
Centralize Git repository/worktree and `.dflow.yaml` discovery so commands invoked from subdirectories and worktrees resolve one canonical context.

## Tasks

- [x] Define canonical repository context and `DFLOW_CWD` semantics
- [ ] Integrate config loading and saving with repository context
- [ ] Migrate validators and command checks to canonical context
- [ ] Run full verification and record ODD evidence

## Acceptance criteria

- Git repositories are discovered from the invocation directory, including Git worktrees and repository subdirectories.
- `.dflow.yaml` is resolved relative to the canonical repository root.
- `DFLOW_CWD` has documented, tested behavior and cannot make validation and config I/O disagree.
- Permission, I/O, and malformed-path errors are returned rather than treated as absence.
- Direct tests cover `EnsureGitRepo`, `EnsureDflowInitialized`, and `EnsureDflowNotInitialized`.

## Evidence

- Branch: `feature/repository-context`
- Commits: pending (commit authorization still required)
- Focused tests: `go test ./pkg/repository` — PASS
- Runtime harness: N/A — package-level repository discovery has no separate runtime boundary
