# receipt-advisory-corrections — feature tracking

Branch: `bugfix/receipt-advisory-corrections` (base: `develop`)
Issue: #24 — `chore: clear the residuals left by review-30d6236cfec24fe0 and the closure-policy text`
(reopened for this unit; no derived issue is created)

## Why this branch exists

`review-c189051ea818fe19` closed **approved** and listed four advisories. The receipt declares all
four non-blocking, none opened a correction, and none is a reason to re-run that review over its
candidate. The maintainer's instruction is that they are corrected now, in this session and in a
dedicated candidate, rather than left recorded inside a closed issue.

Their prose is not retrievable (see the note in `odd/tasks/closure-policy-corrections.md`), so each
item below states the substance found **at the location** and says so, instead of quoting a
requirement that cannot be read back.

## The four advisories and what this branch does about them

| id | severity | location | substance at that location | change |
| --- | --- | --- | --- | --- |
| `R3-spinner-state` | `WARNING` | `cmd/gitutils/git.go:213-215` | The spinner is created in two places and terminated by `Clear()` scattered across the error paths | one creation site and one termination, via `defer spinner.Clear()`; `Stop` and `Clear` share `stopOnce`, so the deferred call is a no-op on the success paths |
| `R3-remote-error-handling` | `SUGGESTION` | `cmd/gitutils/git.go:240-242` | the choice between the two remote-failure messages is made inline | the decision moves to one place; both messages stay byte-identical |
| `R3-test-binary-dependency` | `SUGGESTION` | `cmd/tests/idempotent_delete_test.go:260` | the partial-success contract is pinned only through a freshly built CLI binary | a package-level test drives `gitutils.Delete` directly; the CLI-level test stays, because the exit code is only observable in a process |
| `R3-documentation-assertion` | `SUGGESTION` | `odd/tasks/closure-policy-corrections.md:45-51` | the paragraph states as a standing fact that the reviewer prose "is not retrievable" and that `reopen-results` is a quarantine rather than a reader | reframed as observed in this environment and this version, with the evidence it rests on, so it cannot read as a permanent claim about the tooling |

## Tasks

1. [x] WU1 — the code advisories
   - Single spinner creation and a single `defer`-based termination; every error path stops repeating
     `spinner.Clear()`.
   - One decision point for the remote-failure message pair.
   - No user-visible message changes: every string stays byte-identical.
   - Evidence: see WU1 under `## Evidence`.
2. [x] WU2 — the test and documentation advisories
   - A package-level test for the partial-success contract that does not need the built binary.
   - The `closure-policy-corrections.md` paragraph reframed as environment-observed.
   - Evidence: see WU2 under `## Evidence`.
3. [x] WU3 — checks, identity and close-out
   - Full checks, work-unit commits, the native review of this candidate, the merge, and the close of
     #24.
   - Evidence: see WU3 under `## Evidence`.

## Out of scope

- Any change to `dflow finish`, to the closure policy text, or to `dflow version`: this branch
  answers advisories only.
- Re-litigating the trade-off the previous candidate accepted. The spinner change removes the
  duplication; it must not reintroduce the return in the middle of the operation that `R2-2`
  flagged, and the partial-success comment stays.
- Creating a derived issue: the findings live in the receipt and in #24.

## Evidence

### WU1 — the code advisories

Written by a bounded `gentle-ai-worker` (task `muedn7g2-8-ro5a`) in an isolated worktree, then
applied here as a patch.

**`R3-spinner-state`.** The spinner now has one creation site and one termination:

```go
if remoteErr != nil && !localExisted { return remoteErr }
if !localExisted && !remoteExisted  { return refusal }

spinner := utils.NewSpinner(...)
spinner.Start()
defer spinner.Clear()
```

Every `spinner.Clear()` that had been scattered across the error paths is gone, and the two outcomes
that touch nothing are settled before the spinner exists. The deferred call is safe after a success
path's `Stop` because both terminators share one guard — `cmd/utils/spinner.go`, in `terminate`:
`s.stopOnce.Do(func() { close(s.done); render() })`. That line, not an assumption, is what makes the
first call win and the second a no-op.

The explicit partial-success branch survived the change, so the `R2-2` improvement is not traded
back: the block that reports "the remote half could not be checked" after the local half was deleted
is still there with its comment.

**`R3-remote-error-handling`.** Both remote-failure messages moved into one function,
`remoteDeleteFailure(branch, localExisted, diagnostics)`, which trims the captured diagnostics once
and chooses between the two variants. Every string is byte-identical to the previous revision,
including the interpolated Git text; the repository's own
`TestDeleteReportsWhatRemainsWhenTheRemoteStepFails` asserts that wording and passes.

The remote-failure path was also exercised end to end by the controller, with `origin` set to a
non-bare repository holding the branch checked out, so `git push origin --delete` is rejected: the
local half was deleted, the remote branch was still there, and the command exited 1 naming the half
that remains.

### WU2 — the test and documentation advisories

**`R3-test-binary-dependency`.** `TestDeleteReportsPartialSuccessWithoutABuiltBinary` drives
`gitutils.Delete` directly over the same offline fixture (`origin` at a path that does not exist),
asserting the returned error and the real branch state without a built binary. It deliberately omits
the exit code, which only a process can observe, and it is the case the receipt's location pointed
at. It passed against the unmodified code, so it is reported as characterization: it pins behaviour
that was already there and would not have been noticed had it drifted. The binary-level test is
untouched.

**`R3-documentation-assertion`.** The paragraph in `odd/tasks/closure-policy-corrections.md` no
longer states the reviewer-prose limitation as a standing fact about the tooling. It now says
"observed in this environment, at `gentle-ai 3.6.1`", lists the `gentle-ai review` subcommands it was
drawn from, and quotes the `--help` sentence describing `reopen-results` as an
"exact-revision, maintainer-authorized same-lineage quarantine of unusable reviewer results" gated
behind `--actor`, `--reason` and `--maintainer-authorization`. The controller re-verified both the
version string and the subcommand list against the installed CLI rather than trusting the citation,
because an invented version would have been the same class of defect this whole line of work is
about.

### WU3 — checks, identity and the freeze declaration

Controller-run checks on the integrated tree: `go build ./...`, `go vet ./...`, `gofmt -l .` clean;
`go test -count=1 ./...` green (`ok cmd/tests 25.155s`, `ok cmd/utils 0.552s`); `git diff --check`
clean.

End-to-end verification with a binary built from this tree, in temp fixtures:

| case | observed |
| --- | --- |
| local copy, `origin` unreachable | local deleted, exit 1, both halves named — byte-identical to the previous revision |
| no `origin`, local copy present | exit 0 with the absent-skip message |
| no `origin`, no copy left | the unchanged refusal, exit 1 |
| live `origin` rejecting the remote delete | local deleted, remote branch still present, exit 1 naming the half that remains |

Work-unit commit identity on `bugfix/receipt-advisory-corrections`:

| Unit | Commit | Subject |
| --- | --- | --- |
| WU1 | `7342d68` | `refactor(gitutils): terminate the delete spinner in one place` |
| WU2 (test) | `d719bd7` | `test(gitutils): pin partial success without a built binary` |
| WU2 (doc) | `1ff8ca9` | `docs(odd): scope the reviewer-prose claim to this environment` |

This document's own commit is the follow-up `docs(odd)` commit the ODD rule allows: it carries the
identity record above and closes this stage, and it is the last tracked write before the freeze.

Declared expectation, not a verdict: a native review runs over this frozen candidate against
`develop` (`8a20864`). Its outcome belongs to the native receipt and to Engram, never to a later
commit on this branch. #24 closes again with the merge that carries this candidate.

Process note worth keeping: this is the third work unit in a row in which the delegated writer
received the review reminder for its own intermediate worktree and escalated it correctly instead of
inventing a lifecycle from the shell. The parent owns the candidate; a brief should say so up front
so the writer spends no turn on it.

`size:exception` is **not** claimed: this candidate is far below the 400-line review budget.
