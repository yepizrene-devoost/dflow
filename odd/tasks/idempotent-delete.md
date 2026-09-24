# idempotent-delete — feature tracking

Branch: `feature/idempotent-delete` (base: develop)
Issue: #19 — `fix(gitutils): make delete work when only one of the two copies exists`
Status: authorized; scope is the delete idempotence defect, its regression coverage, the
corrections the native review asked for, and one bounded trial of the pre-freeze freeze order.

## Goal

Make `dflow delete <branch>` idempotent: each of the two copies (local, remote) is deleted
when it exists, and a missing copy is not an error. The command fails only when neither copy
exists, or when the deletion of a copy that does exist failed.

## Why this now

`Delete` in `cmd/gitutils/git.go` ran `git branch -D` first and returned on failure, so the
remote half was never reached when the local branch was already gone. That state is easy to
reach: a half-finished delete, a network failure in the middle, or removing the local copy by
hand first. The reported symptom was a local failure, exit 1, and a remote branch left alive.

A second, process-level driver was added during the work: recording a review outcome in a
`docs(odd)` commit *after* approval mints a fresh unreviewed candidate every time, because
Receipt-driven development derives candidate identity from the tree. The first two reviews of
this branch demonstrated the cycle and its cost. WU5 trials the freeze order that breaks it.

## Tasks

1. [x] WU1 — make `gitutils.Delete` treat the two halves independently
   - Observe both copies before touching either, then delete only the copies that
     exist, so the four cases in the issue table behave predictably:
     both → delete both; local only → delete local and say the remote was absent;
     remote only → delete remote and say the local was absent; neither → non-zero
     with nothing to do.
   - Keep a failure of the remote step after a successful local step reported, naming
     what remains, instead of being silently skipped.
   - Keep the existing current-branch refusal and its placement before the spinner.
   - Evidence: see the WU1 OUTCOME under `## Evidence`.
2. [x] WU2 — regression coverage for the four cases
   - One table-driven case set at the gitutils boundary covering all four rows.
   - The remote-failure report (local deleted, remote still there) named explicitly.
   - At least one end-to-end real-binary assertion for the `missing|exists` row that
     the issue reproduces, since that is the path the bug report walked.
   - Evidence: see the WU2 OUTCOME under `## Evidence`.
3. [x] WU3 — documentation, checks and work-unit commit
   - `README.md` states the idempotent contract and the "nothing to delete" failure.
   - `go build ./...`, `go vet ./...`, `gofmt -l .`, `go test -count=1 ./...`,
     `git diff --check`.
   - Evidence: see the WU3 OUTCOME under `## Evidence`.
4. [x] WU4 — apply the advisory findings the two reviews recorded
   - `cmd/commands/delete.go`: remove the stray blank line inside `Long` that split the
     first sentence into two paragraphs and shipped a malformed `dflow delete --help`.
   - `cmd/gitutils/git.go`: rename the pre-delete observations to `localExisted` /
     `remoteExisted` and rewrite the remote-failure comment, so the branch that the two
     reviews flagged twice reads truthfully instead of implying a live existence check.
   - `cmd/tests/idempotent_delete_test.go`: state why the fixture deletes the local
     pointer by hand.
   - Correct a false claim in this document about Strict TDD.
   - Evidence: see the WU4 OUTCOME under `## Evidence`.
5. [x] WU5 — declare the review stage inside the frozen candidate
   - Tick the review stage as the record of the *freeze act* and of the declared
     expectation, never as the verdict, and never with the lineage or target ids: those
     are derived from the very commit that would carry them, so no ordering can put them
     in the tree they describe.
   - Collapse the pre-freeze commits with `amend` while nothing is frozen, so the work
     unit, the stage closure and the freeze declaration reach the reviewer as one
     candidate instead of growing a new one after each bookkeeping write.
   - Declare the trial's success criterion before freezing.
   - Evidence: see the WU5 OUTCOME under `## Evidence`.

## Out of scope

Forge integration, and distinguishing a network failure inside the pre-delete remote lookup
from a genuinely absent remote branch. `RemoteBranchExists` swallows that difference today and
this work keeps that boundary; only the deletion outcomes change.

Two known adjacent defects are deliberately left alone, because touching them would widen this
candidate with unrelated paths: the same false Strict TDD sentence this document carried also
sits in `odd/tasks/git-output-hygiene.md` and `odd/tasks/init-refuse-reinitialize.md`, and the
follow-up readability pass now belongs to WU4 itself.

## Evidence

### WU1 — make `gitutils.Delete` treat the two halves independently

OUTCOME: done. `Delete` reads both copies once, before touching either (`BranchExists` for
`refs/heads/<branch>`, `RemoteBranchExists` for `origin`), returns
`branch '<branch>' does not exist locally or on origin; nothing to delete` when neither is
present, and then runs each half only when its copy exists. The current-branch refusal and
its placement before the spinner are unchanged. The remote-failure path branches on whether
the local half ran: with a local deletion already done it returns
`deleted local branch '<branch>' but failed to delete remote branch '<branch>': <git
diagnostics>`, and without one the original
`failed to delete remote branch '<branch>': <git diagnostics>`. Both paths `TrimSpace` the
captured stderr, consistently with `runCapturingGit`, so the rendered line does not carry a
stray blank line before Git's own multi-line diagnostics.

Decision worth recording: existence is decided *before* the local deletion rather than
re-checking after it. Deciding it up front is what lets the local half stay untouched in the
`missing|exists` row, which is exactly the row the issue reproduces; re-checking afterwards
would recreate the ordering coupling this work removes.

Files changed: `cmd/gitutils/git.go`.

### WU2 — regression coverage for the four cases

OUTCOME: done. New `cmd/tests/idempotent_delete_test.go`, three tests reusing the existing
`initTempGitRepo`, `initBareGitRepo`, `runGit`, `withWorkingDir`, `branchExists`,
`remoteBranchExists`, `buildDflowCLI`, `setUpCLIEnv` and `startCLIRawOutput` helpers:

- `TestDeleteIsIdempotentAcrossBothCopies` — table-driven, the four rows of the issue's
  contract at the gitutils boundary, asserting the returned error and the state of both
  copies after each call. The `only the remote copy exists` row rebuilds the reported
  half-finished state by pushing the branch and then removing the local pointer by hand.
- `TestDeleteNamesTheCopyThatWasAlreadyGone` — real-binary CLI coverage of the reported
  message shape: `local already gone, remote deleted` (the issue's own reproduction, exit 0
  and the remote gone), `remote already gone, local deleted` (exit 0, says the remote was
  absent) and `neither copy exists` (non-zero, `nothing to delete`).
- `TestDeleteReportsWhatRemainsWhenTheRemoteStepFails` — the remote is a bare repo with
  `receive.denyDeletes=true`, so the remote failure is deterministic, local and
  network-free; asserts non-zero, that the message says the local half was deleted and names
  the remote branch that remains, that the local branch is gone and that the remote branch
  survives.

One correction was needed during the first run: the remote-only row pushed
`feature/example` without ever creating it locally, so `git push` legitimately failed with
`src refspec ... does not match any`. The fixture now always creates the local branch first
when a remote copy is wanted, and removes it again for that row. The production code was
never the cause and was not changed for it.

Files changed: `cmd/tests/idempotent_delete_test.go`.

### WU3 — documentation, checks and work-unit commit

OUTCOME: done. `README.md` lists the idempotent contract and the single failing case;
`CHANGELOG.md` gains a `### Fixed` entry under `## Unreleased`; the `dflow delete` entry in
`.agents/workflows/dflow-workflow.md` notes the idempotence; and `cmd/commands/delete.go`'s
command doc comment and `Long` description no longer claim the remote step is skipped "when
the branch does not exist on origin" — the old wording described the ordering defect as if it
were the contract.

Independent verification with a real binary built from this working tree against a throwaway
repo plus bare remote, all five cases:

| Case | Observed | Exit |
| --- | --- | --- |
| both exist | `Branch 'feature/both' deleted locally and remotely.` | 0 |
| local only | `deleted locally.` + `Remote branch ... does not exist. Skipping remote deletion.` | 0 |
| remote only (issue repro) | `deleted remotely.` + `Local branch ... does not exist. Skipping local deletion.`, remote gone | 0 |
| neither | `branch 'feature/ghost' does not exist locally or on origin; nothing to delete` | 1 |
| remote refuses deletion | `deleted local branch 'feature/denied' but failed to delete remote branch 'feature/denied': ... (deletion prohibited)`; local gone, remote alive | 1 |

The first and third rows are the behaviour change: before this work the third row printed
`failed to delete local branch ...: branch ... not found`, exited 1 and left the remote alive.

Checks: `go build ./...` ok, `go vet ./...` ok, `gofmt -l .` empty, `git diff --check`
clean, `go test -count=1 ./...` green. Existing contract suites re-run green inside that run,
including `TestFailurePathRendersOneErrorIconAndNoSuccessChrome` (`failing delete`, whose
fixture is the neither-exists row) and `successful delete keeps its chrome`.

TDD evidence: none produced, stated precisely. `~/.gentle-ai/state.json` records
`strict_tdd: true`, which `extensions/sdd-init.ts:703` derives from a detected test command
and forwards to SDD phase agents; it was not applied to this feature's inline ODD work, so the
tests were written alongside the implementation as regression coverage instead of RED→GREEN.
The first revision of this document claimed "Strict TDD was not active: no TDD runner is
configured for this repository", which was wrong on both counts — strict TDD is enabled, and
`go test` is a runner. Corrected here.

Boundary, documented and non-blocking: `RemoteBranchExists` returns false both for an absent
remote branch and for a failed `git ls-remote` (offline, no `origin`). The new code inherits
that: with the network down and the local branch present, the remote half is reported absent
and the command exits 0. Distinguishing those two states is a change to the lookup's contract,
not to the deletion order this issue is about, and it is listed in `## Out of scope`.

Files changed: `README.md`, `CHANGELOG.md`, `.agents/workflows/dflow-workflow.md`,
`cmd/commands/delete.go`.

### WU4 — apply the advisory findings the two reviews recorded

OUTCOME: done, all four corrections applied, none of them behavioural.

`cmd/commands/delete.go` — the stray blank line inside the `Long` string was **introduced by
this branch's own work-unit commit**, not inherited. `git blame -L 30,40` attributes the blank
line to `7b923b5` while the two sentence halves belong to `02979700` (2026-04-02), and the
help output built from that tree rendered the sentence as two paragraphs:

```
Delete a branch created with dflow from your local repository and, if it

exists, from the 'origin' remote as well.
```

Removed; the newly built binary renders `... and, if it` / `exists, from the 'origin' remote
as well.` as one paragraph. This was a user-visible defect in `dflow delete --help`, and the
review caught something the branch had shipped.

`cmd/gitutils/git.go` — the point both reviews flagged (R2-2 in lineage 1; R2-1 and R2-2 in
lineage 2, three advisory hits on one spot) was the remote-failure branch:

```go
// The local half may already be gone at this point, so the report says
// what is left ...
if localExists {
```

`localExists` is the observation taken *before* either half is touched, so inside that branch
a true value means this same call has already deleted the local branch — a fact, not a live
check, which the name and the comment both obscured. Both flags are now `localExisted` /
`remoteExisted`, keeping their symmetry, and the comment states what the value actually means
at that point. No condition, message or flow changed.

`cmd/tests/idempotent_delete_test.go` — the fixture line that removes the local pointer by
hand now says why: it rebuilds the half-finished state the issue reports.

Independent behaviour-preservation check: the same five-case end-to-end run against a binary
built from this tree after the rename produced byte-identical output, exits and branch state
to the pre-rename run recorded in WU3 — 0/0/0/1/1, the remote-failure message still naming the
local deletion and the surviving remote, and the refused remote still alive with the local gone.

Checks after the corrections: `gofmt -l .` empty, `go vet ./...` ok, `git diff --check` clean,
`go test -count=1 ./...` green (`ok .../cmd/tests 19.651s`), and no residue of the old
identifiers (`grep localExists\|remoteExists cmd/gitutils/git.go` empty).

Honest limitation: the provider's finding prose was not recoverable. Both `lineage_id` values
were searched under `.git/gentle-ai/` and `~/.gentle-ai/` and are not exposed there, so the
corrections above are judged from the flagged locations and the code, not from the reviewer's
own wording.

Files changed: `cmd/commands/delete.go`, `cmd/gitutils/git.go`,
`cmd/tests/idempotent_delete_test.go`.

### WU5 — declare the review stage inside the frozen candidate

OUTCOME: done, trial declared. What was established first, because the trial rests on it:

- Candidate identity is tree-derived. `review.inspect` composes it from `base_tree`,
  `candidate_tree` and `paths_digest`, so a write to any tracked path mints a new target. This
  is not avoidable by reordering: the ids of a candidate cannot be written into that candidate,
  because writing them changes the tree that produces them.
- The provider classifies the whole candidate, not the delta. Lineage 2 was tier `high` with
  four lenses on a delta over the approved candidate that was documentation only, since the
  risk reason (`process_boundary` in `cmd/gitutils/git.go`) belongs to the candidate.
- A `git commit --amend` cannot preserve an identity either. Demonstrated in a throwaway repo:
  an `--amend --no-edit` with no staged change already produced a different hash through the
  committer timestamp alone (`34b27bb` → `512bc1b`), and staging content changed the tree
  (`3be22be` → `6cf76fa`) and therefore the hash necessarily (`e05c0d9`). The same experiment
  showed what does preserve identity: `git notes add` left both the commit and its tree
  byte-identical. So `amend` belongs to the pre-freeze window, and only there.

Therefore the stage is recorded as the freeze act, in the past tense it already satisfies:

> Frozen with the expectation of approval declared. The candidate identity, lineage and
> verdict live in the native receipt and in Engram (`odd/idempotent-delete/review`); this file
> does not duplicate them, because doing so would mint the candidate it is describing.
> Divergence — a correction or a refusal — reopens this stage and is corrected in a later
> commit, which is by definition a new candidate.

That tick can never become false, whatever the verdict: it claims only that the candidate was
frozen with a declared expectation, which is exactly what happened. It is deliberately not
phrased as an approval, and it grants no delivery authority — the receipt is the authority and
delivery stays under ordinary repository policy.

The pre-freeze commits were then collapsed with `amend` while nothing was frozen, so the work
unit, the closed stages and this declaration reach the reviewer as one candidate. A safety ref
was created first and is removed once the review closes.

Success criterion, declared before freezing: at most 4 reviewer model runs (one review, four
lenses) against the 12 the two-review cycle cost, zero commits after the freeze, and a
`review.inspect` after approval proving the target identity did not move. The measured result
is deliberately **not** in this commit: the trial's own outcome can only be known after the
freeze, so writing it here would produce the very extra candidate the trial exists to avoid.
It is recorded in Engram instead.

Files changed: this document, plus the collapse itself.

### Commit identity

WU1–WU5 are one work unit on this branch: `62ba349` — `fix(gitutils): make delete idempotent
across both copies`, seven paths, 551 insertions and 23 deletions. It was collapsed pre-freeze
from three earlier commits (`7b923b5` the original work unit, `7075d4c` its identity record and
`fff1624` the first review's outcome record) with `git reset --soft` plus a single commit, while
nothing was frozen; the replaced tip stayed reachable from
`backup/idempotent-delete-precollapse` until that safety ref is removed once the review closes.
This stage is closed by the short follow-up `docs(odd)` commit that carries this line, which is
the one exception the repository allows for recording a commit identity. Both commits are
pre-freeze and therefore inside the reviewed candidate.

## Native review

Two reviews were run and closed before WU5, both covering the code that still ships:

- `review-505f3680fd2e6994` over `7075d4c`, base `60d9e074d1f25acb4bd8d7dd7ad9869147242c9e`:
  tier `high`, 7 paths, 398 changed lines, 4 of 4 lenses, correction budget 199, **approved**,
  authority burned as `gentle-ai.review-acknowledged/v1` (consumed revision `sha256:07941b45…`).
- `review-2ef22dfc62e910b9` over `fff1624`, same base: tier `high`, 7 paths, 432 changed lines,
  4 of 4 lenses, correction budget 200, **approved**, authority burned (consumed revision
  `sha256:7876f338…`).

Why there were two: the first review's outcome was recorded in a `docs(odd)` commit after
approval, which changed `initial_review_tree` from `9a6e5c2f…` to `f8f02836…` and minted a new
target (`5223c125…` → `479a4f7a…`) that Receipt-driven development correctly demanded be
reviewed. The second review therefore covered a delta consisting only of the first review's own
record. WU5 exists to stop that cycle.

Operational note from lineage 1: the grouped capture failed in the host relay
(`review-readability` returned malformed JSON), nothing was submitted and nothing was mutated;
following the returned instruction, STATUS was re-queried and the four slots were captured one
at a time in provider order, all admitted as `completed`. No slot was ever replayed from
transcript inference. Lineage 2 also absorbed one `cancelled` native operation on its last
slot; a fresh STATUS reoffered the same slot with the same `subject-hash` and the relaunch
succeeded.

Advisory findings, all `SUGGESTION` / `informational`, none of which opened a correction:

| id | Lineage | Lens | Location | Disposition |
| --- | --- | --- | --- | --- |
| `R2-1` | 1 | readability | `cmd/commands/delete.go:34` | applied in WU4 (blank line removed) |
| `R2-2` | 1 | readability | `cmd/gitutils/git.go:207` | applied in WU4 (rename + comment) |
| `R2-3` | 1 | readability | `cmd/tests/idempotent_delete_test.go:87` | applied in WU4 (intent comment) |
| `R2-1` | 2 | readability | `cmd/gitutils/git.go:205-207` | applied in WU4 (same spot as above) |
| `R2-2` | 2 | readability | `cmd/gitutils/git.go:206` | applied in WU4 (same spot as above) |

Three advisory hits from two independent reviews on one location is the signal that made WU4
worth doing rather than deferring.

Delivery is a separate decision under ordinary repository policy: an approval is a review
outcome, not an authorization to merge, push or open a pull request. The branch remains
local-only (`dflow start` was run with `--no-push`).
