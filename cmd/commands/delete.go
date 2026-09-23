// Package commands defines the end-user subcommands that make up the dflow CLI.
//
// The package contains interactive and non-interactive commands for initializing
// repositories, starting and finishing work branches, deleting branches, and
// managing local dflow metadata stored in Git config.
package commands

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/yepizrene-devoost/dflow/cmd/gitutils"
	"github.com/yepizrene-devoost/dflow/cmd/utils"
)

// DeleteCmd deletes a Git branch locally and remotely using the dflow CLI.
//
// This command requires the exact name of the branch to delete. It will:
//
//  1. Ask for confirmation before proceeding
//  2. Delete the local branch
//  3. Delete the corresponding remote branch from origin (if it exists)
//
// Example usage:
//
//	dflow delete feature/login-form
//
// Autocompletion suggests local branches when available.
var DeleteCmd = &cobra.Command{
	Use:   "delete <branch>",
	Short: "Delete a local branch and its remote counterpart",
	Long: `Delete a branch created with dflow from your local repository and, if it
exists, from the 'origin' remote as well.

The command asks for confirmation before deleting anything and skips the remote
step automatically when the branch does not exist on origin.`,
	Example: `  dflow delete feature/login-form
  dflow delete bugfix/payment-timeout`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		branch := args[0]

		yes, _ := cmd.Flags().GetBool("yes")
		if !yes {
			if !utils.IsInteractive() {
				return utils.NonInteractiveError(
					fmt.Sprintf("confirm deletion of '%s'", branch),
					"pass --yes to delete it",
				)
			}

			var confirm bool
			if err := survey.AskOne(&survey.Confirm{
				Message: fmt.Sprintf("Are you sure you want to delete branch '%s' locally and remotely?", branch),
				Default: false,
			}, &confirm); err != nil {
				return err
			}

			if !confirm {
				utils.Warn("Operation aborted by user.", "🚫")
				return nil
			}
		}

		return gitutils.Delete(branch)
	},
}

func init() {
	DeleteCmd.Flags().BoolP("yes", "y", false, "delete without asking for confirmation")

	DeleteCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		branches := gitutils.GetLocalBranches()
		return branches, cobra.ShellCompDirectiveNoFileComp
	}
}
