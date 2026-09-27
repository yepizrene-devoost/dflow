# Git execution and remote operations (#45)

## Goal
Unify Git subprocess execution, preserve actionable diagnostics, and reuse remote observations during multi-step workflows.

## Tasks

- [x] Define a context-aware Git runner with consistent stdout/stderr capture and non-interactive behavior
- [x] Reuse repository and remote observations across multi-target Git workflows
- [x] Add regression and command-count tests for diagnostics and redundant probes
- [x] Commit the work unit after explicit user authorization
- [x] Run verification and record ODD evidence

## Acceptance criteria

- Git operations use one shared execution path with consistent captured diagnostics.
- Pull, fetch, push, and remote lookup failures preserve Git's actionable stderr.
- Automation and CI use non-interactive Git behavior.
- Multi-step workflows avoid repeated origin probes and redundant fetches where the result is reusable.
- Tests assert diagnostics preservation and meaningful command sequences/counts without weakening existing repository/worktree behavior.

## Out of scope

- Changing branch, merge, or finish semantics beyond execution/reuse behavior.
- Adding provider-specific remote integrations.
- Reworking unrelated CLI output contracts.

## Evidence

Implementation is complete on `feature/git-execution-45`.

- Shared Git execution captures stdout/stderr, sets `GIT_TERMINAL_PROMPT=0`, and preserves actionable diagnostics for pull, fetch, push, remote lookup, and Git config operations.
- Repository sessions reuse canonical discovery; Git workflows cache the `origin` probe across checkout/pull/publish steps.
- Regression coverage includes push diagnostics/context, branch-specific pull output, and a traced assertion that `PullBranch` performs one origin probe.
- Checks: `go test ./...`, `go vet ./...`, `go build ./...`, `git diff --check` — PASS.
- Runtime harness: N/A — existing real temporary-repository integration tests exercise Git behavior.
- Work-unit commit: `bcbd5b6` (`chore(git): unify execution and remote operations`).
- Commit identity: recorded by the follow-up ODD evidence commit.
