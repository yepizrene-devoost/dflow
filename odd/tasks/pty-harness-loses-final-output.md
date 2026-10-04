# pty-harness-loses-final-output — feature tracking

Branch: `bugfix/pty-harness-loses-final-output` (base: `develop` at `a4e0303c2bb6b7228000741790168c1eddc5ea7e`)
Issue: none. This branch's only finish target is `develop` under merge mode `auto`, so it reaches the
repository without a pull request and no issue approval gates its delivery. The defect it fixes is a
CI failure on `develop` left by the merge of the `dflow init` fix for #60.
Status: implemented and independently verified; staging and commit pending explicit authorization.

## Goal

The interactive test harness must keep every byte the child writes, including its final block,
however busy the consumer is while the child exits.

## Why this now

The `Go` workflow failed on `develop` for the merge `a4e0303` (`TestInitCompletesOnboardingWithoutOriginRemote`,
`cmd/tests/init_guard_test.go:182`, `onboarding did not complete:`). Two details identified it as a
harness defect rather than a defect in the `dflow init` fix: the process **exit code was 0** — the
assertion immediately above it passed — and the captured transcript stopped right after the last
prompt. It passed 12/12 on macOS.

`runInteractiveCLI` drained `cmd.StdoutPipe()` in a goroutine while `cmd.Wait()` ran concurrently.
`os/exec` documents that `Wait` closes that pipe once the process exits, so whatever the reader had
not taken yet is discarded without an error. The reader is deliberately idle in exactly that window:
`terminalDriver.reply` sleeps `replyGap` (50 ms) while `consume` holds the driver's mutex, pacing
required by `survey`'s cursor reader.

Two consequences made this worth fixing now rather than tolerating:

- The other interactive tests pass only because they assert on early output. The flaw was latent for
  every one of them and the new test merely happened to assert on the last lines.
- `main` runs the same workflow on push, so promoting the pending `v0.5.1` release without this fix
  would turn `main` red at the moment of the merge, since the failing test travels with it.

## Tasks

1. [x] WU1 — diagnose the failure and prove the mechanism. Reproduce the pattern in a standalone
   program outside the repository, with the reader's pacing modelled, and show that the tail is lost
   deterministically while a `Writer` on `cmd.Stdout` keeps it even with a slow consumer.
2. [x] WU2 — fix the harness. Replace the hand-drained pipe with an `io.Writer` assigned to
   `cmd.Stdout` and `cmd.Stderr`, so `os/exec` owns the copy goroutine and `Wait` waits for it.
3. [x] WU3 — pin it and verify it. A deterministic regression pin, an independent verification, and a
   run on the CI platform itself.
4. [ ] WU4 — commit identity: recorded by the follow-up `docs(odd)` commit, the repository's one
   allowed exception, because a commit cannot contain its own id.

## Decisions recorded

- **A separate bugfix branch, not the release branch** (user decision). The release branch had a
  precedent for carrying code (`c829712` on `release/v0.5.0`), but this route makes `develop` green
  immediately and the fix reaches the release by bringing `develop` into `release/v0.5.1` before the
  pull request to `main`, which is what keeps `main` green at promotion.
- **The pin forces the window instead of racing for it.** `pinSinkPacing` (400 ms) outlives the
  child's remaining lifetime (`pinFinalChunkDelay` plus one write and an exit), so the loss window is
  open by construction and only the mechanism question remains. A pin that reproduced the flake by
  timing luck would be worthless.
- **No production code changed.** The defect lives entirely in the test harness.
- Residuals accepted and recorded rather than hidden, all non-blocking: (a) `Wait` now joins the
  copy goroutine, so after the timeout branch kills only the direct child, a surviving grandchild
  holding the pty can keep that goroutine alive until it exits or the test binary ends — bounded, and
  only reachable on an already-failing path, where the test has already reported the timeout;
  (b) the pin's trailing `HasSuffix` assertion is implied by the exact-equality assertion above it,
  kept as documentation of intent; (c) the pin can false-negative only if a machine delayed the
  child's 50 ms sleep past 400 ms, which is a 7.5x delay and was not observed in 10 of 10 runs.

## Out of scope

- Re-testing the alternative hypothesis that `script(1)` drops its own buffered tail on exit. The fix
  touches only the parent's capture path, so that hypothesis is not disproved here; it is instead
  covered by running the repository's checks on the CI platform itself, containerised.
- Any production behaviour, any ODD document belonging to the release, and the release artifacts
  themselves, which stay stashed while this lands.

## Evidence

### WU1 — diagnosis

Standalone reproduction of the harness's pattern, outside the repository:

```
reader drains immediately -> captured="prompt\nCreated .dflow.yaml\ndflow is ready\n"  tail_present=true
reader pauses before its next read -> captured="prompt\n"  readErr=read |0: file already closed  tail_present=false
```

The replacement pattern, with a consumer deliberately slower than the child:

```
writer delay=0s    -> complete, tail_present=true
writer delay=300ms -> complete, tail_present=true
```

### WU2 — the fix

`cmd/tests/agent_cli_test.go`: capture extracted into `runCapturedCLI(t, timeout, cmd, answers,
sinkPacing)`, which `runInteractiveCLI` delegates to with pacing `0`; `outputSink` is assigned to
both `cmd.Stdout` and `cmd.Stderr` as the identical pointer value, which is what makes `os/exec` use
one pipe and one copy goroutine and therefore one ordered transcript; the dead
`terminalDriver.drive` reader loop and its `StdoutPipe` wiring are gone; the reason is recorded in a
comment where the wiring is, and the comment names the pin so a re-introduction meets the argument.
Unchanged: `replyGap` and its `survey` rationale, the DSR query/report handling, the driver's mutex,
the reactive trigger matching, the timeout path, and the `*exec.ExitError` exit-code extraction. No
assertion, expected value, timeout or skip condition was modified.

### WU3 — pin and verification

The pin (`cmd/tests/pty_transcript_test.go`, `TestCapturedCLIKeepsOutputWrittenBeforeExit`) calls the
same `runCapturedCLI` the interactive tests use, so it exercises the real capture path, and asserts
the exact transcript with the final line present. Against the pre-fix wiring it fails **10 of 10**;
restored, it passes **10 of 10**.

Independent verification: verified with findings, all non-blocking. Scope confirmed to the two test
files; `go build ./...`, `go vet ./...`, `gofmt -l .`, `git diff --check` and
`go test -count=1 ./cmd/... ./pkg/...` all green; every removal in the diff accounted for as the dead
reader, its call site, or an error-message argument; `cmd.Stdout` and `cmd.Stderr` confirmed as the
identical comparable value, with the `os/exec` `interfaceEqual` path quoted as the reason; split-DSR
robustness probed by feeding one stream split at every byte boundary; and the other two interactive
consumers (`TestInitCLIWiresAgentReferences`, `TestInitCLIUpdatesExistingClaudeMd`) confirmed passing
with unchanged assertions.

Platform verification, on the CI platform rather than the development host: a `golang:1.21-bookworm`
container (`go1.21.13 linux/amd64`, `git 2.39.2`, `util-linux script 2.38.1` from `/usr/bin/script`,
which is the `-q -e -f -c` path the harness takes only on Linux). With the pre-fix wiring the pin
fails 3 of 3 with the exact signature of the CI failure — the child's final block written and the
capture stopped at the first chunk. With the fix, both the pin and the previously failing test pass
5 of 5. The workflow's own commands (`go build -v ./...` then `go test -v ./...`) were then run
against the fixed tree in that container: green across all ten packages, `cmd/tests` in 114.6s.

Two `pkg/repository` failures appeared in that first container run and were traced to the experiment
itself, not to the tree: the copy had been taken with `rsync` excluding `.git`, and those tests need a
real worktree. Re-run with `.git` present, `TestDiscoverFallsBackToProcessWorkingDirectory` passes.
Recorded because a reader of this evidence will meet that intermediate output.

### WU4 — commit identity

OUTCOME: pending, written by the follow-up `docs(odd)` commit that records this work unit's commit id.

## Native review

Pending: the candidate is this work unit's commit, and the review runs after the freeze.
