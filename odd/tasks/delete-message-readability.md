# delete-message-readability — feature tracking

Branch: `bugfix/delete-message-readability` (base: `develop`)
No forge issue: the maintainer's instruction for this session is that no issue is opened, reopened or
followed up for this work. The record is this document, the native review receipt and Engram.

## Why this branch exists

Receipt `review-3b2a030239fff304` closed **approved** with exactly one advisory left — `R2-001`,
readability, `SUGGESTION`, `cmd/gitutils/git.go:268`:

```go
return fmt.Errorf("deleted local branch '%s' but failed to delete remote branch '%s': %s", branch, branch, diagnostics)
```

The line carries a sentence that also exists two lines below, repeats `branch` twice, and packs three
`%s` into one statement, which is what the readability lens flags. It was deliberately not pursued in
that candidate and the maintainer's decision now is to correct it here rather than leave it.

## What "correcting it" means, given the constraint this file lives under

Every message in `Delete` is byte-identical to the revision the tests and the previous receipts
describe. That constraint does not relax because the finding is cosmetic:

- The remote-delete sentence is built **once** and reused, so the two variants cannot drift in wording
  while gaining one reading path instead of two long literals.
- Both variants keep their exact bytes, including the interpolated Git diagnostics.
- The byte-identity stops being prose in a document and becomes **machine-checked**: an in-package test
  pins both variants' exact strings, which is only possible now that the sentence is composed in one
  place.

## Tasks

1. [x] WU1 — compose the remote-delete sentence once
   - `remoteDeleteFailure` builds `failed to delete remote branch '<branch>': <diagnostics>` in one
     place; the `localExisted` variant reuses it inside
     `deleted local branch '<branch>' but <that sentence>`. Both outputs byte-identical.
   - No other change in `Delete` or in its call sites.
   - Evidence: see WU1 under `## Evidence`.
2. [x] WU2 — pin the byte-identity
   - In-package test in `cmd/gitutils` (the repository already has in-package tests, e.g.
     `cmd/utils/version_test.go`) asserting the exact string of both variants, so the composition
     cannot silently change what any user sees.
   - Evidence: see WU2 under `## Evidence`.
3. [x] WU3 — checks, identity and close-out
   - Full checks, work-unit commits, the native review of this candidate and the merge.
   - Evidence: see WU3 under `## Evidence`.

## Out of scope

- Any behaviour change in `dflow delete`, and any wording change in any message: this is a
  readability fix under a byte-identity constraint.
- The other advisories from earlier receipts: already answered and merged.
- Anything in the forge: no issue is created, reopened or commented for this unit.

## Evidence

### WU1 — the sentence composed once

Written by a bounded `gentle-ai-worker` (task `muefh0je-1-69ts`) in an isolated worktree, then
applied here and verified with `cmp` against the worktree before committing.

`remoteDeleteFailure` now builds the remote sentence as its first statement and the `localExisted`
variant composes over it:

```go
remoteFailure := fmt.Errorf("failed to delete remote branch '%s': %s", branch, strings.TrimSpace(diagnostics))
if !localExisted {
	return remoteFailure
}
return fmt.Errorf("deleted local branch '%s' but %s", branch, remoteFailure)
```

Three properties the advisory's neighbourhood depends on, all preserved deliberately:

- **Byte-identity.** The second variant is the sentence unchanged; the first is the same sentence
  with `deleted local branch '<branch>' but ` in front of it. `strings.TrimSpace` still runs exactly
  once, on the same input. The existing tests that assert these messages passed before and after.
- **No new unwrap chain.** `%s` is used, not `%w`, so callers never gain an error chain they could
  not see before. The writer added a comment saying exactly that, because sharing wording is the kind
  of change that tempts a `%w`.
- **The shape.** A composed string inside the function rather than a new unexported helper: the
  sentence has one consumer, so a helper would have been a single-caller indirection that adds a name
  without removing a reading step.

### WU2 — the byte-identity pinned

`cmd/gitutils/git_messages_test.go` (new, in-package) pins both variants **whole**, plus three
triangulation cases: padded diagnostics trimmed on both variants, and empty diagnostics keeping the
trailing `: ` separator. The diagnostics carry a path containing a colon, so a moved separator or a
whitespace change cannot slip through a passing run.

**It also closed a real coverage hole.** The writer checked the repository rather than assuming:
the second variant (`localExisted=false`, remote copy exists, remote refuses the deletion) had **no
coverage at all**. The only prior reference was a substring match in `cmd/tests` at
`idempotent_delete_test.go:403`, and it runs in the `localExisted=true` case, so it matches the tail
of the *first* variant and never reaches the second. The new test exercises that branch directly for
the first time.

The test passed against the unmodified code, so it is reported as characterization rather than
RED→GREEN, and that same pass is the byte-identity proof: identical expectations satisfied by the
pre-change and post-change implementations.

### WU3 — checks, identity and the freeze declaration

Controller-run checks on the integrated tree: `go build ./...`, `go vet ./...`, `gofmt -l .` clean;
`go test -count=1 ./...` green (`ok cmd/gitutils 0.235s`, `ok cmd/tests 23.850s`, `ok cmd/utils
0.360s`).

End-to-end verification by the controller with a binary built from this tree, in temp fixtures:

| case | observed |
| --- | --- |
| no local copy, remote refuses the deletion (`receive.denyDeletes`) | `❌   failed to delete remote branch 'feature/remote-only': remote: error: denying ref deletion ...`, remote branch kept, exit 1 — **the second variant, exercised for the first time** |
| local copy present, remote refuses | covered by the repository's own `TestDeleteReportsWhatRemainsWhenTheRemoteStepFails`, green in the suite |

The controller's first attempt at the second row was a malformed fixture: the local branch name did
not match the remote one, so the command skipped the remote half instead of refusing it. Recorded
because the row was then backed by the repository's test rather than by that fixture, and the
difference matters when reading the table.

Work-unit commit identity on `bugfix/delete-message-readability`:

| Unit | Commit | Subject |
| --- | --- | --- |
| WU1 | `79428cf` | `refactor(gitutils): compose the remote-delete sentence once` |
| WU2 | `f006ff6` | `test(gitutils): pin both remote-delete failure variants` |

This document's own commit is the follow-up `docs(odd)` commit the ODD rule allows: it carries the
identity record above and closes this stage, and it is the last tracked write before the freeze.

Declared expectation, not a verdict: a native review runs over this frozen candidate against
`develop` (`a25a4d1`). Its outcome belongs to the native receipt and to Engram. Per the maintainer's
instruction for this session, **the forge is not touched**: no issue is created, reopened or
commented for this unit.

`size:exception` is not claimed: this candidate is a handful of lines.

Two process incidents worth keeping, both recovered without damage:

- The first attempt at this unit was a delegated writer whose task was cancelled mid-flight
  (`parent session shut down`) after four turns. It had written nothing — the worktree was clean and
  the `go install`/`dflow version` output it produced was verification noise, not a change.
- The controller then ran a chained command that attempted the `docs(odd)`-style commits in the main
  worktree **before** the writer's changes had been copied there. The commit failed with nothing
  staged and the chain stopped mid-way, leaving `git.go` copied but the test file missing. It was
  corrected by copying the missing file and proving both files identical to the worktree with `cmp`
  before any commit; no commit, push or review state was affected.
