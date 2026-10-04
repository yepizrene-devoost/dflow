# init-empty-repo-and-missing-origin — feature tracking

Branch: `bugfix/init-empty-repo-and-missing-origin` (base: `develop`, the `uat` alias)
Issue: #60 — `[Bug]: dflow init fails on a repository with no commits and without an origin remote`
Status: implemented and independently verified; staging and commit pending explicit authorization.

## Goal

`dflow init` must not abort on a repository it cannot fully onboard, and must never fail
with a diagnostic that names neither the cause nor an action:

1. A repository with no commits fails fast, before any prompt and before any mutation, with
   a message that names the missing first commit and the next step. `.dflow.yaml` is untouched.
2. A repository without an `origin` remote completes onboarding: publication is skipped with an
   informational line, and the wizard does not ask a question whose answer cannot be acted on.

## Why this now

Discovered while onboarding a brand-new, intentionally local-only repository (a frontend app
kept off any remote while its backend is developed in parallel). Init died at the first real
mutation with `failed to create branch 'main': fatal: not a valid object name: 'main'`, and the
message pointed at neither the cause nor a fix.

Two independent causes, both established by reading the code and reproducing them locally with
git 2.50.1:

- `cmd/gitutils/git.go:68-86` (`CheckOrCreateBranch`) creates a missing branch with
  `git branch <name>` and no start point. In a repository with zero commits the existence probe
  cannot succeed and the creation cannot work either, because there is no object to point at.
  Every base branch is attempted in order (`main`, `develop`, `uat`), so the failure is the first
  of three.
- `cmd/commands/init.go:298-322` (`inspectInitRemoteBranch`) runs
  `git ls-remote --heads origin ...` without checking whether the remote exists. It is the only
  path in the CLI missing that guard: `finish` (`finish.go:180, 230, 267, 314`), `start`'s pull
  (`git.go:214-217`), `delete` (`git.go:417-419`), `status` (`status.go:117`) and the agent
  document generator (`generate.go:241`, "or the repo has no `origin`") all degrade gracefully.
  The publication prompt at `init.go:212` defaults to true, so accepting the default aborts
  onboarding in a repository that cannot be published.

## Tasks

1. [x] WU1 — a repository without commits fails fast with an actionable message
2. [x] WU2 — no `origin` means no publication and no publication question
3. [x] WU3 — regression coverage and the behavior's own help text
4. [ ] WU4 — commit identity: recorded by the follow-up `docs(odd)` commit (the repository's one
   allowed exception, because a commit cannot contain its own id)

## Decisions recorded

- **No initial-commit bootstrap.** Bootstrapping an empty commit so that init completes in a
  repository with no commits was considered and rejected. A repository with no commits can hold
  exactly one branch, and `git checkout -b` on an unborn HEAD does not create a branch — it
  renames the unborn HEAD (reproduced: `git checkout -b develop` on unborn `main` succeeds, after
  which `git branch main` fails with `not a valid object name: 'develop'`). The base branches are
  impossible until the first commit exists, so the honest behavior is a fast, actionable refusal.
  The bootstrap is a separate product decision, not part of a bug fix.
- **The changelog is a release-branch artifact, not a work-unit artifact.** This work unit carries
  no `CHANGELOG.md` entry, by explicit user decision. `RELEASING.md` step 3 puts both the
  generation (`make changelog` → `git cliff --unreleased --config cliff.toml -o CHANGELOG.draft.md`,
  `Makefile:120-128`, gitignored) and the curation into a versioned section on the `release/*`
  branch, with release-only documentation isolated there. Nothing is lost by deferring: because
  the generator reads unreleased commits, this work unit appears in the draft from its
  `fix(init): …` subject and is authored once, at release time. The behavior's own documentation
  does travel with the fix — it is the `dflow init` help text inside `cmd/commands/init.go`.
- **The skip message reuses the existing wording template.** `Remote 'origin' not found. Skipping
  base branch publication.` copies the sentence four sibling lines already use (`git.go:216`,
  `finish.go:181, 315, 360`) and their `📁` icon, rather than inventing a parallel voice for the
  same condition.
- **Two layers of the same guard, deliberately.** The preflight exists both in the command handler
  (before the terminal check, so the failure precedes every prompt) and at the top of
  `applyInitDraft` (so the unit-testable path cannot mutate a commitless repository either).

## Out of scope

- Creating an initial commit to bootstrap a repository with no commits (see the decision above).
- Changing `CheckOrCreateBranch`'s contract or its diagnostic; printing the new message one
  layer higher is sufficient and keeps the exported helper as documented.
- Any other `init` UX: prompt wording, ordering, the merge-mode and agent prompts.
- The `dflow init` behavior on a repository that has commits and an `origin`.
- The changelog entry, `HISTORY.md` and the release notes: release-branch work.

## Evidence

### WU1 + WU2 — the production change

OUTCOME: done. `cmd/commands/init.go` gained a commitless-repository preflight
(`requireInitRepositoryHistory`, called in `RunE` before `utils.IsInteractive()` and before
`collectInitDraft`, and again as the first statement of `applyInitDraft`), the injectable
operations `hasCommits`/`hasOriginRemote` with real defaults (`hasInitCommits`, which probes
`git rev-parse --verify --quiet HEAD` through the shared capture boundary, and
`gitutils.HasOriginRemote`), a publication question offered only when an `origin` exists, and a
publication block guarded so that the skip is reported from exactly one site.

Exact user-facing sentences, each held in one production constant:

- `this repository has no commits yet; create the first commit, then rerun \`dflow init\``
- `Remote 'origin' not found. Skipping base branch publication.`

`--force` still authorizes only regenerating `.dflow.yaml`: the preflight is independent of it.
`CheckOrCreateBranch` and `cmd/gitutils/git.go` are untouched.

Files changed: `cmd/commands/init.go`.

### WU2 (test seams) — the prompt surface changed

OUTCOME: done, and it was a required change rather than a convenience. `initTerminalAnswers`
(`cmd/tests/agent_cli_test.go`) answered the publication prompt with `n`; once the prompt is not
offered without an `origin`, the driver waited for a trigger that never arrives and the run failed
on its timeout. The answer and the driver's doc comment were updated. Both consumers
(`TestInitCLIWiresAgentReferences`, `TestInitCLIUpdatesExistingClaudeMd`) pass unchanged, and no
assertion was weakened.

Files changed: `cmd/tests/agent_cli_test.go`.

### WU3 — regression coverage and the behavior's help text

OUTCOME: done. Unit tests in `cmd/commands/init_test.go` cover the commitless refusal before any
mutation, the publication skip with no `remote:`/`push:` operation, and the permissive nil-probe
seam; `recordingInitOperations()` defaults both new probes to true and records nothing, so the
four pre-existing sequence assertions keep passing **unchanged** (verified by function-body
comparison, not by inspection). Integration tests in `cmd/tests/init_guard_test.go` cover the
commitless refusal (no terminal needed, because the preflight precedes the terminal check), the
completed onboarding without an `origin` through a pseudo-terminal, and — as a triangulation case
— that the publication prompt is still offered when an `origin` exists. A separate
`initTempEmptyGitRepo` helper was added instead of changing `initTempGitRepo`, which
`TestInitFirstRunStillReachesPrompt` depends on. The `dflow init` help text in
`cmd/commands/init.go` now states the origin condition and the commit requirement.

RED before implementation, GREEN after: the unit RED was `ops.hasCommits undefined` (build
failure against the new operations); the integration RED was
`[init] is missing the exact refusal … ❌ dflow init is interactive and requires a terminal`, and
for the skip path `skip report rendered 0 times` with the transcript still showing
`? Do you want to push the base branches to 'origin'? (Y/n)`.

Files changed: `cmd/commands/init_test.go`, `cmd/tests/init_guard_test.go`,
`cmd/tests/gitutils_test.go`, `cmd/commands/init.go` (help text).

### Independent verification

OUTCOME: **verified with findings, all non-blocking.** `go build ./...`, `go vet ./...`,
`gofmt -l .` (empty) and `git diff --check` clean; `go test -count=1 ./...` green across all ten
packages (`cmd/tests 51.352s`).

Discrimination proven in an isolated `cp -R` copy with only the production behavior reverted: the
new unit tests and the two new CLI tests FAIL against the unfixed code (including the
60-second hang on the prompt that no longer exists) and PASS after restoring it. Wording verified
programmatically, not by eye: the three copies of each sentence are byte-identical (84 and 60
characters), and the superseded wording has zero occurrences anywhere.

Real-binary reproduction in temporary directories: a commitless repository exits 1 with exactly
one `❌` and the refusal sentence, prints no "requires a terminal", and creates no `.dflow.yaml`;
`--force` behaves identically; a repository with a commit and no `origin` completes through a
pseudo-terminal, writes `.dflow.yaml`, creates no remote refs and renders the skip line exactly
once; a repository with a local bare `origin` still offers the publication prompt and pushes
nothing when answered `n`.

Accepted residuals, all non-blocking and all recorded rather than hidden:

- The preflight runs twice per complete run, costing one extra read-only `git rev-parse`. This is
  the deliberate price of having the guard both before the prompts and inside the unit-testable
  apply path.
- An unwired (nil) probe is permissive — "has commits", "has an origin". It is unreachable in
  production, where `defaultInitOperations()` is the only source of operations, and it follows the
  existing permissive `generateAgentFiles` seam.
- The prompt gate and the apply guard probe `HasOriginRemote` independently, so a remote appearing
  or disappearing mid-run could desynchronize the prompt from the report. Impossible without
  concurrent remote mutation.
- Outside a Git repository the shared `EnsureGitRepo` check runs first, so the preflight cannot
  replace the more accurate `this is not a Git repository` refusal.

### WU4 — commit identity

OUTCOME: pending. Written by the follow-up `docs(odd)` commit that records this work unit's
commit id.

### Repository state checks

`git diff --stat` over the work unit: 5 files changed, 335 insertions, 14 deletions, plus this
feature document. `CHANGELOG.md` is byte-identical to `develop`.

## Native review

Pending: the candidate is this work unit's commit, and the review runs after the freeze.
