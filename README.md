<p align="center">
  <pre>
                          ██████╗ ███████╗██╗      ██████╗ ██╗    ██╗
                          ██╔══██╗██╔════╝██║     ██╔═══██╗██║    ██║
                          ██║  ██║█████╗  ██║     ██║   ██║██║ █╗ ██║
                          ██║  ██║██╔══╝  ██║     ██║   ██║██║███╗██║
                          ██████╔╝██║     ███████╗╚██████╔╝╚███╔███╔╝
                          ╚═════╝ ╚═╝     ╚══════╝ ╚═════╝  ╚══╝╚══╝ 
                                  Git branching made simple
  </pre>
</p>

<p align="center"><b>dflow</b> – A Git branching CLI inspired by Git Flow with a modern and customizable workflow.</p>

<p align="center">
  <a href="https://opensource.org/licenses/MIT">
    <img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License: MIT">
  </a>
  <a href="https://goreportcard.com/report/github.com/yepizrene-devoost/dflow">
    <img src="https://goreportcard.com/badge/github.com/yepizrene-devoost/dflow" alt="Go Report Card">
  </a>
  <a href="https://pkg.go.dev/github.com/yepizrene-devoost/dflow">
    <img src="https://pkg.go.dev/badge/github.com/yepizrene-devoost/dflow.svg" alt="Go Reference">
  </a>
  <a href="https://github.com/yepizrene-devoost/dflow/actions/workflows/go.yml">
    <img src="https://img.shields.io/github/actions/workflow/status/yepizrene-devoost/dflow/go.yml?branch=main&label=build:%20main" alt="Build: main">
  </a>
  <a href="https://github.com/yepizrene-devoost/dflow/actions/workflows/go.yml">
    <img src="https://img.shields.io/github/actions/workflow/status/yepizrene-devoost/dflow/go.yml?branch=develop&label=build:%20develop" alt="Build: develop">
  </a>
  <a href="https://github.com/yepizrene-devoost/dflow/releases">
    <img src="https://img.shields.io/github/v/release/yepizrene-devoost/dflow?sort=semver" alt="Latest Release">
  </a>
</p>



---

## 🚀 About

**dflow** is a lightweight CLI tool that brings structure, consistency, and adaptability to Git branching workflows. Inspired by [Git Flow](https://nvie.com/posts/a-successful-git-branching-model/), it simplifies modern development practices with:

- Minimal setup and easy onboarding
- Opinionated branching strategies with flexibility
- Support for environments with or without pull requests

Whether you're working solo or in a team, `dflow` helps you keep your Git history clean and your process repeatable.

---

## 📦 Installation

### 🧪 Option 1: Precompiled Binaries (Recommended)

Download the latest binary for your platform from the [Releases page](https://github.com/yepizrene-devoost/dflow/releases).

#### Linux (x86_64)
```bash
# download dflow distributable package
curl -LO https://github.com/yepizrene-devoost/dflow/releases/latest/download/dflow_Linux_x86_64.tar.gz

# create temp dir and extract files into
mkdir -p dflow_tmp && tar -xzf dflow_Linux_x86_64.tar.gz -C dflow_tmp

# move dflow executable bin
sudo mv dflow_tmp/dflow /usr/local/bin/

# clean temp files
rm -rf dflow_Linux_x86_64.tar.gz dflow_tmp
```

#### macOS (Apple Silicon)
```bash
# download dflow distributable package
curl -LO https://github.com/yepizrene-devoost/dflow/releases/latest/download/dflow_Darwin_arm64.tar.gz

# create temp dir and extract files into
mkdir -p dflow_tmp && tar -xzf dflow_Darwin_arm64.tar.gz -C dflow_tmp

# move dflow executable bin
sudo mv dflow_tmp/dflow /usr/local/bin/

# clean temp files
rm -rf dflow_Darwin_arm64.tar.gz dflow_tmp

```

#### Windows (x86_64)

1. Download: [dflow_Windows_x86_64.zip](https://github.com/yepizrene-devoost/dflow/releases/latest/download/dflow_Windows_x86_64.zip)  
2. Extract and add the folder to your system PATH manually.

> 💡 Make sure `/usr/local/bin` (or equivalent) is in your `$PATH`.

---

### 🛠 Option 2: Build from Source

Requires [Go 1.21+](https://golang.org/doc/install):

```bash
git clone https://github.com/yepizrene-devoost/dflow.git
cd dflow
go install
```
---

## 🛠️ Commands

### `dflow init`

Interactive setup for your repository.

```bash
dflow init
```

- Prompts for your `main`, `develop`, and `uat` branch names
- Generates a `.dflow.yaml` file with branch prefixes, base branches, finish targets, and merge rules
- Lets you choose the default merge mode and per-branch exceptions
- Ensures the main branches exist locally and can push them to `origin`

### `dflow start <type> <name>`

Start a new work branch based on your workflow config.

```bash
dflow start feat login-form
dflow start release v1.2.0
dflow start hotfix urgent-patch
dflow start bug broken checkout
```

- Supports `feature`, `release`, `hotfix`, and `bugfix` with aliases such as `feat`, `hot`, `fix`, and `bug`
- Uses the configured prefix and `base` branch for the selected type
- Normalizes multi-word names to kebab-case
- Validates the resulting Git branch name before creating it
- Checks out the new branch and optionally publishes it to `origin`

### `dflow finish`

Finish the current dflow work branch using your configured merge rules.

```bash
dflow finish
dflow finish --dry-run
dflow finish --delete
```

- Detects the current branch type from your configured prefixes
- Resolves the configured `finish_targets` for that branch type
- Automatically merges and pushes only the targets with `merge_mode: auto`
- Reports `manual` targets so they can be completed through PR flow
- Requires a clean working tree before running
- Stops immediately if an auto target hits a merge conflict
- Returns you to the configured `base` branch after a successful finish
- Does not delete the source branch automatically
- Supports `--delete` to remove the finished branch locally and remotely after a successful finish when no manual targets remain
- Supports `--dry-run` to preview the plan without fetching, merging, or pushing

### `dflow delete <branch>`

Delete a branch locally and remotely.

```bash
dflow delete feature/login-form
```

- Asks for confirmation before deleting anything
- Removes the local branch
- Deletes the remote branch from `origin` when it exists

### `dflow config`

Manage project-local dflow metadata stored in Git config.

```bash
dflow config set-author "Author Name" --email author@example.com
dflow config get-author
dflow config list
```

- `set-author` stores the local `dflow.author` and `dflow.email` values
- `get-author` prints the values currently configured for the repository
- `list` shows every local Git config entry under the `dflow.*` namespace

### `dflow completion`

Generate or install shell completion scripts.

```bash
dflow completion zsh
dflow completion install
```

- Supports `bash`, `zsh`, `fish`, and `powershell`
- Can output scripts directly or install them for the current shell

### `dflow version`

Show the current CLI version.

```bash
dflow version
dflow --version
dflow -V
```

---

## 🔧 Configuration

`dflow` uses a `.dflow.yaml` file stored at the root of your repository. It is auto-generated by `dflow init` and looks like this:

```yaml
#
#               ██████╗ ███████╗██╗      ██████╗ ██╗    ██╗
#               ██╔══██╗██╔════╝██║     ██╔═══██╗██║    ██║
#               ██║  ██║█████╗  ██║     ██║   ██║██║ █╗ ██║
#               ██║  ██║██╔══╝  ██║     ██║   ██║██║███╗██║
#               ██████╔╝██║     ███████╗╚██████╔╝╚███╔███╔╝
#               ╚═════╝ ╚═╝     ╚══════╝ ╚═════╝  ╚══╝╚══╝ 

#            dflow config file - autogenerated by 'dflow init'

branches:
    main: main
    develop: develop
    uat: uat
    features: feature/
    releases: release/
    hotfixes: hotfix/
    bugfixes: bugfix/

flow:
    feature:
        base: develop
        finish_targets:
            - develop
            - uat
    release:
        base: uat
        finish_targets:
            - main
            - develop
    hotfix:
        base: main
        finish_targets:
            - main
            - develop
            - uat
    bugfix:
        base: uat
        finish_targets:
            - uat
            - develop

workflow:
    default_merge_mode: auto
    branch_rules:
        main:
            merge_mode: manual
        develop:
            merge_mode: auto
        uat:
            merge_mode: auto
```

### How To Read `.dflow.yaml`

- `branches` defines your primary branch names and the prefixes used to create work branches.
- `flow.<type>.base` defines the branch used when `dflow start` creates a new branch of that type.
- `flow.<type>.finish_targets` defines where that branch is expected to land when `dflow finish` runs.
- `workflow.default_merge_mode` is the fallback mode used for targets not explicitly listed in `branch_rules`.
- `workflow.branch_rules.<branch>.merge_mode` controls whether a target branch is handled by direct merge (`auto`) or left for PR/manual flow (`manual`).

If `uat` and `develop` are the same branch in your repository, dflow automatically deduplicates finish targets so the same branch is not processed twice. In that setup, a feature configured for both `develop` and `uat` will be processed only once.

### How `dflow finish` Works

When you run `dflow finish`, dflow:

1. Detects the current work branch type from the configured prefix.
2. Resolves the `base` and `finish_targets` for that branch type.
3. Splits targets into `auto` and `manual` using the merge rules in `workflow.branch_rules`.
4. For each `auto` target, fetches from `origin`, checks out and updates the target branch, merges the current work branch, and pushes the result.
5. Reports any `manual` targets without merging them.
6. Returns to the configured `base` branch when all automatic merges succeed.

Use `dflow finish --delete` if you want dflow to remove the finished branch
locally and remotely after all automatic targets succeed and no manual follow-up
remains.

Use `dflow finish --dry-run` to inspect that plan safely before touching any branch.

---

## 🥮 Example Workflow

```bash
dflow init
# Select main = main, develop = develop

dflow start feat login-form
# ⇒ Creates and switches to feature/login-form

# work and commit...

dflow finish --dry-run
# ⇒ Shows auto/manual targets and the branch dflow would return to

dflow finish
# ⇒ Merges feature/login-form into every auto target, reports any manual targets,
#    and returns to the configured base branch
```

---

## ✨ Features

- ✅ Interactive `init` wizard
- ✅ Customizable prefixes, base branches, finish targets, and merge rules
- ✅ Support for hybrid workflows (direct merge + PR)
- ✅ Git-aware config, validation, and branch safety checks
- ✅ `dflow finish` with auto/manual target handling and `--dry-run`
- ✅ `dflow delete`, `dflow completion`, and `dflow version`
- ✅ Go Reference available on `pkg.go.dev`
- 📦 Multiplatform builds (via `GoReleaser`)
- 📚 Open-source friendly documentation in `README.md`, `RELEASING.md`, `CHANGELOG.md`, and `HISTORY.md`

---

## 📄 License

This project is licensed under the [MIT License](LICENSE)  
© 2025 Rene Yepiz – yepizrene@gmail.com

---

## 🤝 Contributing

Contributions are welcome! Open an issue or PR and let's improve Git workflows together ✌️
