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

1. [ ] WU1 — discovery registry + reference writer
       (`pkg/agent/registry.go`, `registry_test.go`, `reference.go`,
       `reference_test.go`): the four agents with their instruction files and
       skill directories, and an idempotent, marker-based instruction-reference
       writer. Strict TDD: failing tests first.
2. [ ] WU2 — CLI reference wiring (`cmd/commands/agent.go` default path,
       `cmd/commands/init.go` rewired onto the shared writer,
       `cmd/tests/agent_cli_test.go`): `--agents` selection; `AGENTS.md` always
       ensured; `CLAUDE.md` only when present or explicitly requested. `init`
       stops carrying its own private copy of the reference logic.
3. [ ] WU3 — skill install (`pkg/agent/install.go`, `install_test.go`,
       `cmd/commands/agent.go`): `--install` and `--local`, per-agent skill
       placement, idempotent rewrite, `--json` reporting of every written path.
4. [ ] WU4 — docs and verification: README section for the multi-agent wiring
       and the discovery table; `go test ./...`, `golangci-lint run ./...`, and a
       manual smoke run of each new flag combination.
5. [ ] Commit identity recorded.

## Out of scope

- No change to the generated workflow template or its golden file: only the
  placement of those bytes changes, not their content.
- No skill install from `dflow init` (it would write into `$HOME` during
  initialization); `init` wires instruction references only.
- No MCP server, no config-driven agent list in `.dflow.yaml`.

## Discovered during this work

- `odd/tasks/agent-workflow-hydration.md` (issue #31, merged `bb6d2b1`) still
  shows every task unchecked although the work shipped. Reported to the
  maintainer; not silently rewritten here.

## Commit identity

- Pending.
