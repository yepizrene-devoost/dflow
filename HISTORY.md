# 📘 Project History – dflow

`dflow` is a Git CLI tool designed to simplify branching workflows with conventions and interactive tooling. This document tracks major milestones, decisions, and feature developments over time.

---

## 🛠️ Project Kickoff

**Date:** 2025-07-01  
**Summary:**  
The project was created to formalize Devoost's internal Git workflows using a CLI. Goals included reproducibility, ease of onboarding, and better control over merges and release cycles.

---

## 🧱 Architecture & Tooling

**Decisions:**
- `cobra` selected for CLI structure.
- `survey` used for interactive prompts.
- Configuration stored in `.dflow.yaml`, auto-generated with a banner.
- Commands organized under `cmd/commands/` with helpers in `utils/` and `gitutils/`.

---

## 🚀 v0.1.0 – Initial Release

**Tag:** [`v0.1.0`](https://github.com/yepizrene-devoost/dflow/releases/tag/v0.1.0)  
**Date:** 2025-07-06  
**Highlights:**
- `dflow init`: interactive setup of flow and merge rules.
- `dflow start`: branching for features, releases, hotfixes.
- `dflow config`: set/get local author info via Git config.
- Signed Git tag with GPG key.
- Binary distribution for macOS, Linux, and Windows (`dist/` via Makefile).
- Release uploaded to GitHub via `gh release`.

---

## 🔮 Future Plans

- Add `dflow finish` to complete branches and perform merges automatically.
- Automate changelog generation from Git history.
- Add tests for core Git integrations.
- Improve `config` options to support defaults or shared templates.

---

_Authored and maintained by Rene Yepiz — rene@devoost.com_  
