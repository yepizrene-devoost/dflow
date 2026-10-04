# Prepare v0.5.1 release

Issue: #61 — `[Chore]: prepare v0.5.1` (`type:chore`, `status:approved`)
Branch: `release/v0.5.1`. Cut from `develop` at `a4e0303c2bb6b7228000741790168c1eddc5ea7e`, then
fast-forwarded to `256e39f` so the release carries the interactive-test-harness fix that restored the
`develop` CI (its own work unit: `odd/tasks/pty-harness-loses-final-output.md`).
Last published release: `v0.5.0` (`b9d930fa16b753a8e35a5ad342e3acd1d2c6cadb`)

## Objective and boundaries

Prepare the next patch release from the `develop` cut that already carries the `dflow init`
fix for issue #60. Curate the user-facing changes into the changelog and the release history.
Do not include the open follow-ups #9 and #55, and do not address the three advisory findings
the review of the init fix recorded — they are separate later work.

This is a `release/*` branch: `develop` is its automatic finish target and `main` its manual
PR target, so no PR goes to `develop` and `main` is reached only through a pull request. The
tag is created from `main` after promotion, never during candidate preparation.

The release date is recorded at preparation, because publication happens on the maintainer's same
local day. That is the v0.4.0 and v0.4.1 convention, and it keeps the candidate to one commit. The
v0.5.0 `Status: Planned release` / `Date: Pending publication` placeholder is the form this release
deliberately avoids: it required a second work unit (`8f324da`, `docs(release): date v0.5.0
history`) that also had to rewrite the document preamble, plus a second review candidate for what
was a one-line change. If publication slips past the recorded local day (2026-10-03), reconcile the
date then, as its own authorized change.

A further commit, push, PR, merge, tag, issue closure, or publication still requires its own
authorization. `CHANGELOG.draft.md` and `RELEASE_NOTES.md` are local generated helpers, and
neither is committed.

## Work units

- [x] **R51-1 — Cut the release candidate and generate the draft.** Create `release/v0.5.1`
  from the integrated `develop` cut and generate the draft from the Conventional Commits since
  `v0.5.0` with `git-cliff`. Route: parent inline for the branch and the generated draft, which
  is gitignored. Evidence: the branch sat at the cut `a4e0303` and existed only locally, which
  `git reflog` records as created from HEAD; the specific command that created it is not
  independently provable from the repository, so it is not claimed here. `make changelog` produced a
  draft reporting one `Fixed` entry and one `Internal` entry, and reported five commits skipped at
  parse time — the merge commits; the `docs(odd)` records are dropped separately, by a parser rule
  rather than by a parse error.
- [x] **R51-2 — Curate the release metadata.** Curate the draft into a versioned `CHANGELOG.md`
  section and a `HISTORY.md` entry, in the voice the previous releases use, distinguishing the
  behavior change from the fixes and separating maintenance. No claim of publication, and the
  release date is the maintainer's local day, recorded here rather than pending. Route: parent
  inline for three documentation paths; `CHANGELOG.md`, `HISTORY.md`,
  `odd/tasks/release-v0.5.1.md`.
- [x] **R51-3 — Verify the local candidate.** Run the repository's release-path checks without
  publishing and inspect the notes against the actual commits. Route: delegated verifier. Outcome:
  verified with findings, all non-blocking. `go build ./...`, `go vet ./...`, `gofmt -l .` and
  `git diff --check` clean; `go test -count=1 ./...` green across the ten packages; the seven
  release-path checks pass, `make release-notes` included; the two quoted user-facing sentences are
  byte-identical to the production constants in `cmd/commands/init.go`; no exported `pkg/*` symbol
  was removed or renamed anywhere in `v0.5.0..develop`, so no `Changed` entry is owed, and the
  absence is correct rather than an omission; the Makefile extraction yields `v0.5.1` with content
  below the heading; and the candidate was exactly the three declared paths. Its findings were two
  wording issues and one unprovable attribution, all corrected here.
- [ ] **R51-4 — Record work-unit commit identity (exception).** The metadata work unit's commit
  id is recorded by the short follow-up `docs(odd)` commit, the one exception the repository
  allows, because a commit cannot contain its own id.

No separate date work unit exists: the date is recorded by R51-2, so this release has no
`Set the release-history date` stage and no date-correction identity stage.

## Release checks and delivery boundary

Checks required before promotion: `go test ./...`, `go build ./...`, `go vet ./...`,
`golangci-lint run ./...`, `goreleaser check`, `make release-notes`, and `git diff --check`.
TDD is not applicable to release-only documentation; the functional checks above are the
evidence for this candidate.

Delivery sequence, each step separately authorized: `dflow finish` on this branch (merges into
`develop` and publishes the branch), then a pull request toward `main` — which requires
`status:approved` on the linked issue and a decision between a closing and a non-closing
reference — then the tag, then `make release`, which requires a readable `.env` with a non-empty
`GITHUB_TOKEN` and a newest `## 📦` section whose version matches the tag.

## Progress and next step

R51-1, R51-2 and R51-3 are complete. R51-4 is the last open stage and is the documented exception:
it is closed by the short `docs(odd)` commit that records this work unit's commit id, because a
commit cannot contain its own. No push, PR, merge, tag, or publication has happened.

Review declaration: assess and review the complete committed range from the release cut once
this candidate's commits and their identity record land. No verdict and no approval is presumed
here.
