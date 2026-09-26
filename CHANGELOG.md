# Changelog

## 📦 v0.4.1 – Release Contracts & Version Display

### Changed
- Release builds now retain the `v`-prefixed tag in their version marker and show that marker alone on a clean build, while dirty builds keep the short revision and `-dirty` suffix for provenance. Snapshot builds continue to include their revision.
- `dflow finish` is fully non-interactive: passing `--delete` is the complete confirmation, and the work branch is removed only after every automatic target merge and push succeeds, with no manual targets remaining.

### Fixed
- When an automatic target merge, conflict, non-fast-forward, or target push fails, `dflow finish` leaves the work branch available for manual recovery instead of deleting it or automatically rolling back the merge.

## 📦 v0.4.0 – Multi-Agent Install, Legible Digests & Release Safeguards

### Added
- `dflow agent` generates the workflow document an AI coding agent reads from the project's own `.dflow.yaml`: the configured branch types, their prefixes, bases and finish targets, the per-branch merge modes, and the rules an agent must follow in the repository. The document is derived from the configuration and regenerating it is idempotent, so it cannot drift from the rules dflow enforces. It is written to `.agents/workflows/dflow.md` by default, `--path` chooses another location, an existing file is never overwritten unless `--force` is passed, and `--json` reports the target path, the rendered document, the reference plan and the skill plan as one machine-readable document without writing anything. `dflow init` offers to generate the document and wire it up as part of setup.
- `dflow agent` discovers the instruction files and skill directories of several AI coding agents through a per-agent registry, and wires the generated document into the instruction files of the agents it selects: under the default selection `AGENTS.md` — read by pi, codex and opencode — is ensured and created when missing, while `CLAUDE.md` is only kept current when the project already has one, so a default run never drops a surprise `CLAUDE.md` into a repository that does not use Claude Code. `--agents` names the agents yourself, either as a comma-separated id list or as `all`, and naming them hands the choice to you: naming `claude` creates `CLAUDE.md` and leaves `AGENTS.md` alone, because the file a named agent's instructions canonically live in is the one that naming authorizes creating.
- `dflow agent --install` installs the same rendered document as a discoverable agent skill, written as `SKILL.md` in the skills directory each selected agent reads: `~/.agents/skills/dflow/SKILL.md` under the default selection, and one directory per agent when you name them with `--agents` (or `all`), because in a user's home each agent has its own skills root. `--local` keeps the installation inside the project instead, where `.agents/skills` is the one root pi, codex and opencode all discover, so a single `SKILL.md` serves all three; `--local` on its own is an error, because it only selects the scope of an installation. An install never needs `--force`: the skill is dflow's own artifact and is kept current even when the installed copy was edited by hand, unlike the workflow document, which is left untouched unless `--force` is passed.
- `dflow agent` writes both of those artifacts safely: the installed `SKILL.md` and the instruction references are written atomically — the bytes go to a temporary file beside the target and move into place with one rename — so an interrupted run leaves the previous file whole instead of a truncated one, and a rewrite preserves the permission bits of the file it replaces, keeping a hand-restricted `0600` `SKILL.md` at `0600` instead of widening it to `0644`. A file that does not exist yet has no prior mode to honor and is created at `0644`.
- `pkg/agent`, a new exported library package that holds the agent workflow work: the per-agent discovery registry, the document generator, the instruction-reference writer, and the skill installer.

### Changed
- `dflow update --check` renders its report at one shared text measure: lines longer than the terminal wrap at word boundaries, the "What's new" digest hangs its continuation lines under the bullet text, and the sections of the digest are separated by a blank line. The digest's own version heading is dropped instead of repeating the `what's new in vX:` line above it, and its length is bounded by characters alone rather than by a fixed item count, with an ellipsis line marking a truncated digest.
- The "What's new" digest's section headings render bold on a terminal, so each block is visually anchored. The styling and the wrapping are both terminal-only: a pipe, a file or a command substitution receives plain, unwrapped lines, so piped and scripted output stays easy to copy verbatim.

### Fixed
- `make release` fails fast instead of publishing an empty or stale release body: the notes extraction rejects a `CHANGELOG.md` with no version section or a section heading with no content below it, a rejected run removes any previously generated release body so nothing publishable survives it, and the release preflights a readable `.env` with a non-empty `GITHUB_TOKEN`, a commit that sits exactly on a tag, and a newest changelog section whose version matches that tag.
- A read-only CI job now runs the release-path validation without publishing anything: it validates the GoReleaser configuration and asserts the changelog contract the release body depends on, so a broken configuration or an unextractable changelog section is caught on every push and pull request instead of by a maintainer running `make release`. The GoReleaser configuration is deprecation-free so `goreleaser check` is a real gate, and its inert `changelog:` block is gone because the release body is supplied from `CHANGELOG.md`.

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
