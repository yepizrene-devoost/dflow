---
description: Rules for commit message generation
---

This file contains guidelines for the AI agent (Antigravity) to follow the project's conventions.

## 📌 Git Conventions (dflow)

-   **Workflow**: Always use `dflow` to manage branches.
-   **Naming**: Branches must follow the prefixes defined in `.dflow.yaml`.
-   **Bases**:
    -   When creating a `feat`, base it on `develop`.
    -   When creating a `release`, base it on `develop`.
    -   When creating a `bugfix`, base it on `uat`.
    -   When creating a `hotfix`, base it on `main`.

## 📝 Commit Conventions

-   **Observed repo format**: `type(scope): description`
-   **Scope is optional**: `type: description` is also valid when no scope adds value.
-   **Examples**:
    -   `feat(start): add support for starting bugfix branches`
    -   `docs: update README with bugfix support and config file improvements`
    -   `fix(cli): suppress banner during shell autocompletion`
-   **Rules**:
    -   Always write messages and comments in english.
    -   Use conventional commit types (`feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`).
    -   Prefer `type(scope): description` when the scope clarifies the change.
    -   Use `type: description` when the change spans multiple areas or no single scope is useful.
    -   Keep descriptions lowercase, concise, and action-oriented.
    -   Avoid ending the subject with a period.
    -   Do not include ticket IDs unless the repository convention changes and the branch naming actually requires them.
    -   **Important**: Do not include full or relative directory/file paths in the commit message body (e.g. `app/Http/...`). Mention only the class name or exact file name if absolutely necessary.

## 🤖 Instructions for the Agent

1. **Before any repository write**: Check the current branch and use `dflow start ...` to create and switch to the agent's own work branch before creating or modifying any file, including instruction, planning, and tracking files.
2. **Base branches**: Treat `develop`, `uat`, and `main` as integration targets; do not create direct authoring commits on them.
3. **Finish exception**: A separately authorized `dflow finish` may merge a completed work branch into configured `auto` targets. This integration merge is not direct authoring on the base branch.
4. **Commit messages**: Follow the repository's observed conventional-commit style rather than inventing a ticket prefix that is not present in the branch name or recent history.
5. **Persistence**: Keep this file updated if `.dflow.yaml` rules change.
