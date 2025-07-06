# Changelog

## v0.1.0 - Initial release

### ✨ Features
- Added `dflow init` command:
  - Interactive setup for `.dflow.yaml` with branch names, prefixes, merge mode, and exceptions.
  - Generates configuration with ASCII banner and saves to root.
  - Ensures base branches exist and prompts to push them.

- Added `dflow start` command:
  - Supports creating `feature`, `release`, and `hotfix` branches.
  - Determines base branch and prefix from `.dflow.yaml`.
  - Checks out, pulls base branch, creates new branch, and optionally pushes.

- Added `dflow config` command group:
  - `set-author`: prompts for or accepts name/email and saves in local Git config.
  - `get-author`: displays author/email from config.
  - `list`: lists all local `dflow.*` Git config entries.

### 🧠 Internals & Workflow
- Introduced `.dflow.yaml` format with support for:
  - Branch naming and prefix configuration.
  - Flow definitions (feature base/merge, release base, hotfix base).
  - Workflow merge mode control (`auto` vs `manual`) and per-branch exceptions.
- Added `validators` middleware to ensure CLI commands are run in Git repos and initialized projects.
- Restructured commands under `cmd/commands` and logic into reusable `utils` and `gitutils` packages.

### 🧪 Testing
- Added unit test: `TestSaveAndLoadConfig` using temp dir and `DFLOW_CWD`.

### 💄 UX
- ASCII banners for CLI startup and config headers.
- Interactive prompts via `survey` for all commands.
- Graceful Ctrl+C handling with custom message.
- Refined CLI help for each command group.

---

Category: **features**  
Date: 2025-07-06  
Author: Rene Yepiz **(rene@devoost.com)**
