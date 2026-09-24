# changelog-generation — feature tracking (issue #10)

Branch: `feature/changelog-generation` (base: develop)
Issue: #10 `chore(release): add git-cliff changelog generation` (`status:approved`)
Status: in progress

## Goal

Stop maintaining the changelog purely by hand: `git-cliff` generates a draft
from the Conventional Commits the repo already enforces, and a human curates
that draft into `CHANGELOG.md` (published release body) for each release.

## Decisions recorded before implementation

- Model: **generated + curated** (maintainer decision, 2026-09-24). `CHANGELOG.md`
  stays hand-curated and human-readable; `CHANGELOG.draft.md` becomes the
  git-cliff generated input that the release curator prunes and rewords.
- Tool: `git-cliff` (Model B per `.agents/MEMORY.md`), not GoReleaser's built-in
  changelog block: the draft must exist before the release branch so it can be
  curated, and GoReleaser's `--release-notes=CHANGELOG.md` contract is unchanged.
- `CHANGELOG.draft.md` is a build artifact: generated, never committed, added to
  `.gitignore` (it was previously an untracked helper with no ignore rule).
- Draft filters (cliff.toml): skip merge commits and `docs(odd)` bookkeeping;
  group by Conventional Commit type with human headings; keep scopes as context.

## Tasks

1. [x] Create `feature/changelog-generation` from develop
2. [x] Add `cliff.toml` (conventional grouping, merge/`docs(odd)` filters)
3. [x] Replace the manual `make changelog` target with git-cliff generation
4. [x] Update `RELEASING.md` step 3 and Notes for the generated+curated flow
5. [x] Ignore `CHANGELOG.draft.md` in `.gitignore`
6. [x] Validate: `make changelog` renders a readable draft for v0.2.0..HEAD
7. [ ] Checks, work-unit commit, RDD review, finish to develop

## Evidence

- `cliff.toml` renders the unreleased draft grouped as Added / Fixed /
  Performance / Internal, one bullet per commit subject with its scope in
  parentheses (`- Add the dflow update command (update)`); merge commits,
  `docs(odd)` records and `docs:` commits are filtered. Validated with
  git-cliff 2.14.2 against `v0.2.0..HEAD`.
- `make changelog` regenerates `CHANGELOG.draft.md` and fails fast with an
  install pointer when git-cliff is absent; the draft is gitignored and
  stayed untracked throughout.
- Checks: `git diff --check` clean; `go build ./...` OK (no Go changes in the
  candidate); `go test` suite unaffected by tooling/docs-only change.
- Review declaration: this candidate is declared for RDD review after the
  work-unit commit; expectation recorded here, verdict owned by the review.
