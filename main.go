// dflow is a modern CLI tool for managing Git branching workflows.
// Inspired by Git Flow, it simplifies the process of starting, managing, and finishing
// work branches with a customizable YAML-based configuration.
//
// The tool combines interactive setup, configurable branch rules, and finish automation
// on top of a project-local `.dflow.yaml` file. It supports feature, release, hotfix,
// and bugfix branches, together with per-target merge strategies such as `auto` and
// `manual`.
//
// Common entry points:
//
//	dflow init
//	dflow start feat login-form
//	dflow finish --dry-run
//	dflow config set-author "Author Name" --email email@domain.com
//	dflow version
//
// For user documentation and release notes, visit:
//
//	https://github.com/yepizrene-devoost/dflow
package main

import (
	"github.com/yepizrene-devoost/dflow/cmd/root"
	"github.com/yepizrene-devoost/dflow/cmd/utils"
)

var version = "dev"

func main() {
	utils.SetVersion(version)
	utils.HandleInterrupt()
	root.Execute()
}
