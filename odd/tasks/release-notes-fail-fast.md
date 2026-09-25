# Feature: release-notes-fail-fast (issue #32)

`make release-notes` (and through it `make release`) extracts the latest `## 📦` section of
`CHANGELOG.md` with a bare awk invocation and redirects it into `RELEASE_NOTES.md`. The recipe
swallows every failure: no version section, an empty section, an unreadable `CHANGELOG.md`, or a
failing awk all leave an empty or partial `RELEASE_NOTES.md` on disk and exit 0, so `make release`
goes on to publish a GitHub release with an empty body. Silent failure in the publishing path.

## Design

Extract into a temporary file, validate it, and only then move it into place — the generated
release body never lands on disk in an unpublishable state, and no stale/partial `RELEASE_NOTES.md`
survives a rejected run. Three pointed guards, each its own message:

1. **awk exit status** — a failing extraction (missing or unreadable `CHANGELOG.md`) propagates
   instead of being swallowed.
2. **no version section** — the extracted body does not start with the `## 📦` marker:
   `CHANGELOG.md carries no '## 📦' version section`.
3. **empty version section** — the marker is there but nothing of substance below it, which would
   publish a heading-only release body.

Every guard removes the temporary file before exiting non-zero. `make release` depends on
`release-notes`, so a failed guard stops the release before goreleaser is reached; that makes the
issue's optional "assert non-empty before release" step structural rather than a second copy of the
same check.

Portability: BSD awk (macOS host) and mawk/GNU awk (Linux CI) both support the POSIX constructs
used; in the recipe every shell `$` must be doubled so make does not expand it.

## Out of scope

- Any change to the extraction semantics themselves (first `## 📦` section up to the next one).
- Moving the recipe into a `scripts/` file; the repository already keeps inline shell guards in the
  Makefile (`changelog`) and there is no shell test harness to justify a new one.
- `make changelog` / git-cliff behaviour.

## Scope added by the release-path audit (maintainer decision, issue #32 branch)

A read-only audit of the publishing path for the same class of silent failure (unobserved exit
status, empty/partial output accepted) found more gaps. The maintainer chose to close them on this
branch as separate work units instead of opening issues, so this branch is one reviewable release-
path hardening unit.

- **#1 (HIGH) tag ↔ section mismatch** — extraction takes the latest `## 📦` section
  unconditionally. Tagging `v0.5.0` while the newest CHANGELOG section is `v0.4.0` publishes
  v0.4.0's notes under the v0.5.0 tag, exit 0, non-empty body: the strongest remaining
  silent-mispublish. Fix: assert the extracted heading carries the version being released.
- **#2 (MEDIUM) `.env` / `GITHUB_TOKEN`** — `Makefile` sources `.env` in a `;`-chained recipe line
  without `set -e`; a missing file prints an error and the line continues, so goreleaser builds all
  artifacts and fails late at the publish step with a message far from the cause. Fix: preflight the
  file and the token, fail immediately.
- **#5 (LOW) dead config** — the `changelog:` block in `.goreleaser.yaml` is inert once
  `--release-notes` is supplied, and reads as if releases carried a generated changelog. Fix:
  document or remove it.
- **#3 (MEDIUM) no CI coverage of the release path** — `.github/workflows/go.yml` runs build and
  tests only; every failure mode in this family is first observed on a maintainer machine at release
  time. Fix: a CI job that checks the goreleaser config and the CHANGELOG contract without
  publishing.
- **#6 (LOW) empty-body digest degradation** — `SummarizeReleaseNotes` returns nothing for an empty
  release body and `dflow update --check` then omits the "What's new" section silently. The publish
  guards above remove the input that produces it; pin the presentation with a test rather than
  change the CLI contract.

Audited and found clean: both installer scripts already fail fast on download, checksum and
extraction steps (no swallow-and-continue); their gap is coverage (#3), not behaviour.

## Tasks

- [x] task-1: Guarded `release-notes` recipe in the Makefile
- [x] task-2: Sync `RELEASING.md` with the fail-fast contract
- [x] task-3: Verify the guard matrix (ok / heading-only / no-marker / missing file) and leftover-free temp handling
- [ ] task-4: #1 assert the extracted section matches the version being released (Makefile)
- [ ] task-5: #2 preflight `.env` and `GITHUB_TOKEN` before goreleaser runs (Makefile)
- [ ] task-6: #5 resolve the inert `changelog:` block in `.goreleaser.yaml`
- [ ] task-7: #3 CI job covering the release path without publishing
- [ ] task-8: #6 pin the empty-release-body digest presentation with a test
- [ ] task-9: Verify every added guard and the full Go suite
- [ ] task-10: Work-unit commits on the feature branch; record commit identity

## Evidence

- task-1/2/3: `Makefile` `release-notes` now runs `set -eu`, extracts to `RELEASE_NOTES.md.tmp`, and
  passes three guards before `mv` — awk exit status, `grep -q '^## 📦'` on the extracted body, and
  `awk 'NR > 1 && NF'` for content below the heading; each guard removes the temp file and exits
  non-zero with its own message. Extraction semantics unchanged (verified byte-identical with `cmp`
  against the old bare-awk output on the real CHANGELOG.md). Verified by direct run on the working
  tree: heading-only → exit 2 with the "no content below its heading" message, no marker → exit 2
  with the "carries no '## 📦' version section" message, missing file → exit 2 with the extraction
  message plus awk's diagnostic; in all three, no `RELEASE_NOTES.md` and no `.tmp` are left behind.
  `RELEASING.md` documents the same contract.
- A rejected run leaves any previously generated `RELEASE_NOTES.md` in place (only the temp file is
  removed). Accepted: `release` depends on `release-notes`, so goreleaser is never reached with a
  stale artifact; the file is gitignored and regenerated on every successful run.
- Advisory history: the first verification agent reported two blocking findings (missing marker
  guard, make-vs-shell `$` escaping of the `fail()` message) that did not exist in the tree it
  claimed to read; refuted by direct re-run above. Its third point (stale artifact) is the accepted
  note above.
