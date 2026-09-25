# multi-agent-discovery

Multi-agent skill discovery and cross-agent workflow wiring.

## Issue

Closes #34

## Goal

Make the workflow document `dflow agent` already generates reachable by every
supported AI coding agent, without hand-maintaining one discovery file per
agent and without encoding a discovery table that is factually wrong.

The document is one template; only the placement differs. That is the whole
abstraction: one generator, several discovery adapters.

## Design decisions (resolved 2026-09-25)

The issue's discovery table is not the table the agents actually implement.
Verified against primary sources before writing code:

| Agent | Instructions | Skill directory |
| --- | --- | --- |
| Pi | `AGENTS.md` (also `CLAUDE.md`, `AGENTS.override.md`) | `.agents/skills/`, `.pi/skills/`, `~/.agents/skills/`, `~/.pi/agent/skills/` |
| Codex | `AGENTS.md` (global `~/.codex/AGENTS.md`) | `.agents/skills/`, `~/.codex/skills/` |
| opencode | `AGENTS.md` only, V2 has no `CLAUDE.md` fallback | `.agents/skills/`, `.opencode/skills/`, `~/.config/opencode/skills/` |
| Claude Code | `CLAUDE.md` (`.claude/CLAUDE.md`, `CLAUDE.local.md`) | `.claude/skills/`, `~/.claude/skills/` |

Evidence:

- Codex instructions: <https://developers.openai.com/codex/guides/agents-md>;
  skills: <https://developers.openai.com/codex/skills> ("Codex reads skills from
  repository, user, admin, and system locations. For repositories, Codex scans
  `.agents/skills`"). `CODEX.md` is not a convention; extra filenames come only
  from the user-configured `project_doc_fallback_filenames`.
- opencode instructions: <https://opencode.ai/v2/docs/instructions> (V2
  recognizes only `AGENTS.md`); skills: <https://opencode.ai/docs/skills>, with
  `EXTERNAL_DIRS = [".claude", ".agents"]` in
  `packages/opencode/src/skill/skill.ts`.
- Pi: local `docs/skills.md` (`~/.agents/skills/`, `.agents/skills/`) and
  `docs/configuration.md` (`.pi/skills/`, `<agent-dir>/skills/`).
- Claude Code: <https://code.claude.com/docs/en/skills>.

**Decision 1 — corrected registry.** Build on the verified table, not on the
issue's literal one. `.agents/skills/dflow/SKILL.md` is a single file that Pi,
Codex and opencode all discover; `AGENTS.md` is a single reference that serves
the same three. Claude Code is the only outlier on both axes.

**Decision 2 — `CLAUDE.md` only when it already exists.** dflow refreshes the
reference in `CLAUDE.md` when the project already has one, and never creates it,
so no dflow run drops a surprise file into a repository that does not use Claude
Code. `dflow agent --agents claude` creates and wires it on request.

## Tasks

1. [x] WU1 — discovery registry + reference writer
       (`pkg/agent/registry.go`, `registry_test.go`, `reference.go`,
       `reference_test.go`): the four agents with their instruction files and
       skill directories, and an idempotent, marker-based instruction-reference
       writer. Tests first — RED observed as a build failure on the new symbols
       (the behavioral assertions cannot execute before the API exists), then 33
       tests green after two behavioral fixes (one test-helper bug, one
       whitespace-only spec wrongly treated as auto). Parent verification: exact
       scope (4 new files, nothing else), `go test ./...` green on the parent's
       own run, `gofmt` and `go vet` clean, tests read line by line and found to
       assert exact bytes rather than restating the implementation.

### WU1 correction pass (defect found in parent review)

`ParseAgentSpec("")` returned `nil, nil` for auto mode. The natural CLI
composition — `agents, _ := ParseAgentSpec(spec)` then
`PlanInstructionTargets(agents, exists, explicit)` — would then have built an
empty selection: no readers on any target and `CLAUDE.md` excluded even when it
exists. Auto mode would have silently lost the registry's whole point. No test
covered it. Fixed in the same work unit, together with two refinements:

- `ParseAgentSpec("")` now returns `Agents()`, so auto vs named is expressed by
the caller's `explicit` flag and the two functions compose totally.
- `InstructionTarget.Agents` is now a registry fact (every registry agent that
reads the file), never a projection of the selection, so `--json` reporting can
name the agents a write serves and the list is never empty.
- `CLAUDE.md` is created only when claude is named: the gate is the named
agent's PRIMARY instruction file (the first entry of its `Instructions`), not
any file it reads. `--agents claude` creates `CLAUDE.md`; `--agents pi` does not,
even though Pi also reads it. This is decision 2, enforced in the code.
- `EnsureInstructionReference` reports `changed == false` on every error path
(it claimed `true` alongside a failed create).

Discriminating-test check by the parent: subtests `b`, `c` and `d` of
`TestPlanInstructionTargetsSelectionCases` and the auto case of
`TestParseAgentSpec` cannot pass against the pre-correction rules, so the
correction is genuinely pinned rather than merely re-described. The correction
pass did not itself run tests-first; that discipline gap is recorded here rather
than smoothed over.

Known limitations accepted deliberately:

- A project already carrying the unmarked `## dflow Workflow` block written by
issue #31 keeps it: rule 6 is a no-op, so it never gains a marker and never gets
a path update. Not duplicating sections outweighs migrating them.
- A hand-mangled marker pair (one marker deleted) is a silent no-op, because
dflow never guesses a block boundary. `changed == false` cannot distinguish
"already referenced" from "malformed", so the CLI cannot warn about it either.
- `InstructionTarget.Agents` is registry-derived while the PRIMARY gate reads the
caller-supplied agent value. The two agree for every selection the CLI can
produce (both come from `ParseAgentSpec`), and diverge only for synthetic agent
values no caller constructs.
The create-failure path added by the correction has no direct test: reaching it
requires a permission failure, since `os.ReadFile` on a path with a regular-file
parent returns `ENOTDIR`, which `os.IsNotExist` does not classify as not-exist.
Verified by inspection.
2. [x] WU2 — CLI reference wiring (`cmd/commands/agent.go` default path,
       `cmd/commands/init.go` rewired onto the shared writer,
       `cmd/tests/agent_cli_test.go`): `--agents` selection; `AGENTS.md` always
       ensured; `CLAUDE.md` only when present or explicitly requested. `init`
       stops carrying its own private copy of the reference logic.
       Tests first — all ten pins observed failing before any source write, then
       green; three triangulation subtests added after green (their pre-
       implementation failures were the already-observed `unknown flag: --agents`
       and are recorded as such rather than claimed as a second RED).
       Parent verification: exact scope (two modified files, one new test file),
       `pkg/agent` untouched, `gofmt`/`go vet` clean, the focused pins green on
       the parent's own runs plus a `-count=6` stability probe, and the whole
       suite run on LINUX in a container — see below.

### WU2 decisions and the interactive-test cost

- **`--json` is now strictly read-only**, and the `--force` existence check moved
  under the write path. Before this, `dflow agent --json` failed with
  "already exists" whenever the document was present: a read-only render mode
  refusing to render. The JSON grew a `references` array (`file`, `agents`,
  `create`) and deliberately carries no `changed` field, because whether a
  reference file would change cannot be known without writing it.
- **`explicit := spec != ""`, not `Changed("agents")`.** The flag's own contract
  says `""` is auto, so an explicit `--agents ""` must mean what an absent flag
  means; under `Changed` it would have become a named "every agent" selection and
  created a `CLAUDE.md` nobody asked for. Pinned by a subtest.
- **`init` prints "<file> already references the dflow workflow"** when the
  reference is already current, instead of today's unconditional "Updated". The
  changed case (a normal first run) is byte-identical to the old wording; only
  the no-op case stopped claiming a write it did not make.
- **The two `init` pins need a pseudo-terminal**, because `dflow init` refuses to
  run without a terminal and its survey blocks on cursor-position queries no
  `script(1)` answers by itself. This is the first pty driver in the repository,
  and it is the one part of WU2 with a real maintenance cost: the answers are
  triggered by prompt text (rewording a survey prompt breaks the pins), the host
  must provide `script(1)` or the pins `t.Skip` silently, and replies are paced
  50 ms apart because survey's cursor reader discards a second report that lands
  in the same read. It is reactive rather than time-driven, and bounded by a
  60 s timeout that fails with the captured transcript instead of hanging.

### WU2 Linux verification (the announced risk, closed with evidence)

The first handoff declared the util-linux `script` invocation UNVERIFIED, and CI
runs on `ubuntu-latest`, so the unverified branch would have been exercised for
the first time in CI. Verified instead by running it on Linux in a container
(`golang:1.21-bookworm`, util-linux `script 2.38.1`):

- `go test ./cmd/tests/ -run TestInitCLI -count=2 -v` — both pins PASS on both
  runs, no skip, `ok ... 6.284s`.
- `go test ./... -count=1` — every package green on Linux, `cmd/tests` in 61.3 s.

The remaining accepted costs are the ones listed above (prompt-wording coupling,
the `t.Skip` coverage gap on hosts without `script(1)`, and the 50 ms reply gap);
none of them is a delivery risk for this repository's CI any more.
3. [x] WU3 — skill install (`pkg/agent/install.go`, `install_test.go`,
       `cmd/commands/agent.go`, `cmd/tests/agent_cli_test.go`): `--install` and
       `--local`, per-agent skill placement, idempotent rewrite, `--json`
       reporting of every planned path. Tests first — RED observed first as a
       build failure on the new symbols, then behaviourally with all twelve new
       or modified CLI pins failing on `unknown flag: --install`. Parent
       verification: exact scope, focused and full suites green on the parent's
       own runs, `gofmt` and `go vet` clean, `install.go` read line by line.

   WU3 decisions:

   - **Auto (`--agents` absent) targets the PORTABLE root only**, resolved as
     `LookupAgent(AgentPi)` rather than hardcoded paths: `~/.agents/skills` in
     user scope, `.agents/skills` with `--local`. This is the path issue #34
     names for a plain `--install`, and one file serving several agents is the
     point of the work unit. Two alternatives were rejected: installing into
     every non-claude agent's private root writes three identical copies that
     can drift, and writing `~/.claude` on an unqualified run is the surprise
     file decision 2 forbids. `--agents all` widens to every registry root
     (four in user scope, two in project scope).
   - That settles one principle for both discovery surfaces: **auto targets the
     portable artifact** — `AGENTS.md` on the instruction side, the shared
     `.agents/skills` root on the skill side — and naming agents widens it.
   - `SKILL.md` is wholly dflow's artifact, so it is rewritten whenever its bytes
     differ and needs no `--force`. The workflow document keeps the opposite rule
     because a team may have hand-edited it. The two may therefore differ;
     deliberately.
   - The install write is atomic (temporary file beside the target, then one
     rename), so an interrupted run cannot leave a truncated `SKILL.md`.
   - `SkillFilePath` is the single "~" resolver, shared by the writer and the
     read-only `--json` reporter so the two cannot drift apart.

   ### WU3 correction pass (defect the parent refused to ship)

   The handoff recorded this as an accepted interaction; it was not acceptable.
   `dflow agent --install` exited non-zero when the document already existed and
   `--force` was absent — and every project that ever ran `dflow init` has that
   document, so the only way to install the skill was the destructive flag whose
   entire purpose is overwriting a hand-edited document. The flag coerced users
   toward the one destructive path in the command. Now, with `--install`, an
   existing document and no `--force`, the run leaves the document
   byte-identical, says so truthfully through `utils.Info` instead of claiming it
   generated the document, and still wires the references and installs the skill
   from freshly rendered bytes. Without `--install` the guard is unchanged, and
   `--force` still regenerates. The flipped pin plants a hand-edited sentinel
   after the first install and asserts all of it, including the absence of the
   "Generated agent workflow" line.

4. [ ] WU5 — advisory follow-ups from the WU2 review, added after that review:
       the `R1-nonatomic-instruction-overwrite` WARNING (`pkg/agent/reference.go`
       still truncates in place and must adopt the atomic writer WU3 added) plus
       the five readability suggestions. Parent decision: fixed inside this
       branch rather than deferred to an issue.
5. [ ] WU4 — docs and verification: README section for the multi-agent wiring,
       the discovery table and the install flow; `go test ./...`,
       `golangci-lint run ./...`, and a smoke run of each new flag combination.
       Runs last so the docs describe the final state.
6. [ ] Commit identity recorded.

## Work-unit commit identity (WU1)

- Review declaration: this commit is the frozen candidate for the native RDD
  review of work unit 1; the expected outcome is an ordinary review over the
  diff against `2944582` (develop at branch time).
- Commit SHA: `5812032` feat(agent): add the multi-agent discovery registry and
  reference writer.
- Review outcome: APPROVED — lineage `review-90b610c3e69e4f91`, tier `medium`,
  one consolidated lens (`review-reliability`), 1267 changed lines against a
  correction budget of 200 (unused; the auto-mode defect was found by parent
  verification before the review started and fixed inside the same candidate).
  Authority burned (consumed revision
  `sha256:2ad8ca070b871a44b01602496c92979443ba4fc3a8d1bebb7bc6b9d942ab416a`).
  Neither envelope the provider returned carried findings or advisories; no
  stronger claim about their absence is available.

## Work-unit commit identity (WU2)

- Review declaration: this commit is the frozen candidate for the native RDD
  review of work unit 2; the expected outcome is an ordinary review over the
  diff against `2944582` (develop at branch time), which this time also carries
  the WU1 identity record above.
- Commit SHA: `3785b23` feat(agent): wire instruction references and add
  --agents selection.
- Review outcome: APPROVED — lineage `review-e0e412ba263a2faa`, tier `high`
  (raised by the report's own risk evidence: "code that starts other processes in
  cmd/tests/agent_cli_test.go", i.e. the pty driver), 4/4 lenses (`review-risk`,
  `review-resilience`, `review-readability`, `review-reliability`), 2145 changed
  lines against a correction budget of 200 (unused). Authority burned (consumed
  revision
  `sha256:aac96a682112438b3e4072fdadce0637955f6364132615181ce93fb976a33b37`).
- Non-blocking advisories, all `informational`, carried into WU5:
  `R1-heading-false-positive-noop` (risk), `R1-nonatomic-instruction-overwrite`
  (risk, **WARNING**), `R2-append-adjacency` (readability),
  `R2-json-agents-field` (readability), `R2-unused-registry-fields`
  (readability), `R2-updated-wording` (readability).
- Consent note: the first START for this candidate returned an unresolved
  `consent/v3` envelope whose binding expired unanswered; per policy the
  envelope was not resent, a fresh START was issued, and the subsequent run
  proceeded with four lenses.

## Work-unit commit identity (WU3)

- Review declaration: this commit is the frozen candidate for the native RDD
  review of work unit 3; the expected outcome is an ordinary review over the
  diff against `2944582` (develop at branch time).
- Commit SHA and review outcome: recorded in the next work-unit commit, never
  pre-written here.

## Out of scope

- No change to the generated workflow template or its golden file: only the
  placement of those bytes changes, not their content.
- No skill install from `dflow init` (it would write into `$HOME` during
  initialization); `init` wires instruction references only.
- No MCP server, no config-driven agent list in `.dflow.yaml`.

## Discovered during this work

- The WU3 stability probe found a real test-harness defect: redirecting `HOME`
  (needed so an install pin cannot write into the developer's real skills
  directories) also changed how the shared `go build` in
  `cmd/tests/clibuild_test.go` resolves its module cache, because with no
  `GOMODCACHE` set that cache is `$GOPATH/pkg/mod` under the process's HOME. The
  build then downloaded modules into the throwaway home, and since a module cache
  is read-only, `t.TempDir`'s own cleanup failed with `permission denied` —
  turning a green pin into a cleanup error. Fixed the same way issue #33 fixed
  the compiler cache: capture the host's `GOMODCACHE` at package initialization
  and pin it in the redirected environment. Any future pin that redirects HOME
  must do the same.
- `odd/tasks/agent-workflow-hydration.md` (issue #31, merged `bb6d2b1`) still
  shows every task unchecked although the work shipped. Reported to the
  maintainer; not silently rewritten here.

## Commit identity

- Pending.
