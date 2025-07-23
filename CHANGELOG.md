# Changelog

## 📦 v0.1.2 – Bugfix Flow, Shell Completion & Interactive Commands

### ⚙️ Added
- Support for `bug` / `bugfix` branches:
  - Added `bugfixes:` and `bugfix_base:` fields in `.dflow.yaml`.
  - New branch types accepted in `dflow start`: `bug`, `bugfix`.
- New `dflow delete` command:
  - Deletes local and remote branches with interactive confirmation.
  - Includes autocompletion for local branch names.
- Shell autocompletion support:
  - Added `dflow completion` command with subcommands for `bash`, `zsh`, `fish`, and `powershell`.
  - Added `dflow completion install` for persistent setup.
  - Autocompletion added to `start` and `delete` commands.
- Spinner utility for better UX during Git operations (`pull`, `push`, `delete`).
- New message helpers in `utils`: `Success`, `Error`, `Info`, and `Warn` with emoji icons and formatted output.

### 📝 Changed
- `.dflow.yaml`:
  - Switched to 4-space indentation for consistency.
  - Extended to include bugfix flow definitions.
- `dflow init`:
  - Now validates and optionally creates missing base branches.
  - Prompts to push base branches to remote with improved error handling.
- `dflow start`:
  - Improved error messages and UX flow.
  - Added support for `bug`/`bugfix`, `fix`, and `hot` as aliases.
  - Interactive push prompt with success feedback.
- CLI logs now use consistent and expressive output via emoji helpers.

### 🧪 Internal
- Major refactor of `gitutils`:
  - Split responsibilities into well-defined helpers like `Delete`, `PushBranch`, `RemoteBranchExists`, etc.
  - Wrapped Git calls with spinners and error reporting.
- Added autocompletion support via Cobra’s native completion framework.
- Expanded GoDoc coverage and in-line documentation across all commands and utilities.
- Graceful `Ctrl+C` handling with custom signal listener.

## 🔮 Future Plans
- Add `dflow finish` to complete branches and perform merges automatically.
- Automate changelog generation from Git history.
- Add tests for core Git integrations.
- Improve `config` options to support defaults or shared templates.
---
