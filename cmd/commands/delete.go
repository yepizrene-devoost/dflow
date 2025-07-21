package commands

import (
	"github.com/spf13/cobra"
	"github.com/yepizrene-devoost/dflow/cmd/gitutils"
)

// DeleteCmd represents the `delete` command.
//
// It deletes a Git branch both locally and remotely. The user must specify the
// name of the branch as an argument.
//
// Example usage:
//
//	dflow delete feature/my-branch
var DeleteCmd = &cobra.Command{
	Use:   "delete <branch>",
	Short: "Delete branch created previously",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		branch := args[0]
		return gitutils.Delete(branch)
	},
}

func init() {
	DeleteCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		branches := gitutils.GetLocalBranches()
		return branches, cobra.ShellCompDirectiveNoFileComp
	}
}
