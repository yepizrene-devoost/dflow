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
- [x] task-3b: A rejected run removes any previously generated `RELEASE_NOTES.md` (found by verification)
- [x] task-4: #1 assert the extracted section matches the version being released (Makefile)
- [x] task-5: #2 preflight `.env` and `GITHUB_TOKEN` before goreleaser runs (Makefile)
- [x] task-6: #5 resolve the inert `changelog:` block in `.goreleaser.yaml`
- [x] task-7: #3 CI job covering the release path without publishing
- [x] task-8: #6 pin the empty-release-body digest presentation with a test
- [x] task-9: Verify every added guard and the full Go suite
- [ ] task-10: Work-unit commits on the feature branch; record commit identity

## Evidence

- task-9 (independent verification, `gentle-ai-verify` on the integrated tree): ZERO blocking
  findings. Full `go test ./...` green including `cmd/tests 28.351s`; `gofmt -l .` clean; `go vet ./...`
  clean; production `cmd/` and `pkg/` untouched by the test work unit. The `release` preflight was
  exercised with a stubbed `goreleaser` on PATH: valid tag reached it (exit 0), while mismatched tag,
  `dev`, missing/unreadable `.env` and empty `GITHUB_TOKEN` each exited 2 with their own message and
  never reached it. A rejected run after a valid one left no `RELEASE_NOTES.md` and no `.tmp`.
  Extraction stayed byte-identical to the old bare awk on the real CHANGELOG.md, and unrelated targets
  (`build`, `install`, `changelog`, `clean`) were diffed as unchanged.
- task-9 advisory, deliberately not acted on: `RELEASING.md` states the new contract twice, in the
  procedural "The release target:" list and again in `## Notes`. That duplication predates this branch
  (the section already repeated the `RELEASE_NOTES.md` facts), and collapsing it is a doc restructure
  larger than this unit.
- task-9 advisory, not a defect: after a successful `release-notes` a failing `release` preflight
  still leaves a valid `RELEASE_NOTES.md` on disk. It is fresh and matches the current CHANGELOG.md,
  so publishing it by hand would be correct; the stale-body problem is specifically a body that no
  guard ever validated.
- task-9 note on exit codes: failing recipes exit 2, because make reports the recipe's exit 1 as its
  own error exit 2. The two verification agents disagreed on which number to quote; both were
  describing the same non-zero failure, and no consumer depends on the exact value.

- task-6 (#5 dead config): premise CONFIRMED before deleting, not assumed. The installed goreleaser
  2.18.2 documents the semantics in its own `release --help`: `--release-notes  Load custom release
  notes from a markdown file (will skip GoReleaser changelog generation)`. `make release` is the only
  release path and always passes the flag; no config references `{{ .Changelog }}`; `archives.files`
  attaches the repository's own `CHANGELOG.md` (a plain file entry, not a generated changelog); and
  `dist/` from the real v0.3.0 run carries no `Changelog` artifact. The block arrived as
  `goreleaser init` boilerplate (`2b9581a`) and became inert the moment `--release-notes` was added
  (`d9c7582`). Deleted, with a comment stating where the release body really comes from.
  Upstream web docs were NOT consulted (no web tools in that session); the evidence is the installed
  binary, its JSON schema, and this repo's own artifacts.
- task-6 finding, deliberately NOT fixed: `goreleaser check` exits 2 on three pre-existing
  deprecations (`snapshot.name_template`, `archives.format`, `archives.format_overrides.format`).
  Identical output against the pre-change bytes, so this branch neither introduced nor hid it. It
  matters for task-7: a CI gate running `goreleaser check` is born red on a clean checkout, so the
  job must either resolve the deprecations or keep that step informational with the exit code and
  its meaning spelled out.
- task-8 (#6 empty-body digest): test-only, production code untouched. `notes_test.go` gains a
  contrast test making the load-bearing nil-vs-empty-non-nil distinction legible (the existing table
  uses `slices.Equal`, which cannot tell a nil slice from an empty one) plus a new
  title-wrapped-in-blank-runs case; `update_cli_test.go` gains the first CLI-level pin of the
  empty-body path, asserting the report stays four well-formed lines with no "what's new" heading and
  no dangling separator. Observed output for an empty published body, pinned as-is:
  current version / latest release / "an update is available" / "release notes: <url>".
  Checks: `go test ./cmd/selfupdate/...` ok, focused CLI test ok, full `./cmd/tests/...` ok 27.9s,
  `gofmt -l` clean on both files, `go vet ./cmd/...` clean.
- task-3b/4/5 (`release` target preflight, found by verification + audit findings #1 and #2): all
  guards run in one `set -eu` line before goreleaser is reached — `.env` exists, `.env` is readable,
  sourcing failure propagates instead of being swallowed by the old `;`-chain, `GITHUB_TOKEN` is
  non-empty, HEAD is on an exact tag (refuses `VERSION=dev`), and the version token in the first line
  of `RELEASE_NOTES.md` equals the tag being published, naming BOTH versions when they differ. A
  rejected run now removes the temp file AND the destination, so nothing publishable is left on disk
  from any run. The `.env` message points at `.env.example`, which is committed and already referenced
  by `RELEASING.md`.
- task-1/2/3: `Makefile` `release-notes` now runs `set -eu`, extracts to `RELEASE_NOTES.md.tmp`, and
  passes three guards before `mv` — awk exit status, `grep -q '^## 📦'` on the extracted body, and
  `awk 'NR > 1 && NF'` for content below the heading; each guard removes the temp file and exits
  non-zero with its own message. Extraction semantics unchanged (verified byte-identical with `cmp`
  against the old bare-awk output on the real CHANGELOG.md). Verified by direct run on the working
  tree: heading-only → exit 2 with the "no content below its heading" message, no marker → exit 2
  with the "carries no '## 📦' version section" message, missing file → exit 2 with the extraction
  message plus awk's diagnostic; in all three, no `RELEASE_NOTES.md` and no `.tmp` are left behind.
  `RELEASING.md` documents the same contract.
- A rejected run must leave no release body on disk at all. The first implementation removed only
  `RELEASE_NOTES.md.tmp`, so a rejected run left the previous run's `RELEASE_NOTES.md` in place —
  unreachable through `make release` (which aborts first), but `RELEASE_NOTES.md` is exactly what
  `goreleaser release --release-notes=RELEASE_NOTES.md` consumes, so a standalone goreleaser
  invocation would have published a stale artifact that nothing re-validated. This contradicted the
  Design section above, which had the contract right and the implementation wrong. Fixed by having
  `fail()` remove both the temp file and the destination.
- Advisory history: the first verification agent reported two blocking findings (missing marker
  guard, make-vs-shell `$` escaping of the `fail()` message) that did not exist in the tree it
  claimed to read; refuted by direct re-run above — its own quoted output printed the message only
  the "missing" guard produces. Its third point (stale artifact) is the real one recorded above.
- Nit adopted from the same verification: the marker guard reads `head -n 1 "$$tmp" | grep -q`
  instead of grepping the whole file. Equivalent while the extractor always starts at the marker,
  but the guard no longer depends on that detail staying true.

## Work-unit identity

| Unit | Commit | Subject |
| --- | --- | --- |
| WU1 | `47a644f` | `chore(release): fail fast when the release-notes extraction finds no version section` |
| WU2 | `9310b70` | `chore(release): preflight the tag, token and changelog version` |
| WU3 | `ec50e5f` | `chore(release): drop the inert changelog block and modernize deprecated keys` |
| WU4 | `96983af` | `chore(ci): check the release configuration and changelog contract without publishing` |
| WU5 | `e80b49f` | `test(selfupdate): pin the empty release-body digest degradation` |

task-10 closes with this record. Every stage above is closed in a commit that contains its work;
this document is the follow-up `docs(odd)` commit the commit-identity exception allows. Nothing in
this unit is left open, and no CHANGELOG.md entry was added: entries are curated per release from
`make changelog`, not per branch.
