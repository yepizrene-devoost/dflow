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

- Use `dflow` to manage branches, following `.dflow.yaml`.
- For `feat` branches, use `develop` as the base branch.
- For `release` branches, use `develop` as the base branch.
- For `bugfix` branches, use `uat` as the base branch.
- For `hotfix` branches, use `main` as the base branch.

## Source Of Truth

- Treat `.agents/workflows/commit-rules.md` as the source of truth for commit-message conventions.
- If these rules are updated there, keep this file aligned with them.
