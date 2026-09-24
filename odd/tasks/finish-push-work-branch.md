# finish-push-work-branch — feature tracking

Branch: `feature/finish-push-work-branch` (base: `develop`)
Closes forge issue #26 (`feat(finish): push the work branch as part of the finish flow`).

## Why this branch exists

`dflow finish` merges the work branch into its `auto` targets and pushes those
targets, but it never publishes the work branch itself: the branch reaches
`origin` only if the author ran `git push -u origin <branch>` by hand. That is a
backup gap for `auto` targets and a hard prerequisite for `manual` ones — a pull
request cannot be opened until the branch exists on `origin` — so every future
PR-based promotion would start with a manual Git command. Discovered while
finishing `feature/installer-cli` (issue #11).

## Design decisions

The issue proposed a solution and rejected two alternatives; those choices are
taken as given, and the decisions below are the ones it left open:

- **Always publish, idempotently.** The issue rejected "push only when a
  `manual` target exists": the backup value applies to `auto` finishes too, and
  one rule is simpler to explain than two. `--no-push` is the escape hatch for
  offline repositories and CI sandboxes, mirroring `dflow start`.
- **Publish before the first merge.** A merge that fails then leaves the
  reviewed candidate on `origin` instead of only on this machine, and the PR a
  `manual` target needs can be opened from the published branch.
- **Publish even when there is no `auto` target.** With only `manual` targets
  there is no merge loop, but publishing is the whole job: it is the step the
  pull request needs. The publish therefore cannot live inside the merge loop.
- **Idempotent means identity, not existence.** The remote commit for the branch
  is compared with the local one before pushing, so an already published,
  up-to-date branch is reported as such instead of claiming a push that did not
  happen. This repository already corrected one finish message that overstated
  what it had done; the new one starts honest.
- **Existence and identity share one remote lookup.** `remoteBranchRevision` is
  extracted as the single `ls-remote` primitive and `RemoteBranchExists` is
  defined over it, so the two questions cannot drift apart in the states they
  distinguish (absent, failed lookup, no `origin`). Verified behaviour-preserving:
  `ls-remote --heads origin <branch>` matches a ref name exactly, so the
  refactor changes no answer the previous implementation gave.

## Out of scope

- PR creation for `manual` targets. This unit only makes the published branch
  available to that flow; nothing opens a PR.
- `manual` targets themselves: `finish` still never merges them.
- Any change to `dflow start`'s `--push`/`--no-push` semantics.

## Tasks

1. [x] WU1 — publish the work branch before merging
   - `cmd/gitutils/git.go`: extract `remoteBranchRevision`; define
     `RemoteBranchExists` over it.
   - `cmd/gitutils/finish.go`: `PushWorkBranch` — skip without `origin`, no-op
     when the remote already holds this exact commit, otherwise
     `git push -u origin <branch>`, failing loudly.
   - `cmd/commands/finish.go`: publish before the merge loop and on the
     manual-only path, `--no-push`, and the dry-run lines plus the
     `publish_work_branch` field of the JSON plan.
   - Tests in `cmd/tests/finish_cmd_test.go`, CHANGELOG, README and the
     agent-facing workflow doc.
   - Evidence: see WU1 under `## Evidence`.
2. [x] WU2 — pin the publish contracts
   - `cmd/tests/status_cli_test.go` drives the real binary over
     `finish --dry-run --json`: it pins `publish_work_branch` true for the
     default plan and false for `--no-push`, so the documented contract cannot
     drift from the struct.
   - `cmd/tests/finish_cmd_test.go` gains the `--no-push` coverage the first
     round left open, and `cmd/tests/remote_lookup_test.go` pins the premise the
     identity comparison rests on: the remote lookup matches the ref name
     exactly, so a longer branch name on `origin` is never a match.
   - Evidence: see WU2 under `## Evidence`.
3. [x] WU3 — checks, identity, native review and close-out
   - Full checks on the integrated tree, the work-unit commit identity, the
     native review of the frozen candidate, and the finish/issue closing per
     the repository's issue policy.
   - Evidence: see WU3 under `## Evidence`.

## Evidence

### WU1 — publish before merging

Written by a bounded `gentle-ai-worker`. The first pass (task `muekhzlm-1-2mbh`) stopped with
`status: interaction_required`: two of the five specified tests encoded premises that could not
hold against the fixture the same task defined — the config commit lands on the work branch, so
pushing before it left `origin` behind, and the shared fixture pushes `develop` with `app.txt`, so
an `ls-tree` "not merged" assertion was false on arrival. Both observations were correct and the
parent's brief was at fault. The worker continued (task `muekp44r-2-owtr`) with a branch-scoped
failing pre-push hook and the config committed before the push.

Independent verification by a read-only `gentle-ai-verify` (task `mueks4sv-3-k3g2`) over the
uncommitted tree, all four checks green (`gofmt -l .`, `go build ./...`, `go vet ./...`,
`go test -count=1 ./...`):

- **Dry-run safety**, traced per path: `--dry-run` with auto targets, without them, `--dry-run
  --json`, and `--json` without `--dry-run` can reach neither `FetchOrigin`, nor `PushWorkBranch`,
  nor the merge loop.
- **`--no-push` completeness**: no path publishes with the flag set; with it unset the only
  non-publishing outcomes are the two by-design ones (no `origin`, origin already at the same
  commit).
- **Refactor equivalence**: the new `remoteBranchRevision` + `RemoteBranchExists` keep the same
  bool, the same nil/error outcome and the same error text as the previous implementation for all
  three states (no `origin`, unreachable `origin`, present/absent branch).
- It also flagged, correctly, that no test exercised `--no-push`, and that the JSON field and the
  exact-match premise were unpinned — both closed by WU2.

**The mutation check that changed the tests.** The controller disabled the identity short-circuit
in `PushWorkBranch` and ran the already-published test: it **still passed**. Git skips the pre-push
hook for a ref that is already up to date, so a failing hook cannot prove a push was skipped — the
verifier had flagged exactly this as reasoned-only, and the experiment settled it against the
design. The discriminator was replaced (`task muel2oa9-4-q7pf`): `remote.origin.pushurl` pointed at
a path that cannot be a repository, which makes any attempted push fail while `ls-remote` keeps
working, because ls-remote reads the fetch URL and only push uses the push URL (observed: lookup
exit 0, push exit 128). The reworked test is `TestFinishSkipsThePublishWhenOriginAlreadyHoldsTheCommit`.
The mutation check was re-run after the rework: **without** the short-circuit the test fails with
`failed to push branch 'feature/publish-me'` and the spinner shows the publish was attempted;
**with** it the test passes. The mutated file was restored byte-identical (sha256-verified) both
times.

### WU2 — the pins

Same continuation task (`muel2oa9-4-q7pf`):

- `status_cli_test.go` asserts `publish_work_branch` `true` in the existing `finish --dry-run
  --json` subtest and adds a `--no-push` subtest asserting `false`, both against the real binary
  with the no-human-chrome guard.
- `TestFinishNoPushKeepsTheWorkBranchLocal` closes the `--no-push` coverage gap: the work branch
  stays local, the auto target is still merged and pushed, the run ends on `develop`.
- `TestRemoteBranchExistsDoesNotMatchALongerBranchName` pins the exact-match premise: with only
  `feature/demo/sub` on `origin`, `feature/demo` is absent and the longer name is present.

### Known boundaries, deliberately not fixed here

- **The dry-run plan states intent, not capability.** On a repository without `origin`,
  `finish --dry-run` prints "Would publish … to origin" and the JSON plan emits
  `"publish_work_branch": true`, while a real run skips the publish. This extends the pre-existing
  dry-run convention — "Would merge … and push the target branch" has always overclaimed the same
  way on such repositories — and `publish_work_branch` reports the flag exactly as
  `delete_requested` reports its flag. Fixing it would mean teaching the plan about capability,
  which is a separate decision, not part of this unit.
- **A failed `fetch` aborts before the publish**, so the README's "a failed merge still leaves the
  branch on the remote" claim has a narrower sibling that no doc states: a failed fetch leaves it
  local. Not contradicted anywhere; left unstated.

### WU3 — checks, identity and the freeze declaration

Work-unit commit identity on `feature/finish-push-work-branch`:

| Unit | Commit | Subject |
| --- | --- | --- |
| WU1 | `c7b571c` | `feat(finish): publish the work branch before merging` |
| WU2 | `a79e969` | `test(finish): pin the publish contracts of the plan and the remote lookup` |

Checks behind this candidate: `gofmt -l .`, `go build ./...`, `go vet ./...` clean and
`go test -count=1 ./...` green (`cmd/gitutils`, `cmd/tests`, `cmd/utils`) on the integrated code,
run by the writer after the last code change; independently green from the read-only verifier on
the pre-rework tree; and re-confirmed after each mutation-check restore. This document's own
commit is the follow-up `docs(odd)` commit the ODD rule allows: it carries the identity record
above and closes this stage, and it is the last tracked write before the freeze. A read-only
verifier re-runs the four checks on the frozen tree; its result belongs to Engram.

Declared expectation, not a verdict: a native review runs over this frozen candidate against
`develop` (`231297a`). Its outcome belongs to the native receipt and to Engram. Issue #26 stays
open until the delivering merge into `develop` reports `manual_targets:[]`; the closing comment
then names that merge commit, per the issue-closure rule.
