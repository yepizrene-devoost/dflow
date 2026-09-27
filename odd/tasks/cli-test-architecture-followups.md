# CLI test architecture follow-ups

Issues: #50, #51, #48
Branch: `feature/cli-test-architecture-followups`
Base: `fe55d912ae09443c74b6fb9a9aedb690473824f1`

## Objective and scope

Correct two misattached Go documentation comments independently, then improve CLI test isolation and regression coverage without changing CLI behavior. Keep each issue in separate reviewable work units; split #48 by coherent test behavior. No push, finish, issue closure, or release is implied.

## Method and delivery

TDD mode: strict (`~/.gentle-ai/state.json`, `strict_tdd: true`); runner `make test` (`go test ./...`). Comment-only units have no behavior to drive RED; verify with focused package checks and documentation readback. #48 behavioral test changes require observed RED/GREEN where applicable. Route: #50/#51 inline (one mechanical file each); #48 delegated writer (multiple non-trivial files), with independent verification as native assessment directs. Full applicable checks: `go test -count=1 ./...`, `go vet ./...`, `go build ./...`, `git diff --check`. RDD is on; freeze each committed work unit separately via native assessment/review as warranted, not the accumulated branch. Each unit's checklist and review declaration must be in its candidate before freeze; commit identities may be recorded in short follow-up ODD commits before the next freeze.

Forecast: #50 and #51 <20 authored lines each; #48 likely >400 total, so delivery strategy `ask-on-risk` before crossing the threshold. Keep logical slices reviewable; never cut useful tests for a line budget.

## Tasks

- [x] C50 — Relocate `runCapturingGit` comment from `gitResult` to its function; verify attachment and package checks. Work unit: docs(gitutils), issue #50.
- [x] C51 — Relocate `RemoteBranchExists` comment from private helper to exported function; verify attachment and package checks. Work unit: docs(gitutils), issue #51.
- [ ] T48a — Establish isolated CLI test state boundaries with flag/output-format leak guards, choosing the smallest safe reset design; retain existing test behavior. Work unit: test(cli), issue #48.
- [ ] T48b — Move pure planning/validation/error-format contracts into targeted unit tests; retain representative real-Git integration coverage. Work unit: test(cli), issue #48.
- [ ] T48c — Add explicit stderr/failure/platform-capability and remote-heavy command-count regression assertions; streamline redundant integration cases only when equivalent coverage remains. Work unit: test(cli), issue #48.
- [ ] I — Record each work-unit commit identity and native assessment/review outcome before delivery; close via short `docs(odd)` identity commits before freezing subsequent candidates when needed.

## Evidence and next step

C50: `36c51a871dbfd3b615a5a6346730d3f90b19104c`, comment relocation only; independent verifier observed `go test -count=1 ./...`, `go vet ./...`, `go build ./...`, `git diff --check` pass. Native review `review-3641ce1111dbe24c` approved and acknowledged; authority burned, not delivery permission.

C51: comment now immediately precedes exported `RemoteBranchExists`; executable code unchanged. Independent verifier observed the same four checks pass. Freeze declaration: assess C51's own commit from C50's boundary and request native review if offered, without presuming approval. Next: commit/review C51, then T48a. C51 identity and review outcome pending.
