# release-v0.4.1

Prepare and publish `v0.4.1`: everything merged into `develop` since the
`v0.4.0` tag is unreleased.

## Issues

Tracks #41 (`[Chore]: prepare v0.4.1`)

Branch: `release/v0.4.1` (base: `develop`)

## Goal

Turn the post-`v0.4.0` maintenance changes into a published release: a curated
`CHANGELOG.md` section for `v0.4.1`, its `HISTORY.md` entry, a verified release
candidate, promotion to `main`, the tag, and the published GitHub release.

## Version decision

- **`v0.4.1`** is a patch release: the changes are backward-compatible fixes
  and contract clarifications, with no breaking change or substantial new
  feature requiring a minor bump.
- Issue #9 (stacked-branch rebasing) remains open and is out of scope.

## Tasks

- [x] **WU1** — Generate and curate the `v0.4.1` section in `CHANGELOG.md` from
      the commits since `v0.4.0`; remove internal release bookkeeping and keep
      the notes user-facing.
      Surfaces: `CHANGELOG.md`.

      Evidence: the new section is `## 📦 v0.4.1 – Release Contracts & Version
      Display` with `Changed` and `Fixed` entries covering the version-marker
      correction and the non-interactive finish/deletion safety contract. The
      section is user-facing and `git diff --check` passes.

- [x] **WU2** — Add the `v0.4.1` entry to `HISTORY.md` and verify the release
      metadata against the selected tag version.
      Surfaces: `HISTORY.md`.

      Evidence: the entry is above `v0.4.0`, links to the `v0.4.1` tag, carries
      the current date, and summarizes the three release highlights. Heading,
      tag, and date checks pass.

- [x] **WU3** — Run the repository test and release-path validation checks on
      the release candidate.
      Surfaces: validation evidence only.

      Evidence: `git diff --check`, `go test ./...`, `make release-notes`,
      `goreleaser check`, and the CI-equivalent changelog contract check all
      passed. The HISTORY heading, tag, and date checks passed after correcting
      an initial verifier pattern. Generated ignored release artifacts were
      removed; `make release` was not run.

- [ ] **WU4** — Finish the release branch, promote it to `main`, tag `v0.4.1`,
      and publish the GitHub release after the required standalone confirmations.
      Surfaces: release workflow and GitHub release.

## Out of scope

- Implementing issue #9 or other new features.
- Changing application behavior beyond release metadata and documentation.
- Bundling branch finish, merge, tag, push, or publish confirmations together.

## Commit identity

- WU1 + WU2 + WU3: `a2a84ee` (`chore(release): prepare the v0.4.1 notes
  and history entry`), on `release/v0.4.1`.
