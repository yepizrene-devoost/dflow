// pkg/utils/validators.go (o similar)
package utils

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
	"github.com/yepizrene-devoost/dflow/cmd/utils"
)

func EnsureGitRepo() error {
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		return errors.New("this is not a Git repository")
	}
	return nil
}

func EnsureDflowInitialized() error {
	if _, err := os.Stat(".dflow.yaml"); os.IsNotExist(err) {
		return errors.New("dflow is not initialized in this repository. Run `dflow init` first")
	}
	return nil
}

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
