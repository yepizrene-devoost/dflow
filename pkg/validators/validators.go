// Package validators provides pre-execution checks for dflow commands.
//
// These checks ensure that commands are executed within a Git repository
// and that the project has been initialized with a .dflow.yaml file.
package validators

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
	"github.com/yepizrene-devoost/dflow/cmd/utils"
)

// EnsureGitRepo returns an error if the current directory is not a Git repository.
//
// It checks for the existence of a `.git` folder in the current working directory.
func EnsureGitRepo() error {
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		return errors.New("this is not a Git repository")
	}
	return nil
}

// EnsureDflowInitialized returns an error if `.dflow.yaml` is not found in the current directory.
//
// This check ensures that the user has run `dflow init` before using other commands.
func EnsureDflowInitialized() error {
	if _, err := os.Stat(".dflow.yaml"); os.IsNotExist(err) {
		return errors.New("dflow is not initialized in this repository. Run `dflow init` first")
	}
	return nil
}

// WithChecks wraps a Cobra command handler function (`RunE`) with repository and config validations.
//
// If `skipDflowCheck` is false, it verifies that `.dflow.yaml` exists.
//
// Typical usage:
//
//	cmd.RunE = validators.WithChecks(false, func(cmd *cobra.Command, args []string) error {
//		// command logic here...
//		return nil
//	})
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
