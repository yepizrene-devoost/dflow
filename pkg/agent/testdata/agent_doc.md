---
name: dflow
description: "Trigger: dflow start, dflow finish, branch workflow, merge rules, finish targets, dflow status, dflow delete. Project's dflow configuration and branch workflow."
metadata:
  source: .dflow.yaml
  generated_by: dflow agent
---

# dflow Workflow

<!-- Generated from .dflow.yaml — regenerate with: dflow agent -->

## Branch Types

| Type | Aliases | Prefix | Base | Finish Targets |
|---|---|---|---|---|
| feature | feat | feature/ | develop | develop |
| release | — | release/ | develop | main, develop |
| hotfix | hot, fix | hotfix/ | main | main, develop |
| bugfix | bug | bugfix/ | develop | develop |

## Merge Rules

| Branch | Mode | Behavior |
|---|---|---|
| develop | auto | Direct merge via `dflow finish` |
| main | manual | Open a Pull Request |

Default mode: manual

## Commands

| Command | Purpose |
| --- | --- |
| `dflow init` | Initialize `.dflow.yaml` (run once per project; refuses to overwrite an existing file, use `--force` to regenerate it) |
| `dflow agent` | Generate an agent workflow file from `.dflow.yaml` |
| `dflow start <type> <name>` | Create and switch to a work branch |
| `dflow finish` | Merge the current branch into its configured `auto` targets |
| `dflow finish --dry-run` | Preview the finish plan without merging or pushing |
| `dflow finish --dry-run --json` | Preview the finish plan as a single JSON document |
| `dflow finish --delete` | Also delete the branch when no `manual` targets remain |
| `dflow finish --no-push` | Merge without publishing the work branch |
| `dflow status [--json]` | Report branch, detected type, resolved targets and Git state |
| `dflow delete <branch> [--yes]` | Delete a branch locally and remotely; idempotent |
| `dflow update [--check] [--force] [--yes] [--json]` | Update the dflow binary to the latest release |
| `dflow config set-author "Name" --email ...` | Store local `dflow.author` / `dflow.email` |
| `dflow completion [install]` | Generate shell completions |
| `dflow version` | Show the CLI version |

`dflow finish` publishes the work branch to `origin` before merging, so a
`manual` target's PR can be opened right after with no manual `git push`;
`--no-push` keeps it local.

## Merge mode is the decision authority

`workflow.branch_rules[].merge_mode` in `.dflow.yaml` decides whether a target
is merged directly by dflow or through a pull request:

| Target merge_mode | Agent action |
|---|---|
| `auto` | direct merge via `dflow finish` (clean tree; fully non-interactive). No PR. |
| `manual` | open a PR toward that target. Never use `dflow finish`. |

This repository: manual default. `develop = auto`. `main = manual`.

`auto` and `manual` are the only valid values. An unknown or missing merge mode
makes `dflow finish` and `dflow status` fail, naming the branch and the
offending value, instead of silently skipping the target.

Consequences:

- Never suggest a PR toward an `auto` target; it merges directly.
- Always use a PR toward a `manual` target (release/hotfix promotions).
- `branch_rules` entries override `default_merge_mode` per branch; an unlisted
  branch falls back to the default.

## finish guardrails

`dflow finish` is the exception, not the default. Use it only when:

- the target is `auto`, AND
- the working tree is clean, AND
- direct merge is explicitly requested by running `dflow finish` (or the repo has no `origin`).

Do not run `dflow finish` to:

- complete a `manual` target (open a PR instead), or
- finish a branch that still has stacked children (it would break their merge bases).

Default agent behavior: propose a PR. Direct merge is explicit, never assumed. `dflow finish`
is fully non-interactive; `--delete` is the complete confirmation and never prompts.
The work branch is deleted only after every automatic target merge and push succeeds,
never when manual targets remain, and never after a merge, conflict, non-fast-forward,
or target-push failure. There is no automatic rollback or merge abort.

## Issue lifecycle

- `.dflow.yaml` defines configured branch names, bases, finish targets, and merge
  modes. It does not declare the forge's default branch.
- Do not infer issue auto-closing behavior from configured integration targets.
  Verify the forge default separately before predicting how a closing keyword behaves.
- If work remains after a merge, use a non-closing reference such as
  `References #<n>` instead of `Closes`, `Fixes`, or `Resolves`.
- Issues track any work unit (feature, bug, chore, docs), not only production incidents.

### Closing an issue

Close an issue only after all tracked scope is complete and a human explicitly
confirms that standalone action:

- **Finish state** — verify that no implementation, merge, tagging, publication,
  or other tracked delivery work remains.
- **Labels** — remove every `status:` label naming a state that has ended.
  Keep the issue's `type:` label.
- **Closing comment** — name the relevant delivery commit and review lineage when used.

## Chained branches

`dflow start <type> <name> --from <parent>` creates a stacked branch.

When a stacked parent merges, rebase each child with
`git rebase --onto <target> <old-parent> <child>` and retarget its PR.

## JSON output is a contract

`dflow status --json`, `dflow finish --dry-run --json` and `dflow update --json`
are contracts: stdout carries exactly one JSON document and nothing else.
A failure exits non-zero with `{"error": ...}`. Parse with a JSON parser.

## Commit conventions

- **Workflow**: Always use `dflow` to manage branches.
- **Format**: `type(scope): description` — scope is optional.
- **Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`.
- **Rules**: English, lowercase, no period, no ticket IDs, no file paths.
