# Changelog

## 📦 v0.2.0 – Finish Automation, Versioning & Flow Documentation

### Added
- `dflow finish` command to resolve configured finish targets, merge `auto` targets, and report `manual` targets.
- `dflow finish --dry-run` to preview the finish plan without modifying branches.
- `dflow version`, `dflow --version`, and `dflow -V`.
- Workflow helpers and Git finish helpers with test coverage for finish planning and branch synchronization.

### Changed
- `.dflow.yaml` now supports explicit `flow.<type>.base` and `flow.<type>.finish_targets` rules.
- `workflow.branch_rules` now supports explicit branch objects with `merge_mode`.
- `dflow init` now generates flow defaults for a customizable adjusted Git workflow.
- CLI help and documentation were refreshed to match the current command set and configuration model.

### Internal
- Added integration tests for `dflow finish` and expanded Git helper coverage.
- Updated exported GoDoc comments for the next Go Reference release.
