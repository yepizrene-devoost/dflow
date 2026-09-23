# closure-policy-corrections — feature tracking

Branch: `bugfix/closure-policy-corrections` (base: `develop`)
Issue: #24 — `chore: clear the residuals left by review-30d6236cfec24fe0 and the closure-policy text`
Status: authorized. The maintainer's instruction for this branch is explicit: the residuals are
corrected here, and no derived issue is opened for them.

## Why this branch exists

Two independent things were left over by the previous work unit (`ae7476a`, issues #21–#23), and
both are corrections to work that already shipped:

1. **The closure policy carries a factual error.** An independent verification of the two
   observables the rule tells a reader to act on found the trigger right and its stated reason
   wrong. Issue #22 was closed on that text.
2. **Three non-blocking advisories** from the approved receipt `review-30d6236cfec24fe0` — `R2-1`,
   `R2-2` (readability, `SUGGESTION`) and `R3-1` (reliability, `WARNING`).

Both groups were originally parked in #24 as follow-ups. The maintainer's decision is that they are
fixed in this candidate instead, so #24 closes with the work rather than staying open.

## Verified defects in the closure policy

Evidence: independent verification against a binary built from `13a2674`, in temp fixtures, with
the repository untouched (`git status --porcelain` empty before and after).

| # | Claim in the policy | Reality |
| --- | --- | --- |
| 1 | "Check it at the finish, **before the branch is deleted**" | `dflow finish` does **not** delete the branch. Without `--delete` it survives and stays dry-runnable (`git show-ref --verify` → `BRANCH_EXISTS`). `--delete` removes it only when no manual target remains; with a manual target left it prints `⚠️ Skipping delete ... because manual follow-up is still required.` |
| 2 | A post-finish dry-run can fail because the branch was deleted | In the normal, no-delete case the failure is `❌ branch "develop" does not match any configured dflow prefix` — the post-finish checkout is not a work branch. The operational advice holds; the attributed cause does not. |
| 3 | reports `Manual targets: none` | The rendered line is `ℹ️  Manual targets: none` — U+2139, U+FE0F and **two** spaces (`e284 b9 ef b8 8f 20 20`). Substring matching works; whole-line comparison does not. |
| 4 | machine equivalent is `"manual_targets": []` | The document is compact: `"manual_targets":[]`. Field and value are right; the spacing is not present in the output, so a substring match on the policy's literal form fails. |

Plus a gap found while applying the rule to #22 itself: it removes `status:approved` and
`status:needs-review` **by name**, so a `status:needs-design` label — a state that had equally
ended — survives a faithful reading of the rule.

## Advisories from the approved receipt

The reviewer's own wording is **not retrievable** in this environment: the payload travelled through
the host relay into the native review store, the facade exposes no read-back, and
`gentle-ai review reopen-results` is a maintainer quarantine mutation rather than a reader. What the
receipt certifies is identity, lens, location, severity and disposition. So the work below addresses
the substance at each location, not a quoted requirement, and says so where it matters.

## Tasks

1. [x] WU1 — issue #24, policy text (`.agents/workflows/dflow-workflow.md`)
   - Correct the timing reason: `finish` keeps the branch unless `--delete`, so the check is "while
     the branch still exists as a work branch", not "before it is deleted".
   - Quote the rendered line with its icon and spacing, and say explicitly that the substring is the
     signal.
   - Write the JSON equivalent as the parser-facing contract it is, without invented spacing.
   - Generalize the label rule: remove every `status:` label naming a state that has ended, keep the
     `type:` label, and name `status:needs-design` as the case that exposed the gap.
   - Evidence: see WU1 under `## Evidence`.
2. [x] WU2 — `R2-1`: no flag state in a package-level variable (`cmd/root/version.go`)
   - `showRevisionOnly` disappears; `--revision` is resolved inside `RunE`, where a lookup failure
     can be returned instead of being carried through package state. `PreRunE` keeps only the format
     decision, which genuinely must be made before the command renders.
   - Behaviour unchanged: `dflow version --revision` still prints one token, `--json` still wins.
   - Evidence: see WU2 under `## Evidence`.
3. [x] WU3 — `R2-2` and `R3-1`: the two-half decision, named and pinned (`cmd/gitutils/git.go`)
   - `Delete` decides both halves once, before the spinner starts and before either copy is touched,
     so the "remote could not be checked" outcome is a single explicit branch rather than a return
     dropped in the middle of the operation. Message texts stay byte-identical; no behaviour moves.
   - `R3-1` is answered by pinning what the receipt called out and the wording cannot confirm: the
     partial-success state is deliberate and must stay observable through tests — local deleted with
     an uncheckable remote exits non-zero naming the half that is gone, a re-run with nothing local
     and an unreachable `origin` touches nothing, and a re-run with no `origin` at all still exits 0.
   - Evidence: see WU3 under `## Evidence`.
4. [x] WU4 — checks, commit identity and close-out
   - Full checks, the work-unit commits, the native review of this candidate, and the merge that
     closes #24.
   - Evidence: see WU4 under `## Evidence`.

## Out of scope

- Any behaviour change to `dflow finish`, `dflow delete` or `dflow version`. This branch corrects a
  text and answers advisories; if the reliability advisory turns out to want different semantics,
  that is a separate decision with its own candidate.
- Backfilling the policy onto #17–#20 again: already done and verified in the previous work unit.
- The documentation issue template from #23: still not created, still conditional.

## Evidence

### WU1 — the closure policy corrected (issue #24)

Written by a bounded `gentle-ai-worker` (task `muecrwh6-6-w8qj`) in an isolated worktree, then
applied here as a patch. Documentation only; `AGENTS.md` needed no change, because its
`## Source Of Truth` reference to "issue closure" was already correct.

Every claim the writer put in the text was re-verified against a binary built from the previous
tip, in throwaway fixtures, with the repository untouched. The evidence it returned, which is where
the four corrections come from:

- The `--delete` sentence quoted from the CLI itself: *"The source branch is not deleted
automatically unless you explicitly pass `--delete` and no manual targets remain"*
(`cmd/commands/finish.go:35-36`).
- A plain `finish` leaves the branch in place (`git show-ref --verify refs/heads/<branch>` → a real
ref, and a later `finish --dry-run` on it exits 0); `--delete` without a manual target removes it
(`not a valid ref`, exit 128); `--delete` with a manual target keeps it
(`⚠️  Skipping delete ... because manual follow-up is still required.`).
- The chrome: the trigger line dumps as `e2 84 b9 ef b8 8f 20 20` before `Manual targets: none` —
U+2139, U+FE0F and two spaces (`cmd/utils/messages.go`, `printWithIcon`'s `%-3s`).
- The compact document: `"manual_targets":[]`, with no space, which is why the rule now says to
parse the field rather than match a spacing that does not exist.
- The label gap, from #22's own timeline: `labeled status:needs-design` then `unlabeled`, i.e. a
state that had ended while the rule named only the other two.

**One controller review edit.** The writer had embedded the unrelated diagnostic
`❌ branch "develop" does not match any configured dflow prefix` inside the rule as the explanation
for a later failed check. That is the same class of defect this branch exists to remove: a fresh
quote of a message that is not the trigger and can change under the rule. It was replaced with the
cause in prose (the base branch is not a work branch, not that a branch was deleted).

Disclosure the writer volunteered: it ran `rm -rf` over four as-yet-uncreated temp paths while
recreating fixtures, against its own rule. No repository content was involved and it verified
`git status --porcelain` empty before and after, but the deviation is recorded rather than dropped.

Escalation handled: the writer received the review reminder for its own worktree candidate and
correctly refused to start a review (the facade is not in its tool list), handing the disposition
to the controller. The controller declined to review that candidate, because an intermediate
worktree is discarded after integration and reviewing it would burn an authority on a tree that
ceases to exist and force a second review of the same edits.

### WU2 and WU3 — the advisories (`R2-1`, `R2-2`, `R3-1`)

Written by a bounded `gentle-ai-worker` (task `muecsemb-7-2isv`) in an isolated worktree, then
applied here. The reviewer's own wording for all three is unavailable (see above), so the writer was
instructed to work the substance at each location and say so — not to answer a quoted demand.

- **`R2-1`** — `showRevisionOnly` is gone. `PreRunE` keeps only the format decision, with a comment
  explaining why that asymmetry is deliberate; `--revision` is resolved in `RunE` next to the branch
  that consumes it, so a lookup failure is returned instead of parked in package state. `--json`
  still wins, because the revision lookup sits after the machine-readable branch.
- **`R2-2`** — the failed-lookup outcome is settled in one explicit branch before the spinner and
  before either copy is touched, and the local deletion became a named `deleteLocalBranch` step so
  its message cannot drift between the two call sites. Every message is byte-identical to the
  previous revision.
- **`R3-1`** — answered by pinning the observable contract rather than guessing at a change: the
  partial-success comment states that a failure after the local half exits non-zero while naming
  the half already deleted, and two real-binary tests fix the two runs of that state
  (`TestDeleteSurfacesAPartialSuccessWhenTheRemoteLookupFails`,
  `TestDeleteStillRefusesARerunWithNoOriginOnceTheLocalHalfIsGone`). Both passed against the
  unmodified code and are reported as characterization, not RED→GREEN: they pin behaviour that was
  already there and would have gone unnoticed had it drifted.

**The writer corrected the controller's brief, and was right.** The brief's fourth pinning case said
that re-running the delete with no `origin` at all, after a successful local-only delete, still exits
0 with the skip message. That is false: with neither copy left the code takes the unchanged refusal
(`nothing to delete`, non-zero), which is the correct reading of #19's contract and matches both the
`Delete` doc comment and the command's help. The writer refused to encode the wrong assertion, pinned
the real behaviour with the reason, and reported the divergence instead. The controller re-verified
it end to end: see the table under WU4.

Accepted trade-off, recorded rather than smoothed over: the restructure creates the spinner in two
places instead of one, so the same message construction appears twice. That is the price of keeping
the unknown outcome out of the middle of the operation, and it is two lines confined to one
function; a third shape that shares the spinner would put the return back in the middle that `R2-2`
flagged.

### WU4 — checks, identity and the freeze declaration

Controller-run checks on the integrated tree: `go build ./...`, `go vet ./...`, `gofmt -l .` clean;
`go test -count=1 ./...` green (`ok cmd/tests 23.472s`, `ok cmd/utils 0.334s`); `git diff --check`
clean. The two writers ran the same full suite inside their own worktrees and reported green.

End-to-end verification by the controller, with a binary built from this tree, in temp fixtures:

| case | observed |
| --- | --- |
| `version` / `--revision` / `--json` / `--revision --json` | `dflow dev ae7476a-dirty`, the full hash, one document; JSON wins |
| local copy, `origin` unreachable | local deleted, exit 1, message names the half gone and the half uncheckable |
| re-run in that state | exit 1 on the lookup, nothing touched |
| no `origin`, local copy present | exit 0 with the absent-skip message |
| no `origin`, no copy left | the unchanged refusal, exit 1 |

Work-unit commit identity on `bugfix/closure-policy-corrections`:

| Unit | Commit | Subject |
| --- | --- | --- |
| WU1 | `3c9d1d9` | `docs(workflow): correct the issue-closure rule against the real finish behavior` |
| WU2 | `f9db08d` | `refactor(version): resolve the revision flag where it is used` |
| WU3 | `ca10435` | `refactor(gitutils): settle the delete outcome before either copy is touched` |

This document's own commit is the follow-up `docs(odd)` commit the ODD rule allows: it carries the
identity record above and closes this stage, and it is the **last tracked write before the freeze**.

Declared expectation, not a verdict: with the tree frozen, a native review runs over this candidate
against `develop` (`ae7476a`). Its outcome belongs to the native receipt and to Engram, never to a
later commit on this branch — post-approval the tree is not touched again. #24 closes with the merge
that carries this candidate, so no derived issue is left open.

`size:exception` is **not** claimed here: the changed lines are far below the 400-line review budget,
unlike the previous work unit which needed the label.
