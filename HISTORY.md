# 📘 Release History – dflow

> 🔄 Reverse chronological order — newest first.

This document is a timeline of **tagged releases**. For the detailed list of
changes in each release, see [`CHANGELOG.md`](./CHANGELOG.md).

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
