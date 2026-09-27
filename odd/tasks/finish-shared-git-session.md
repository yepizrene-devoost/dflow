# Finish shared Git session

Issue: #49
Branch: `feature/finish-shared-git-session`

## Goal
Reuse one repository-bound Git session across a `dflow finish` run so remote discovery and repository context are not repeatedly resolved, while preserving existing failure recovery and public helper behavior. Also codify the requested agent rule that repository file writes require the agent's own work branch and direct commits on base branches are avoided.

## Scope and constraints
- Agent file creation and modification require an own dflow work branch, including instruction and tracking files. Base branches are integration targets, not authoring branches; `dflow finish` may still merge into configured auto targets after separate authorization.
- Keep public Git helper functions compatible; preserve no-origin behavior, publish order, error diagnostics and restoration on failure.
- Do not push, merge, close the issue, or commit without the separately required authorization.
- Delivery strategy: ask-on-risk; expected authored diff under 400 lines, excluding generated files.
- TDD mode: unknown (no explicit project/session setting identified); run ordinary checks, with `make test` (`go test ./...`) as the repository runner.

## Tasks
- [x] T1 — Codify the own-work-branch-before-any-file-write rule and avoid direct authoring commits on base branches in repository agent instructions; verify aligned wording.
- [x] T2 — Reuse one Git session across finish orchestration with behavior-preserving helper wrappers and focused regression coverage; run focused checks.
- [x] T3 — Run applicable full verification and reconcile evidence.
- [ ] T4 — After separate authorization, commit the work unit and record its commit identity.

## Route and checks
- T1: bounded policy edit, delegated with T2 where shared instruction context helps; inspect exact wording.
- T2: delegated writer (multiple non-trivial implementation/test files); focused `go test ./cmd/tests -run '^TestFinish' -count=1` and `go test ./cmd/gitutils ./pkg/repository`.
- T3: independent delegated verification; `make test`, `git diff --check`, `gofmt -d` on changed Go files, and readback of branch, changed scope and checklist.
- T4: commit only after explicit human authorization; record the SHA in a short follow-up `docs(odd)` commit.

## Evidence
- Initial state: clean `develop`; created `feature/finish-shared-git-session` via `dflow start feat finish-shared-git-session --no-push` before any file write.
- Mapping: #49 repeats `newGitSession()` across finish helpers; session caches origin and pins repository context. The public wrappers can remain for other callers.
- T1: `AGENTS.md`, `.agents/workflows/dflow-workflow.md`, and `.agents/workflows/commit-rules.md` now agree on own-branch-before-any-file-write, base-branch authoring prohibition, and separately authorized `dflow finish` integration exception; parent read back the diff.
- T2: One `GitSession` flows through finish fetch, publish, pull, merge, target push, restoration, and optional delete; public helpers retain standalone wrappers. New integration trace assertion covers one origin probe and four fixed repository discoveries. Worker observed `go test ./cmd/tests -run '^TestFinish' -count=1`, `go test ./cmd/gitutils ./pkg/repository`, and `git diff --check` pass; parent inspected diff.
- T3: independent verifier observed `make test` pass (`cmd/tests` 45.019s; other packages partly cached), `git diff --check` pass, and `gofmt -d cmd/commands/finish.go cmd/gitutils/git.go cmd/gitutils/finish.go cmd/tests/finish_cmd_test.go` produce no output. Exact four-discovery trace count couples the test to validator/config internals; no uncached full rerun, race test, or separate build was done.
- Commit authorization: maintainer explicitly approved staging and committing this work unit on `feature/finish-shared-git-session`; push, merge and issue closure remain unauthorized. T4 remains open only for the permitted follow-up commit-identity record.
- Review declaration: freeze the resulting committed work unit for native review; this is an expectation, not a verdict or an approval.
- Pending: work-unit commit identity; no delivery authorized.
