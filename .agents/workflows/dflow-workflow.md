---
description: dflow usage and branch/merge workflow for agents
---

# dflow Workflow

Source of truth for how to use `dflow` in this repository. The authoritative
configuration lives in `.dflow.yaml`; this file explains how to read and apply it.

## Commands

| Command | Purpose |
| --- | --- |
| `dflow init` | Initialize `.dflow.yaml` (run once per project; refuses to overwrite an existing file, use `--force` to regenerate it) |
| `dflow start <type> <name>` | Create and switch to a work branch |
| `dflow finish` | Merge the current branch into its configured `auto` targets |
| `dflow finish --dry-run` | Preview the finish plan without merging or pushing |
| `dflow finish --dry-run --json` | Preview the finish plan as a single JSON document |
| `dflow finish --delete` | Also delete the branch when no `manual` targets remain |
| `dflow status [--json]` | Report branch, detected type, resolved targets and Git state |
| `dflow delete <branch> [--yes]` | Delete a branch locally and remotely; idempotent, it deletes whichever copy still exists |
| `dflow config set-author "Name" --email ...` | Store local `dflow.author` / `dflow.email` |
| `dflow completion [install]` | Generate shell completions |
| `dflow version` | Show the CLI version |

## Branch types

Aliases: `feat|feature`, `release`, `fix|hot|hotfix`, `bug|bugfix`.

| Type | Prefix | Base |
| --- | --- | --- |
| feature | `feature/` | `develop` |
| release | `release/` | `develop` |
| hotfix | `hotfix/` | `main` |
| bugfix | `bugfix/` | `develop` |

`uat` is an alias of `develop` in this repository, so duplicated targets are
deduplicated automatically. Names are normalized to kebab-case and must respect
Git ref rules.

## Merge mode is the decision authority

`workflow.branch_rules[].merge_mode` in `.dflow.yaml` decides whether a target
is merged directly by dflow or through a pull request:

| Target `merge_mode` | Agent action |
| --- | --- |
| `auto` | direct merge via `dflow finish` (clean tree + human confirmation). No PR. |
| `manual` | open a PR toward that target. Never use `dflow finish`. |

This repository: `develop = auto`, `main = manual`.

`auto` and `manual` are the only valid values. An unknown or missing merge mode
now makes `dflow finish` and `dflow status` fail, naming the branch and the
offending value, instead of silently skipping the target.

Consequences:

- Never suggest a PR toward `develop`; it merges directly.
- Always use a PR toward `main` (release/hotfix promotions are `manual`).
- `branch_rules` entries override `default_merge_mode` per branch; an unlisted
  branch falls back to the default.

## finish guardrails

`dflow finish` is the exception, not the default. Use it only when:

- the target is `auto`, AND
- the working tree is clean, AND
- a human explicitly confirmed a direct merge (or the repo has no `origin`).

Do not run `dflow finish` to:

- complete a `manual` target (open a PR instead), or
- finish a branch that still has stacked children (it would break their merge bases).

Default agent behavior: propose a PR. Direct merge is explicit, never assumed.

## Issue lifecycle

- The repository default branch is `develop`, so an issue closes when the work
  that addresses it lands in **`develop`** — not `main`.
- Close it with a GitHub closing keyword (`Closes #<n>`) in the commit or merge
  message. A merge **without** that keyword leaves the issue open.
- `main` and release branches do not close issues. The release promotion
  (`develop` → `main`) is recorded in the changelog, not in issues.
- Issues track any work unit (feature, bug, chore, docs), not only production
  incidents.

## Chained branches

`dflow start <type> <name> --from <parent>` creates a stacked branch from an
existing work branch instead of the configured base. Pass `--push` or
`--no-push` to publish non-interactively; passing both is an error, and a
missing terminal now fails fast before the branch is created instead of
prompting.

When a stacked parent merges, rebase each child with
`git rebase --onto <target> <old-parent> <child>` and retarget its PR. A
`dflow chain` command (list/rebase) is tracked as a follow-up.

## JSON output is a contract

`dflow status --json` and `dflow finish --dry-run --json` are contracts for
scripts and agents: stdout carries exactly one JSON document and nothing else —
no banner, no icons, no progress lines — and a failure still exits non-zero with
the reason in an `{"error": ...}` document. Parse it with a JSON parser, never by
matching substrings. `finish --json` requires `--dry-run`; a mutating finish has
no report to serialize.

When the current branch matches no configured prefix, `status --json` succeeds
with an empty `branch_type` and an empty `targets` array, which is the
machine-readable way to say "not a work branch".

## Commit conventions

See `commit-rules.md`. Conventional Commits, English, lowercase.
