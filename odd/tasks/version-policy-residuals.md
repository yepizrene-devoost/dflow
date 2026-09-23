# version-policy-residuals — feature tracking

Branch: `feature/version-policy-residuals` (base: `develop`)
Session: `feature/version-policy-residuals - ISSUE(21,22,23)`
Issues: #21 (`feat(version): report the exact build revision`), #22 (`chore(workflow): define a
uniform issue-closure policy`), #23 (`chore: clear the residuals and small follow-ups left by the
delete idempotence work`)
Status: authorized; three approved issues land together in one branch, one review candidate.

## Goal

Ship the exact build revision in every version entry point (#21), write the uniform issue-closure
policy and backfill the closed issues to match it (#22), and clear the residuals left by #19 (#23).

## Decisions recorded before implementation

- #22 — the closing-comment proposal is adopted as decided by the maintainer: close an issue when
  `dflow finish --dry-run` reports `Manual targets: none`; remove `status:approved` and
  `status:needs-review` at close, keep the type label; one closing comment carrying the merge
  commit (and the review lineage when one exists); the rule lives in
  `.agents/workflows/dflow-workflow.md`, referenced from `AGENTS.md`; backfill #17, #18, #19, #20.
- #23 `RemoteBranchExists` — the ambiguity is resolved in this branch: a failed `git ls-remote` is
  no longer reported as "branch absent".
- #23 documentation issue template — not created; the issue's own condition ("only if a real need
  appears") is not met. The decision is recorded in this document and in the closing comment.
- #21 presentation — `dflow <version> <abbrev-rev>` with a `-dirty` suffix when `vcs.modified`
  is true; the full 40-character revision is exposed for scripts through a flag; a binary without
  VCS stamping falls back to today's output.

## Tasks

1. [x] WU1 — issue #21: version string carries the VCS revision
   - `cmd/utils`: read `runtime/debug.ReadBuildInfo()` once; extend the version string with the
     abbreviated revision (7 chars) and `-dirty` when `vcs.modified=true`; fall back to the current
     behavior when no VCS stamping exists.
   - Expose the full 40-character revision through a flag on `dflow version`.
   - Keep every entry point consistent: `version`/`ver`, root `--version`/`-V`, `utils.PrintBanner`.
   - `Makefile`: stop describing the wrong tree — the `VERSION` fallback must not claim a released
     tag for a non-release tree.
   - Tests: version formatting, dirty suffix, no-stamping fallback, flag output.
   - Evidence: see WU1 under `## Evidence`.
2. [ ] WU2 — issue #23: code and prose residuals
   - `cmd/gitutils/git.go:218-221` (R2-1): readability of the deletion-outcome switch `default:` arm.
   - `cmd/gitutils/git.go:177-178` (R2-2): the "observe both copies before touching either" comment.
   - `RemoteBranchExists`: separate "absent" from "lookup failed"; update `Delete()` and the two
     `finish.go` call sites; offline with a local branch present must not report the remote half as
     absent and exit 0.
   - Stale Strict TDD sentence in `odd/tasks/init-refuse-reinitialize.md`; verify the
     `odd/tasks/git-output-hygiene.md` claim (grep does not find the sentence there — record the
     discrepancy).
3. [ ] WU3 — issue #22: the issue-closure policy
   - Rule written in `.agents/workflows/dflow-workflow.md`, referenced from `AGENTS.md` (not
     restated): trigger, labels at close, closing-comment traceability, rejected/duplicate
     out-of-scope note.
4. [ ] WU4 — checks, docs and commit identity
   - `go build ./...`, `go vet ./...`, `gofmt -l .`, `go test -count=1 ./...`, `git diff --check`.
   - `README.md` and `CHANGELOG.md` where the behavior is user-visible.
   - Housekeeping (no commit): delete the merged local branch `feature/idempotent-delete`.
   - Record work-unit commit SHAs here.

## Out of scope

- The `.github/ISSUE_TEMPLATE/documentation.yml` form — not created, per decision above.
- Issues closed as rejected, duplicate or invalid: their terminal handling is not defined by the
  new policy and stays a human decision.
- Build reproducibility: the goal is self-reporting, not reproducible builds.
- Release-channel labeling beyond what `main.version` already carries.

## Evidence

### WU1 — version carries the exact build revision (issue #21)

Written by a bounded `gentle-ai-worker` (task `mudzkv9f-1-a5j0`) over the surfaces
`cmd/root/version.go`, `cmd/utils/*`, `Makefile`, `cmd/tests/version_cli_test.go`, `README.md`,
`CHANGELOG.md`. The controller had already written `cmd/utils/version.go` and the `utils.go`
accessors inline before the routing correction below; the worker reviewed and kept them.

**Routing defect, recorded because it is the reason this work unit has mixed authorship.** The
first three files were written inline by the controller. The Multi-file write rule (2+
non-trivial files) is a mandatory delegation trigger and the justification was not on the table;
the human caught it ("no estás delegando"), not the harness. Everything after that point was
delegated, and the remaining two corrections were kept small and local (one file each) as the
quick-fix path allows.

Design decision taken before implementation and held through it: the commit comes **only** from
the VCS stamp, and `-X main.version` stays reserved for the channel marker. Two consequences made
that the cheapest correct shape:

- `make install` can no longer inject a revision that contradicts the stamp, so the `-dirty`
  suffix always describes the same tree the token names. The rejected alternative was to keep
  `git describe --long --dirty` in the Makefile and disable stamping (`-buildvcs=off`) to avoid
  the conflict — that would have thrown away the `-dirty` marking the issue explicitly asks for.
- A release build still gets `v0.2.0 <rev>`: goreleaser injects the tag into the marker, the
  toolchain supplies the revision, neither can lie about the other.

Behavior contract, verified by the controller independently of the worker's report, from this
dirty tree at `1ba2f58e04642c56bb5e2fc4448e3a2f72af82b4`:

```console
$ go build -o $T/dflow . && $T/dflow version
dflow dev 1ba2f58-dirty
$ $T/dflow version --revision
1ba2f58e04642c56bb5e2fc4448e3a2f72af82b4
$ $T/dflow version --json
{"version":"dev","revision":"1ba2f58e04642c56bb5e2fc4448e3a2f72af82b4","dirty":true}
$ go build -buildvcs=false -o $T/dflow-novcs . && $T/dflow-novcs version
dflow dev
$ $T/dflow-novcs version --revision
unknown
$ $T/dflow-novcs version --json
{"version":"dev","revision":"","dirty":false}
```

The unstamped pair is the fallback the issue demands: the marker alone, no empty second token.

TDD: strict TDD applied where the behavior was new. RED came from the stamp seam leaking across
cases (`sync.OnceValue` is not resettable) — 8 subtests in `./cmd/utils/` failed against the
carried-over memo, and `version --revision`/`--json` failed with `unknown flag`; both went GREEN
after the `sync.Once` + value seam and the flag work. The `version`/`ver`/`--version`/`-V`
agreement test and the `PrintBanner` test passed on first run and are stated as regression
coverage, not RED→GREEN evidence, because that consistency already existed in the carried-over
code. Order independence re-checked with `-count=3 -shuffle=on`.

Two review-driven corrections by the controller after the worker returned, both in
`cmd/root/version.go`, because the worker's own report could not be taken as its review:

- `_ = utils.EmitJSON(report)` swallowed the encode error, so a `--json` caller could get exit 0
  with no document — a false success, against the repo's "JSON in, JSON out" contract. `Run`
  became `RunE` and the error is returned, which `Execute` renders as one `{"error": ...}`
  document. `status.go` already does it this way.
- The `--revision` lookup used `err == nil && revisionOnly`, turning a lookup failure into a
  silent fallback to the human line. Both flags are now resolved in `PreRunE`, where a failure is
  returned.

The worker had listed `--revision --json` precedence as an open risk pinned only by help text; a
subtest now pins it (`--json` wins — a caller that asked for JSON never receives a bare hash).

Checks: `go build ./...`, `go vet ./...`, `gofmt -l .` clean, `go test -count=1 ./...` green
(`ok cmd/tests 20.129s`, `ok cmd/utils 0.193s`).

Residuals accepted for WU1, all small:

- The Makefile change is verified by `make -n build` and by evaluating the expression against the
  `v0.2.0` ref, not by an automated test; nothing in CI runs the Makefile.
- `resetVCSStamp()` is test-only. A non-test caller could bypass the once-per-process memo, which
  is a seam cost, not a behavior risk.
- `PrintBanner` still writes to `os.Stdout` rather than a Cobra writer (pre-existing, unchanged,
  out of this issue's scope).
- GoReleaser's build flags were not re-inspected beyond confirming it injects `main.version`,
  which the channel-only decision leaves meaningful.

### WU2 — pending

### WU3 — pending

### WU4 — pending
