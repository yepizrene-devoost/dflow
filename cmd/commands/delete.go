// Package commands provides the CLI subcommands for dflow, enabling users to manage
// Git branching workflows using a consistent, configurable model.
//
// This includes project-local configuration commands under `dflow config`,
// allowing users to set and retrieve metadata such as author name and email
// for use in changelogs and other automated processes.
package commands

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/yepizrene-devoost/dflow/cmd/gitutils"
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
	Short: "Delete branch created previously",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		branch := args[0]
		var confirm bool
		err := survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Are you sure you want to delete branch '%s' locally and remotely?", branch),
			Default: false,
		}, &confirm)
		if err != nil {
			fmt.Println("⚠️  Deletion cancelled.")
			return nil
		}

		if !confirm {
			fmt.Println("🚫 Operation aborted by user.")
			return nil
		}

		return gitutils.Delete(branch)
	},
}

func init() {
	DeleteCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		branches := gitutils.GetLocalBranches()
		return branches, cobra.ShellCompDirectiveNoFileComp
	}
}
