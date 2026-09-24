# Changelog

## 📦 v0.3.0 – Self-Update, Installers & CLI Contracts

### Added
- `dflow update` to self-update the binary: release discovery against GitHub releases, SHA-256 checksum verification, and atomic binary replacement. It shows the release-notes summary before downloading anything and, on an interactive terminal, asks for confirmation (`--yes` skips it); answering no aborts with exit 0 and the binary untouched, non-interactive runs keep the previous prompt-free behavior, and after a successful swap the summary and the full release URL are shown again.
- `dflow update --check` includes a "What's new" summary taken from the release body, rendered as a terminal-friendly digest (section headings marked with `▸`, list items with `•`, the body's own title dropped) and truncated with the full release URL.
- After any dflow command finishes, one line on `stderr` announces a newer published release (`A new dflow release is available: vX.Y.Z — run dflow update`). The check runs in the background with a short timeout, is cached for 24 hours in `$XDG_CACHE_HOME/dflow/update-check.json` (fallback `~/.cache/dflow/`), never touches stdout contracts (`--json`, completion, `version --revision`), is skipped for development builds, and can be disabled with `DFLOW_NO_UPDATE_CHECK=1`.
- POSIX (`scripts/install.sh`) and Windows (`scripts/install.ps1`) installer scripts, published as release assets alongside the archives.
- `dflow start <type> <name> --from <parent>` bases a work branch on another work branch (chained/stacked) instead of the configured base, with `--push`/`--no-push` for non-interactive use: passing both is an error, and a missing terminal choice fails fast before the branch is created.
- `dflow status --json` reports the current branch type and finish targets as one machine-readable document, and merge modes are validated against the configuration.
- `dflow finish --delete` removes finished branches locally and remotely after a successful finish when no manual targets remain.
- `dflow version` reports the commit the binary was installed from next to the channel/version marker, suffixed `-dirty` when the build tree had uncommitted changes; `dflow version --revision` prints the full 40-character commit hash alone, or `unknown` when the binary carries no VCS stamp; `dflow version --json` prints the marker, the revision, and the dirty flag as one machine-readable document.

### Changed
- `dflow init` now generates feature branches from `develop` with finish targets `develop` and `uat`, and release branches from `uat` with finish targets `main` and `develop`; documentation and tests were updated to reflect the new workflow defaults.
- Command output is terminal-aware: progress lines and prompts degrade gracefully on non-interactive terminals, git's own text stays out of the caller's stdout, and status icons render as chrome next to the message instead of inside it.

### Fixed
- Failed dflow commands exit with a non-zero exit code.
- `dflow start` leaves the repository as it found it when a step fails instead of stopping mid-way with a half-created branch.
- `dflow finish` no longer fails when the publish comparison against `origin` errors, and it no longer claims to have deleted a remote branch it skipped.
- `dflow delete` no longer reports a remote branch as absent when `origin` is configured but cannot be reached: the local half is still deleted and the command then says the remote could not be checked, and with no local branch it touches nothing and fails on the lookup. A repository with no `origin` configured is a known absence, not a failed check, and still deletes the local half successfully.
- `dflow delete` is now idempotent: it deletes whichever copy of the branch still exists, reports an already absent copy instead of failing, and fails when the branch exists in neither place or a copy that exists could not be deleted.
- `dflow init` now refuses to overwrite an existing `.dflow.yaml`; use `--force` to regenerate it.
- The `Makefile` no longer injects the latest release tag into development builds: a tag is used only when `HEAD` is exactly on it, and the installed commit is self-reported by the binary from its VCS stamp.

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
