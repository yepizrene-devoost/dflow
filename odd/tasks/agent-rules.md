# agent-rules — feature tracking

Branch: `feature/agent-rules` (base: develop)
Status: done (merged to develop bb6d2b1; branch deleted)

## Goal

Add explicit agent-facing rules (dflow workflow usage, relevant skills, project
memory) and align `HISTORY.md` with its release-history purpose.

## Tasks

1. [x] Create `feature/agent-rules` from develop
2. [x] Commit 1: `chore(agents) track agent instructions and issue templates`
3. [x] Write `.agents/workflows/dflow-workflow.md`
4. [x] Write `.agents/MEMORY.md`
5. [x] Update `AGENTS.md` (skills, source-of-truth, memory pointer)
6. [x] Commit 2: `docs(agents) dflow workflow + project memory`
7. [x] Rewrite `HISTORY.md` as release history
8. [x] Move kickoff narrative to `README.md` About
9. [x] Commit 3: `docs(history) align HISTORY.md`
10. [x] Report and pause before merge to develop

## Decisions

- `HISTORY.md` = release history only (no decisions, no roadmap, no per-release detail).
- `MEMORY.md` = curated current-state decisions only.
- Changelog = git-cliff (Model B), handled as a separate task.
- Relevant skills live inside `AGENTS.md`.
- `workflow.branch_rules[].merge_mode` in `.dflow.yaml` is the authority for PR vs `finish`.

## Evidence (commits)

- 2ae3844 chore(agents): track agent instructions and issue templates
- 5dcda83 docs(agents): add dflow workflow, skills, and project memory
- f0def30 docs(history): align HISTORY.md with release-history purpose
