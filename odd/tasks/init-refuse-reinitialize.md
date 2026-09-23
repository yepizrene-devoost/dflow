# init-refuse-reinitialize — feature tracking

Branch: `bugfix/init-refuse-reinitialize` (base: `develop`, the `uat` alias)
Issue: #20 — `fix(init): refuse to reinitialize a project that already has a .dflow.yaml`
Status: authorized; scope is the refusal, its `--force` opt-in and its regression coverage.

## Goal

`dflow init` must refuse to overwrite an existing `.dflow.yaml`, naming the file and
the explicit opt-in that unlocks regeneration, and exit non-zero without touching it.

## Why this now

`init` runs through `validators.WithChecks(true, ...)`, where that `true` skips the
`.dflow.yaml` check every other command performs, and `SaveConfig` writes
unconditionally. Completing the wizard therefore replaces the whole file — branch
names, prefixes, flow rules, finish targets and merge modes — with no warning and no
backup. `.dflow.yaml` is a hand-edited, committed, reviewable team contract, so the
loss is a silent regression of the branch model.

The remaining hole is the interactive path only: a non-interactive `dflow init`
already fails fast for lack of a terminal, so a script cannot clobber the file today.
The accident worth protecting against is a human typing `dflow init` out of habit.

## Tasks

1. [x] WU1 — refuse to reinitialize, with `--force` as the only destructive path
   - `pkg/validators`: add `EnsureDflowNotInitialized()`, the mirror of the existing
     `EnsureDflowInitialized()`, returning
     `this project is already initialized with .dflow.yaml; use --force to regenerate it`.
   - `cmd/commands/init.go`: declare `--force`, and run the guard as the first thing in
     the handler — before the terminal check — so the most specific refusal wins and the
     file is untouched. `WithChecks(true, ...)` stays as it is: `init` must not require a
     pre-existing config, and the missing half is the inverse check.
   - `--force` deliberately stops at the guard: it authorizes regeneration, it does not
     make the command non-interactive.
   - Evidence: see the WU1 OUTCOME under `## Evidence`.
2. [x] WU2 — regression coverage
   - Real-binary coverage in a new `cmd/tests/init_guard_test.go`: an initialized repo
     refuses `dflow init` non-zero with the file byte-identical, a single error icon and
     the message naming `.dflow.yaml` and `--force`; `dflow init --force` gets past the
     guard (it fails later for lack of a terminal) and leaves the file untouched; an
     uninitialized repo still reaches the terminal check.
   - A unit test for the validator itself, mirroring the repo's `os.Chdir` pattern.
   - `go build ./...`, `go vet ./...`, `gofmt -l .`, `go test -count=1 ./...`,
     `git diff --check`.
   - Evidence: see the WU2 OUTCOME under `## Evidence`.
3. [x] WU3 — docs
   - `README.md`: the `dflow init` section documents the refusal and `--force`.
   - `CHANGELOG.md`: a `### Fixed` entry under `Unreleased`.
   - `.agents/workflows/dflow-workflow.md`: the command table notes the destructive path.
   - Evidence: see the WU3 OUTCOME under `## Evidence`.

## Out of scope

- A backup of the replaced file: the issue asks for a refusal plus an explicit opt-in, not
  for a retention mechanism.
- Any guard inside `SaveConfig`. It stays the low-level writer with documented overwrite
  semantics; the refusal belongs to the command that owns the `--force` decision.
- Making the wizard drivable without a terminal, and any other `init` behavior.

## Evidence

### WU1 — refuse to reinitialize, with `--force` as the only destructive path

OUTCOME: done. `pkg/validators.EnsureDflowNotInitialized()` is the mirror of
`EnsureDflowInitialized()`: `os.Stat(".dflow.yaml")` succeeded means the file exists, and the
error carries exactly `this project is already initialized with .dflow.yaml; use --force to
regenerate it`. `cmd/commands/init.go` registers `--force` and runs the guard as the first
statements of the handler, before `utils.IsInteractive()`; the first survey prompt moved from
`:=` to `=` because `err` now exists in that scope. `Long` and `Example` document the refusal
and the opt-in. `WithChecks(true, ...)` and `utils.SaveConfig` are untouched by design: `init`
must not require a pre-existing config (the missing half was the inverse check), and
`SaveConfig` stays the low-level writer with documented overwrite semantics.

Independent verification (real binary, initialized temp repo, no TTY): exit `1`, exactly one
`❌` line reading the sentence verbatim, stderr empty, no usage/success chrome; SHA-256 of
`.dflow.yaml` identical before and after (`2ae71e31…986d`), and no backup or variant file
created. Blast radius checked per file: only `init` gained a flag, `dflow init --help` shows
it, and the guard provably runs before the terminal check.

Files changed: `pkg/validators/validators.go`, `cmd/commands/init.go`.

### WU2 — regression coverage

OUTCOME: done, with one correction applied after the first verification pass. New
`cmd/tests/init_guard_test.go` holds four tests that reuse the existing `buildDflowCLI`,
`setUpCLIEnv`, `initTempGitRepo`, `startCLIRawOutput` and `withWorkingDir` helpers:
`TestInitRefusesToReinitialize` (refusal wording, non-zero exit, exactly one `❌`, file
byte-identical against a hand-written `.dflow.yaml` with custom branches, flow rules, merge
modes and an extra key), `TestInitForceStillRequiresTerminal` (`--force` reaches the terminal
check and still leaves the file untouched), `TestInitFirstRunStillReachesPrompt` (an
uninitialized repo still reaches the terminal check) and `TestEnsureDflowNotInitializedUnit`
(`nil` in an empty dir, exact sentence when the file exists).

Correction: the first verification pass found that no test pinned the exact refusal sentence —
only three substrings and a `--force` containment check — so message drift was uncaught. Both
assertions now compare against one package-level `initRefusalMessage` const, which the second
pass confirmed is character-for-character identical to the production literal (82 codepoints,
no differing index). A mutation experiment in a throwaway copy of the repo (`regenerate` →
`recreate` in the production string) turned BOTH assertions red while the unmutated control
stayed green, so they discriminate; the real worktree was never mutated.

Checks: `go build ./...` ok, `go vet ./...` ok, `gofmt -l .` empty, `git diff --check` clean,
`go test -count=1 ./...` green (`ok .../cmd/tests 17.181s`), and `TestStartCLI` passes with all
12 subtests including `init_requires_a_terminal`.

Strict TDD was not active: no TDD runner is configured for this repository, so the tests are
regression coverage rather than RED→GREEN evidence.

Boundaries, all documented and non-blocking: the destructive `--force` regeneration under a
real TTY is not exercised end-to-end, because driving the `survey` wizard headlessly was
already proven unreliable in the session that filed the issue — the guarantee is proven up to
the terminal-check boundary, and the TTY path rests on `SaveConfig`'s documented overwrite. A
broken `.dflow.yaml` symlink is treated as "not initialized" (ENOENT), which is symmetric with
`EnsureDflowInitialized`; a `.dflow.yaml` that is a directory or an empty file is refused.
Residuals accepted: the three-substring loop is now strictly subsumed by the exact-sentence
assertion (kept, harmless), and the shared const does not machine-check the README/CHANGELOG
wording.

Files changed: `cmd/tests/init_guard_test.go`.

### WU3 — docs

OUTCOME: done. `README.md` documents the refusal and `--force` in the `dflow init` section,
including that `--force` only authorizes the overwrite and still requires a terminal;
`CHANGELOG.md` gains a `### Fixed` entry under `## Unreleased`;
`.agents/workflows/dflow-workflow.md` notes the destructive opt-in in the command table.
Independent verification scanned all three documents plus the command help for a claim the
code does not implement (for example that `--force` skips the wizard) and found none: the
wording matches observed behavior.

Files changed: `README.md`, `CHANGELOG.md`, `.agents/workflows/dflow-workflow.md`.

### Commit identity

Not yet recorded: the working tree holds the change and no commit has been made. Committing is
a human decision in this repository, so the work-unit commit identity is recorded in a short
follow-up `docs(odd)` commit once the human authorizes staging.
