// dflow is a modern CLI tool for managing Git branching workflows.
// Inspired by Git Flow, it simplifies the process of starting, managing, and finishing
// feature, release, and hotfix branches with a customizable YAML-based configuration.
//
// Key features:
//   - Interactive init wizard for first-time setup
//   - Commands to start features, releases, and hotfixes
//   - Automatic or manual (PR-based) merge modes
//   - Branch rules per environment (e.g., protect main, allow direct to develop)
//   - Generates changelogs with metadata for traceability
//
// Example usage:
//
//	dflow init
//	dflow start feat login-form
//	dflow config author "Author name" --email email@domain.com
//
// For full documentation, visit:
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
