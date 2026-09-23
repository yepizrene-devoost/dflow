# AGENTS.md

Repository instructions for coding agents working in this project.

## Commit Messages

- Follow the commit convention defined in `.agents/workflows/commit-rules.md`.
- Preferred format: `type(scope): description`
- Scope is optional when it does not add clarity: `type: description`
- Example: `feat(start): support multi-word branch names`
- Always write commit messages in english.
- Use conventional commit types only: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`.
- Keep the description concise and lowercase.
- Do not invent a ticket prefix when the branch name and recent repository history do not use one.
- Do not include full or relative file paths in commit messages.

## Branch Workflow

- Use `dflow` to manage branches, following `.dflow.yaml` and `.agents/workflows/dflow-workflow.md`.
- For `feat` branches, use `develop` as the base branch.
- For `release` branches, use `develop` as the base branch.
- For `bugfix` branches, use `uat` as the base branch.
- For `hotfix` branches, use `main` as the base branch.
- `develop` merges directly (`auto`); `main` requires a PR (`manual`). Never suggest a PR toward an `auto` target.

## Relevant Skills

When available, load these skills while working in this project:

| Skill | Applies to |
| --- | --- |
| `gentle-ai` | harness discipline (clarify, TDD, delegate, protect review) |
| `gentle-ai-work-unit-commits` | reviewable work-unit commits |
| `gentle-ai-chained-pr` | stacked/chained PRs (slices under 400 lines) |
| `gentle-ai-branch-pr` | creating PRs with issue-first checks |
| `gentle-ai-issue-creation` | issue creation and triage |
| `gentle-ai-cognitive-doc-design` | docs with low cognitive load |
| `gentle-ai-comment-writer` | PR/issue comments |

Optional: `gentle-ai-judgment-day` (adversarial dual review, critical work only)
and `gentle-ai-rdd-defect-workflow` (only under the RDD switch). Resolve exact
skill paths from the local skill registry.

## Project Memory

- Read `.agents/MEMORY.md` before making consequential changes; it holds the
  current-state decisions and architecture context. Keep it short and current
  (it is not a log).

## Task Tracking (odd/)

- The ODD feature checklist lives in `odd/tasks/<feature>.md` and is committed to the repo.
- Tick a task `[x]` as soon as it finishes, and sync the Engram mirror and the
  visible todo list — a finished task is never left unmarked.
- Close every stage before staging a commit: no stage may stay open inside the commit that
  contains its work, and a stage that cannot be closed is confirmed before staging. The
  ordering rule and its single exception (a short follow-up `docs(odd)` commit that records the
  work-unit commit SHAs) are owned by the `## ODD: close every stage before committing` section
  of the global `~/.pi/agent/AGENTS.md`; keep this section aligned with it instead of restating it.

## Source Of Truth

- Treat `.agents/workflows/commit-rules.md` as the source of truth for commit-message conventions.
- Treat `.agents/workflows/dflow-workflow.md` as the source of truth for dflow usage, merge/PR rules and issue closure.
- If these rules are updated there, keep this file aligned with them.
