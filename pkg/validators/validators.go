// Package validators provides pre-execution checks for dflow commands.
// It ensures the CLI runs in valid Git repositories and verifies that dflow is properly initialized.
package validators

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
	"github.com/yepizrene-devoost/dflow/cmd/utils"
)

// EnsureGitRepo returns an error if the current directory is not a Git repository.
// It checks for the presence of a .git folder.
func EnsureGitRepo() error {
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		return errors.New("this is not a Git repository")
	}
	return nil
}

// EnsureDflowInitialized returns an error if the .dflow.yaml file is not found.
// This ensures that `dflow init` has been run in the current project.
func EnsureDflowInitialized() error {
	if _, err := os.Stat(".dflow.yaml"); os.IsNotExist(err) {
		return errors.New("dflow is not initialized in this repository. Run `dflow init` first")
	}
	return nil
}

// WithChecks wraps a Cobra command handler with pre-validation checks.
// It ensures the command is run inside a Git repo and optionally verifies
// that dflow has been initialized (i.e., .dflow.yaml exists).
//
// Use skipDflowCheck = true for commands like `dflow init` that should run
// before initialization.
func WithChecks(skipDflowCheck bool, fn func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if err := EnsureGitRepo(); err != nil {
			utils.Error(err.Error())
			return nil
		}

		if !skipDflowCheck {
			if err := EnsureDflowInitialized(); err != nil {
				utils.Error(err.Error())
				return nil
			}
		}
		return fn(cmd, args)
	}
}
