# finish-publish-advisories — feature tracking

Branch: `bugfix/finish-publish-advisories` (base: `develop`)
No forge issue: these are the four informational advisories left by native review lineage
`review-25d63b77e9fc5143` (the approved review of `feature/finish-push-work-branch`, merged as
`fc40e1a`). The receipt's closure is explicit that they never reopen that review and are separate
later work; this branch is that later work, and the record is this document, the new review's
receipt and Engram.

## Why the findings are derived from ids and locations

The consumed lineage's full finding texts are no longer retrievable: the delivering merge
consumed the authority and the native store no longer lists the lineage, and only the closure
summary survives (id, lens, location, severity, disposition). Each correction below is therefore
derived from the advisory id, the exact location it names, and the code that stands there today.
The new review of this candidate judges the corrections on their own merits.

## The four corrections

1. **R2-dup-publish-skip-message** (`SUGGESTION`, `cmd/commands/finish.go:166`) — the sentence
   "Skipping publish of '%s' because --no-push was given." is spelled out three times (the dry-run
   manual-only path, the dry-run auto path, and the mutating manual-only path). Compose it once
   and reuse it, so the paths cannot drift apart in what they claim — the same shape as the
   `remoteDeleteFailure` composition the repository made before.
2. **R2-publish-field-overclaim** (`SUGGESTION`, `cmd/commands/finish.go:289-291`) — the
   `publish_work_branch` field comment says the plan reports whether the work branch "would be
   published", which overclaims: a run without an `origin` remote still skips it. State exactly
   what the field reports — the plan's publish step, the same class as `delete_requested` — not a
   capability guarantee. Comment-only; the JSON contract and its pinned tests stay as they are.
3. **R2-test-lookup-helpers-duplicate** (`SUGGESTION`, `cmd/tests/finish_cmd_test.go:353-365`) —
   the `remoteRevision` test helper repeats the `ls-remote` invocation the older `remoteBranchExists`
   helper already makes. Define the boolean helper in terms of the revision helper, so the test
   package has one remote-lookup site.
4. **R3-publish-failure-path** (`WARNING`, `cmd/gitutils/finish.go:246-252`) — the identity
   comparison that makes the publish idempotent is an optimization for the honest message, but its
   lookup failure (`origin` configured yet unreachable at that moment) aborted the whole finish.
   The push is the operation of record and reports its own failure; a pre-check that says nothing
   must not turn the cheaper path into a load-bearing one. Downgrade the lookup failure to a
   warning and attempt the publish anyway; only a push failure fails the finish. No new test: when
   the lookup fails the push fails with it (same remote), so every observable outcome is already
   covered by the existing tests.

## Out of scope

- Any change to the `publish_work_branch` JSON field's name or value: the field and its pinned
  contract tests stay byte-for-byte.
- The dry-run "intent, not capability" boundary for the target-push lines: documented, pre-existing,
  and larger than these advisories.
- The forge: no issue is opened, commented or closed for this unit.

## Tasks

1. [x] WU1 — close the four advisories
   - The four corrections above, in `cmd/commands/finish.go`, `cmd/gitutils/finish.go` and
     `cmd/tests/finish_cmd_test.go`.
   - Evidence: see WU1 under `## Evidence`.
2. [x] WU2 — checks, identity, native review and close-out
   - Full checks, the work-unit commit identity, the native review of the frozen candidate and
     the finish per the repository's issue policy.
   - Evidence: see WU2 under `## Evidence`.

## Evidence

### WU1 — the four corrections

Written by a bounded `gentle-ai-worker` (task `muem2i7m-6-1hep`) from the derived spec above, with
no interaction required: every named location matched its premise. Controller review of the diff
confirmed it. Checks on the writer's run: `gofmt -l .`, `go build ./...`, `go vet ./...` clean,
`go test -count=1 ./...` green (`cmd/gitutils 0.312s`, `cmd/tests 27.912s`, `cmd/utils 0.635s`),
with `TestFinishSkipsThePublishWhenOriginAlreadyHoldsTheCommit` and
`TestFinishFailsWhenTheWorkBranchCannotBePublished` explicitly green after the `PushWorkBranch`
restructure.

- The skip sentence is composed once (`skipPublishMessage`) and printed identically from the three
  paths; user-visible text is byte-identical.
- The `publish_work_branch` comment now claims the plan's publish step, not a capability; the JSON
  tag, value and field order are untouched, so the pinned CLI contract tests stand unchanged.
- `remoteBranchExists` delegates to `remoteRevision`: one `ls-remote` site in the test package.
- The identity comparison is no longer load-bearing: a failed lookup warns and the publish is
  still attempted; only a failed push fails the finish. The lookup-failure-with-successful-push
  outcome has no dedicated test by design — when the lookup fails the push fails with it (same
  remote), so every observable outcome is already covered.

### WU2 — checks, identity and the freeze declaration

Work-unit commit identity on `bugfix/finish-publish-advisories`:

| Unit | Commit | Subject |
| --- | --- | --- |
| WU1a | `612f4ef` | `fix(gitutils): stop failing the finish when the publish comparison errors` |
| WU1b | `5382868` | `refactor(finish): compose the skip-publish sentence once` |
| WU1c | `5d64527` | `test(finish): keep a single remote lookup helper` |

Checks behind this candidate: the writer's four-command run on the integrated code (green, per
WU1). This document's own commit is the follow-up `docs(odd)` commit the ODD rule allows: it
carries the identity record above and closes this stage, and it is the last tracked write before
the freeze. A read-only verifier re-runs the four checks on the frozen tree; its result belongs
to Engram.

Declared expectation, not a verdict: a native review runs over this frozen candidate against
`develop` (`fc40e1a`). Its outcome belongs to the native receipt and to Engram. No forge action
attaches to this unit: the advisories belong to the previous receipt, not to an issue.
