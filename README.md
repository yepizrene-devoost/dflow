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

- Select base branches (`main`, `develop`, etc.)
- Define prefixes for `feature`, `release`, `hotfix`
- Choose merge behavior (auto vs manual/PR)
- Creates a `.dflow.yaml` config file

---

### `dflow start <type> <name>`

Start a new branch based on your workflow config.

```bash
dflow start feat login-form
dflow start release v1.2.0
dflow start hotfix urgent-patch
dflow start bug broken checkout
```

- Supports branch types: `feature`, `release`, `hotfix`, and `bugfix`
- Auto-generates branch names like `feature/login-form` or `bugfix/broken-checkout`
- Supports multi-word names, normalizing to kebab-case (e.g. `"login screen bug"` → `login-screen-bug`)
- Validates Git branch name safety before creation
- Creates the branch and checks it out

---

### `dflow finish`

Finish the current dflow work branch using your configured merge rules.

```bash
dflow finish
```

- Detects the current branch type from your configured prefixes
- Resolves the configured `finish_targets` for that branch type
- Automatically merges and pushes only the targets with `merge_mode: auto`
- Reports any `manual` targets so they can be completed through PR flow
- Stops immediately if a merge conflict occurs on an auto target

---

### `dflow config`

Manage user-level configuration.

```bash
dflow config author "Author Name" --email authoremail@domain.com
```

- Stores global author info used in changelogs
- Respects `.gitconfig` or local overrides

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
    release:
        base: develop
        finish_targets:
            - uat
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

---

## 🥮 Example Workflow

```bash
dflow init
# Select main = main, develop = develop

dflow start feat login-form
# ⇒ Creates and switches to feature/login-form

# work and commit...

dflow finish
# ⇒ Merges feature/login-form into every auto target and reports any manual targets
```

---

## ✨ Features

- ✅ Interactive `init` wizard
- ✅ Customizable prefixes and merge rules
- ✅ Support for hybrid workflows (direct merge + PR)
- ✅ Git-aware config and validation
- ✅ `dflow finish` with auto/manual target handling
- 📦 Multiplatform builds (via `GoReleaser`)
- 🌐 Multi-language documentation (`README.md`, `README.es.md`)

---

## 📄 License

This project is licensed under the [MIT License](LICENSE)  
© 2025 Rene Yepiz – yepizrene@gmail.com

---

## 🤝 Contributing

Contributions are welcome! Open an issue or PR and let's improve Git workflows together ✌️
