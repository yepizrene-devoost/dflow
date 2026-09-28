# Prepare v0.5.0 release

Issue: #56
Branch: `release/v0.5.0` (local, based on `develop` at `4511a5e2a1c7db6c451ddb305351d1dca49a54ed`)
Last published release: `v0.4.1` (`fd13903edbd24c211f422ec5f8007696a0830b93`)

## Objective and boundaries

Prepare the next minor release from the already-integrated `develop` cut. Curate user-facing changes into the changelog and a release-history entry, then verify the local candidate. Do not include open follow-ups #9 or #55. No push, finish/merge, PR creation, tag, issue closure, or publication is authorized by this preparation task; each external action needs its own decision.

## Work units

- [x] **R50-1 — Establish the local release candidate and tracking.** Create `release/v0.5.0` from the agreed cut, confirm its base and local-only state, create issue #56, and mirror this checklist. Route: parent inline for GitHub/branch and tracking; no source implementation. Evidence: branch HEAD equals the cut; issue #56 read back as open with matching body and `type:chore`.
- [x] **R50-2 — Curate release metadata.** Draft the delta from `v0.4.1` using `git-cliff` and actual commit evidence; update `CHANGELOG.md` and `HISTORY.md` in English. Highlight user-facing additions and fixes, mention exported library symbols changed/removed where applicable, and distinguish maintenance. Do not claim a release has been published; mark the release date as pending if publication date is unknown. Route: delegated writer (two non-trivial documentation files); paths `CHANGELOG.md`, `HISTORY.md`. Check: compare notes to committed changes and verify correct version heading and history entry.
- [x] **R50-3 — Verify the local candidate.** Run `go test ./...`, `go build ./...`, `go vet ./...`, `golangci-lint run ./...`, `goreleaser check`, `make release-notes`, and `git diff --check` without publishing; inspect notes against commits and confirm branch state. Route: delegated verifier; all seven checks passed, lint reported zero issues, branch remains local at the agreed cut with only the expected metadata and tracking changes. Publication, cross-platform builds, and remote refresh were not tested.
- [ ] **R50-4 — Record work-unit commit identity (exception).** After an explicitly authorized metadata commit, record its full SHA in a short follow-up `docs(odd)` commit, closing this stage before staging that follow-up. This is the sole ODD commit-identity exception. If commit permission is not yet given, leave this stage open and do not stage.

## Release checks and delivery boundary

Forecast: about 100–250 authored documentation lines, no generated files committed. Delivery strategy: `ask-on-risk`; one metadata work unit plus the ODD identity exception. TDD mode: not applicable to release-only documentation; functional checks above are required. The release branch has `develop` as an automatic finish target and `main` as a manual PR target; no PR goes to `develop`. The published tag must be created from `main` after promotion, not during candidate preparation. `CHANGELOG.draft.md` and `RELEASE_NOTES.md` are local generated helpers, not committed artifacts.

## Progress and next step

R50-1 verified: local branch HEAD equals the cut; issue #56 and checklist mirror read back. R50-2 verified: `CHANGELOG.md` and `HISTORY.md` add 41 lines for the planned release; git-cliff draft was cross-checked against all 41 commits (including 11 it skipped), exported API checked for removed/renamed symbols (none). R50-3 verified: `go test ./...`, `go build ./...`, `go vet ./...`, `golangci-lint run ./...` (zero issues), `goreleaser check`, `make release-notes`, and `git diff --check` passed. Publication date remains pending. Next: request explicit permission for staging/committing the metadata work unit; no commit, push, PR, merge, tag, issue closure, or publication has happened.
