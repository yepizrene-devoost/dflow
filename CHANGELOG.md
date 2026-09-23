# Changelog

## Unreleased

### Changed
- `dflow init` now generates feature branches from `develop` with finish targets `develop` and `uat`.
- `dflow init` now generates release branches from `uat` with finish targets `main` and `develop`.
- Documentation and tests were updated to reflect the new workflow defaults.

### Added
- `dflow finish` now publishes the work branch to `origin` before merging, so a `manual` target can be promoted through a pull request without a manual `git push`; publishing is idempotent (a branch `origin` already holds at the same commit is left untouched) and `dflow finish --no-push` keeps the branch local.
- `dflow finish --delete` to remove finished branches locally and remotely after a successful finish when no manual targets remain.
- `dflow version` now reports the commit the binary was installed from next to the channel/version marker, suffixed `-dirty` when the build tree had uncommitted changes.
- `dflow version --revision` to print the full 40-character commit hash alone, or `unknown` when the binary carries no VCS stamp.
- `dflow version --json` to print the marker, the full revision, and the dirty flag as one machine-readable document.

### Fixed
- `dflow delete` no longer reports a remote branch as absent when `origin` is configured but cannot be reached: the local half is still deleted and the command then says the remote could not be checked, and with no local branch it touches nothing and fails on the lookup. A repository with no `origin` configured is a known absence, not a failed check, and still deletes the local half successfully.
- The `Makefile` no longer injects the latest release tag into development builds: a tag is used only when `HEAD` is exactly on it, and the installed commit is self-reported by the binary from its VCS stamp.
- `dflow init` now refuses to overwrite an existing `.dflow.yaml`; use `--force` to regenerate it.
- `dflow delete` is now idempotent: it deletes whichever copy of the branch still exists, reports an already absent copy instead of failing, and fails when the branch exists in neither place or a copy that exists could not be deleted.

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
