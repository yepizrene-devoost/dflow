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
2. [x] WU2 — issue #23: code and prose residuals
   - `cmd/gitutils/git.go:218-221` (R2-1): readability of the deletion-outcome switch `default:` arm.
   - `cmd/gitutils/git.go:177-178` (R2-2): the "observe both copies before touching either" comment.
   - `RemoteBranchExists`: separate "absent" from "lookup failed"; update `Delete()` and the two
     `finish.go` call sites; offline with a local branch present must not report the remote half as
     absent and exit 0.
   - Stale Strict TDD sentence in `odd/tasks/init-refuse-reinitialize.md`; verify the
     `odd/tasks/git-output-hygiene.md` claim (grep does not find the sentence there — record the
     discrepancy).
   - Evidence: see WU2 under `## Evidence`.
3. [x] WU3 — issue #22: the issue-closure policy
   - Rule written in `.agents/workflows/dflow-workflow.md`, referenced from `AGENTS.md` (not
     restated): trigger, labels at close, closing-comment traceability, rejected/duplicate
     out-of-scope note.
   - Evidence: see WU3 under `## Evidence`.
4. [x] WU4 — checks, docs and commit identity
   - `go build ./...`, `go vet ./...`, `gofmt -l .`, `go test -count=1 ./...`, `git diff --check`.
   - `README.md` and `CHANGELOG.md` where the behavior is user-visible.
   - Housekeeping (no commit): delete the merged local branch `feature/idempotent-delete`.
   - Record work-unit commit SHAs here.
   - Evidence: see WU4 under `## Evidence`.

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

### WU2 — the residuals left by #19 (issue #23)

Written by a bounded `gentle-ai-worker` (tasks `mudzxyr9-2-j3lh`, then `mue0fc0h-4-j3lh` for the
correction below) on `cmd/gitutils/*`, `cmd/commands/delete.go`, three test files and
`odd/tasks/init-refuse-reinitialize.md`.

**The brief was wrong and the tests caught it.** The contract handed to the writer — "`git
ls-remote` fails when `origin` is unreachable **or absent**" — made a repository with no `origin`
remote at all a failure case. Three pre-existing delete subtests went red
(`TestFailurePathRendersOneErrorIconAndNoSuccessChrome/successful delete keeps its chrome`,
`TestShortValueArgumentRendersAsValue`, `TestStartCLI/delete_--yes`), and the writer correctly
stopped and asked to expand its surfaces so it could give those fixtures an `origin`.

That request was refused, and the fixtures were left untouched. A missing `origin` is a **known
absence**, decidable locally: `HasOriginRemote()` is `git remote get-url origin`, no network, and
the repository already treats that state as benign in five places (`FetchOrigin`/`Pull`,
`CheckoutBranch`, `PullBranch`, `PushBranchUpdate`, and `dflow status`'s `has_origin`). For
`dflow delete` with no `origin`, "the remote branch does not exist, skipping remote deletion" with
exit 0 is a true statement and it is the guarantee #19 shipped. #23 names the other state — "with
the network down and the local branch present" — which is `origin` configured and the lookup
failing. The fix went into the lookup: known absence returns `false, nil` before any network call,
the unknown case returns an error. All three red subtests then passed with no edit to their files,
which is the check that the diagnosis was right rather than the fixtures being wrong.

Final behavior table for `RemoteBranchExists` / `Delete`:

| `origin` | lookup | remote copy | `Delete` outcome |
| --- | --- | --- | --- |
| not configured | not attempted | known absent | local half deleted, skip message, exit 0 |
| configured | succeeds | absent | local half deleted, skip message, exit 0 |
| configured | **fails** | unknown | local half deleted, then error naming what could not be checked; with no local half, nothing is touched |
| configured | succeeds | present | both halves deleted, or the remote failure names the half that remains |

R2-1 and R2-2 are presentation-only as the review recorded them: the outcome switch now reads
`case remoteExisted:` instead of `default:`, and the "observe both copies" comment states the
reason without re-deriving it. The three outcome messages are byte-identical, so no new test was
invented for them and none is claimed.

TDD: strict TDD applied to the behavior change. RED for the known-absence pair came from the
lookup returning the failure (`a missing origin is a known absence, not a lookup failure: failed
to check remote branch 'feature/example' on origin: ...`) and the CLI exiting 1 where 0 is
required; GREEN after the early return. The discriminating pair asserts both states inside one
test (`missing origin → (false, nil)` with no request, `configured-but-unreachable → error`), so a
change that collapsed them again could not pass.

**Mutation check of that claim, and an incident while running it.** The early return was removed
and the focused run went red exactly on the two new tests, confirming they discriminate rather
than merely pass. Restoring the file the second time used `git checkout cmd/gitutils/git.go`, which
reverts to `HEAD` and discarded the unit's uncommitted work on that file; it was recovered from the
`/tmp` copy taken before the mutation, and the recovery was verified by `cmp`, by the diffstat
returning to 65 insertions / 14 deletions, by `go build ./...`, and by a full green `go test
-count=1 ./...`. Recorded because `git checkout` on a file carrying uncommitted work is the trap,
not the mutation: the restore must come from the backup, and a second mutation round on an
uncommitted file should copy the backup in before testing rather than reaching for Git.

Two small controller edits after that, both local: the `--help` sentence in `cmd/commands/delete.go`
narrowed to "a configured 'origin' cannot be reached" so it does not read as "no origin is an
error" (delegated, then reviewed), and the new comment sentence in `git.go` rewritten — "every
branch copy lives in a remote" was wrong as written, since a local copy is also a branch copy.

A third edit came out of the end-to-end check rather than from the report. The failure message
ended with git's explanation **and** `: exit status 128`, which is not what the existing delete and
merge messages do: they let git's own text end the sentence. It now does the same. The assertion
pinning that shape was added to the unknown-case test, and it was mutation-checked by restoring the
`: %s: %w` form and seeing it go red.

**A second trap in that mutation check.** The first attempt to run the new assertion used
`-run TestRemoteBranchExistsSeparatesAbsentFromFailedLookup`, and it passed with the mutation
still in place. The reason was not that the assertion was useless: it had landed in
`TestRemoteBranchExistsKeepsAMissingOriginAndAFailedOriginApart`, because two tests ended with the
same `if exists { ... }` block and the edit matched the other occurrence. Running the mutation
against the file rather than against the intended test name produced a false green and nearly a
wrong conclusion — the assertion discriminates, the run was the defect. Both incidents are recorded
because both are cheap to repeat.

**Refuted item from the issue.** #23 lists the stale Strict TDD sentence as living in both
`odd/tasks/init-refuse-reinitialize.md` and `odd/tasks/git-output-hygiene.md`. Only the first is
true: `git show 60d9e07:odd/tasks/git-output-hygiene.md | grep -ci tdd` → `0`, and the same count
against the working tree. That file was not edited. The correction in `init-refuse-reinitialize.md`
follows the shape #19 used: it states what evidence exists, what does not, and quotes the sentence
that was wrong, instead of inventing RED→GREEN retroactively.

Accepted residual, flagged for review: `CheckoutBranch` now resolves the remote **before**
`FetchOrigin`. Both orders are honest (an unreachable `origin` already failed at the fetch), but
the error a user sees when offline changes from `failed to fetch origin` to `failed to check remote
branch ... on origin`, and the writer's stated reason was to make the unknown case reachable from a
fixture. Kept, because it also avoids a pointless full fetch when the branch is not on the remote
at all.

### WU3 — the issue-closure policy (issue #22)

Written by a bounded `gentle-ai-worker` (task `mudzyhab-3-3ofy`) in an isolated worktree on
`.agents/workflows/dflow-workflow.md` and `AGENTS.md` only; the patch applied to this tree clean.

Both quoted observables were verified here rather than taken from the report: `Manual targets: %s`
renders `none` for an empty list (`cmd/commands/finish.go:131` with `formatBranchList`), and
`"manual_targets": []` is a real field of the finish plan document
(`cmd/commands/finish.go:241`). The structural reason the rule gives for not relying on `Closes
#N` is also real: `develop` is an `auto` target and the finish runs `git merge --no-ff --no-edit`
(`cmd/gitutils/finish.go:90`), so Git writes the merge message itself and no PR carries the keyword.

Two defects found in review, both corrected:

- **Two triggers that disagreed.** The pre-existing bullet said an issue closes when the work
  "lands in `develop`", while the new checklist made the trigger "no manual targets remain". On a
  `release` or `hotfix` branch those give opposite answers, because `develop` receives the merge
  while the PR toward `main` is still open. The document now names one trigger and the old bullet
  points at it.
- **The trigger was half-written.** It said the issue stays open while a manual target remains but
  not when to close it, and the check cannot be repeated later: once the branch is finished
  (especially with `--delete`) there is no branch left to dry-run. It now says to check at the
  finish, and to close when the last manual target's PR merges.

Prose-only, so strict TDD did not apply and is not claimed; the verification is the source greps
above plus `git diff --check`. The backfill of #17, #18, #19 and #20 is a forge action and belongs
to the close-out, not to this commit.

### WU4 — checks and commit identity

Checks over the whole branch, run in this tree with all three units applied:
`go build ./...`, `go vet ./...`, `gofmt -l .` clean;
`go test -count=1 ./...` green (`ok cmd/tests 23.366s`, `ok cmd/utils 0.350s`, four packages with no
test files); `git diff --check` clean.

End-to-end verification of the delete contract with a real binary built from this tree, not the
test suite alone:

| fixture | observed |
| --- | --- |
| no `origin`, local branch present | `Branch 'feature/x' deleted locally.` + `Remote branch 'feature/x' does not exist. Skipping remote deletion.`, exit 0 |
| `origin` at a missing path, local branch present | local branch deleted, then `deleted local branch 'feature/y' but failed to check remote branch 'feature/y' on origin: fatal: ... does not appear to be a git repository ...`, exit 1 |

Housekeeping: the merged local branch `feature/idempotent-delete` was already absent
(`git branch --list` is empty for it, and `git branch -a` lists only `develop`, `main` and this
branch), so the item was satisfied by the state #19 left behind and needed no deletion. Nothing was
removed by this session.

The documentation issue template was not created, per the decision recorded at the top of this
document: issue #23 makes it conditional on a real need appearing, and no issue or request in this
session asked for one. That choice is reported in the closing comment on #23 rather than being
silently dropped.

Work-unit commit identity on `feature/version-policy-residuals`:

| Unit | Commit | Subject |
| --- | --- | --- |
| WU1 | `b016811` | `feat(version): report the exact build revision` |
| WU2 | `b97b730` | `fix(gitutils): tell an absent remote branch apart from an unchecked one` |
| WU3 | `1b9ec51` | `docs(workflow): define a uniform issue-closure policy` |

This document's own commit is the follow-up `docs(odd)` commit the ODD rule allows: it carries the
identity record above and closes WU4, and it is the last tracked write before the freeze.

`size:exception` applies to this branch: the changed-line total is far above the 400-line review
budget, because WU1 alone adds 418 lines of tests. The exception is a maintainer decision and is
recorded on the issues, since `develop` is an `auto` target and no pull request carries it.
