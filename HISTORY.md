# 📘 Release History – dflow

> 🔄 Reverse chronological order — newest first.

This document is a timeline of **release versions**, including candidates
prepared before tagging. A candidate's tag link becomes available after
publication. For the detailed changes, see [`CHANGELOG.md`](./CHANGELOG.md).

---

## 📦 v0.5.1 – Safer Onboarding & Local Repositories

**Tag:** [`v0.5.1`](https://github.com/yepizrene-devoost/dflow/releases/tag/v0.5.1)<br>
**Date:** 2026-10-03

**Highlights:**

- `dflow init` refuses a repository with no commits before it asks anything, naming the missing first commit and leaving `.dflow.yaml` untouched, while `--force` keeps its single meaning: regenerating the configuration.
- Onboarding completes in a repository without an `origin` remote: base-branch publication is skipped in the wording the CLI already uses for that condition, and a question whose answer cannot be acted on is no longer asked.

---

## 📦 v0.5.0 – Repository Context & Safer Recovery

**Tag:** [`v0.5.0`](https://github.com/yepizrene-devoost/dflow/releases/tag/v0.5.0)<br>
**Date:** 2026-09-27

**Highlights:**

- Commands resolve the canonical repository from nested directories, linked worktrees, and `DFLOW_CWD`, keeping configuration access anchored to the worktree root.
- Configuration and onboarding fail before persistence when branch, flow, or merge rules are invalid, and `dflow init` preserves clear retry state after partial branch setup.
- `dflow finish` reports the exact failed phase and completed targets, cleans up conflicted merges, and attempts to restore the original work branch.
- `dflow start` preserves underlying Git diagnostics and gives an actionable recovery command when branch publication fails.
- The public Go API adds repository discovery and session types plus explicit configuration validation, without removing or renaming existing exported `pkg/*` symbols.

---

## 📦 v0.4.1 – Release Contracts & Version Display

**Tag:** [`v0.4.1`](https://github.com/yepizrene-devoost/dflow/releases/tag/v0.4.1)<br>
**Date:** 2026-09-25

**Highlights:**

- Release builds retain their `v`-prefixed tag and omit the redundant revision on clean builds.
- `dflow finish` is fully non-interactive, with `--delete` as its explicit deletion intent.
- Failed automatic merges and target pushes leave the work branch available for manual recovery.

---

## 📦 v0.4.0 – Multi-Agent Install, Legible Digests & Release Safeguards

**Tag:** [`v0.4.0`](https://github.com/yepizrene-devoost/dflow/releases/tag/v0.4.0)  
**Date:** 2026-09-25

**Highlights:**

- `dflow agent` derives a workflow document from `.dflow.yaml` and installs it as a discoverable skill for several AI coding agents.
- One generated document serves several agents: `AGENTS.md` for pi, codex and opencode, one `SKILL.md` in the project's shared `.agents/skills` root when the skill is installed locally, and `CLAUDE.md` only when Claude is named or the file already exists.
- The `dflow update` report and its "What's new" digest are sized for a terminal: one shared text measure, an airier list, and bold section headings.
- The release path fails fast on a stale changelog, a missing tag or a token-less `.env` instead of publishing an empty release body.

---

## 📦 v0.3.0 – Self-Update, Installers & CLI Contracts

**Tag:** [`v0.3.0`](https://github.com/yepizrene-devoost/dflow/releases/tag/v0.3.0)  
**Date:** 2026-09-24

**Highlights:**

- `dflow update` self-updates the binary with checksum verification, a release-notes digest, and an interactive confirmation.
- POSIX and Windows installer scripts ship as release assets.
- Terminal-aware output, machine-readable JSON contracts, and non-zero exit codes on failure.
- `dflow start --from` for chained/stacked branches with non-interactive push choices.

---

## 📦 v0.2.0 – Finish Automation, Versioning & Flow Documentation

**Tag:** [`v0.2.0`](https://github.com/yepizrene-devoost/dflow/releases/tag/v0.2.0)  
**Date:** 2026-04-02

**Highlights:**

- `dflow finish` automates the configured finish flow, with `--dry-run` and `--delete`.
- `dflow version`, `dflow --version`, and `dflow -V`.
- `.dflow.yaml` gained explicit `base` and `finish_targets` rules per branch type.

---

## 📦 v0.1.2 – Bugfix Flow, Shell Completion & Interactive Commands

**Tag:** [`v0.1.2`](https://github.com/yepizrene-devoost/dflow/releases/tag/v0.1.2)  
**Date:** 2025-07-22

**Highlights:**

- `bug` / `bugfix` branches and the new `dflow delete` command.
- Shell autocompletion (`bash`, `zsh`, `fish`, `powershell`) and `dflow completion install`.
- Spinner and emoji-based output helpers for consistent CLI messages.

---

## 📦 v0.1.1 – Documentation, GPG Signing & Metadata Enhancements

**Tag:** [`v0.1.1`](https://github.com/yepizrene-devoost/dflow/releases/tag/v0.1.1)  
**Date:** 2025-07-19

**Highlights:**

- GitHub Actions workflow for build/test on `main` and `develop`.
- Default `.dflow.yaml` generation and `.env` added to `.gitignore`.
- License migrated from **GPLv3** to **MIT**.
- Makefile targets (`build`, `build-all`, `release`, `install`, `lint`, `changelog`, `clean`).

---

## 🚀 v0.1.0 – Initial Release

**Tag:** [`v0.1.0`](https://github.com/yepizrene-devoost/dflow/releases/tag/v0.1.0)  
**Date:** 2025-07-06

**Highlights:**

- `dflow init`: interactive setup of flow and merge rules.
- `dflow start`: branching for features, releases, hotfixes.
- `dflow config`: set/get local author info via Git config.
- Signed Git tag with GPG key and cross-platform binary distribution.
