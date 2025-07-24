// Package validators provides pre-execution checks for dflow commands.
//
// These checks ensure that commands are executed within a Git repository
// and that the project has been initialized with a .dflow.yaml file.
package validators

import (
	"errors"
	"fmt"
	"os"
	"strings"

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

// IsValidGitBranchName checks whether a given branch name is valid according to Git's reference format rules.
//
// A branch name is considered invalid if it contains any of the following:
//   - Double dots ("..")
//   - Special characters such as ~, ^, :, ?, *, [, \, or the sequence "@{"
//   - Starts with a dash ("-")
//   - Ends with a slash ("/"), a dot ("."), or ".lock"
//   - Is an empty string
//
// These checks are based on Git's rules for ref names described in `git-check-ref-format(1)`.
//
// Returns true if the branch name is valid, false otherwise.
func IsValidGitBranchName(name string) (bool, string) {
	if name == "" {
		return false, "branch name is empty"
	}

	if strings.HasPrefix(name, "-") {
		return false, "branch name cannot start with '-'"
	}
	if strings.HasPrefix(name, "/") {
		return false, "branch name cannot start with '/'"
	}
	if strings.HasSuffix(name, "/") {
		return false, "branch name cannot end with '/'"
	}
	if strings.HasSuffix(name, ".") {
		return false, "branch name cannot end with '.'"
	}
	if strings.HasSuffix(name, ".lock") {
		return false, "branch name cannot end with '.lock'"
	}

	invalidPatterns := []string{"..", "~", "^", ":", "?", "*", "[", "\\", "@{"}
	for _, pattern := range invalidPatterns {
		if strings.Contains(name, pattern) {
			return false, fmt.Sprintf("branch name cannot contain '%s'", pattern)
		}
	}

	// Component-level checks (split by '/')
	parts := strings.Split(name, "/")
	for _, part := range parts {
		if part == "." || part == ".." {
			return false, "branch name cannot contain path elements '.' or '..'"
		}
		if part == "" {
			return false, "branch name cannot contain empty path segments ('//')"
		}
	}

	// Control characters (ASCII 0-31) and DEL (127)
	for i := 0; i < len(name); i++ {
		if name[i] < 32 || name[i] == 127 {
			return false, "branch name contains control characters"
		}
	}

	return true, ""
}
