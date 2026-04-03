// Package commands defines the end-user subcommands that make up the dflow CLI.
//
// The package contains interactive and non-interactive commands for initializing
// repositories, starting and finishing work branches, deleting branches, and
// managing local dflow metadata stored in Git config.
package commands

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/yepizrene-devoost/dflow/cmd/gitutils"
	"github.com/yepizrene-devoost/dflow/cmd/utils"
	"github.com/yepizrene-devoost/dflow/pkg/validators"
)

// StartCmd creates and switches to a new Git branch based on the dflow branching model.
//
// Supported branch types:
//
//   - feat|feature  : Creates a feature branch from the configured feature base
//   - release       : Creates a release branch from the configured release base
//   - fix|hot|hotfix: Creates a hotfix branch from the configured hotfix base
//   - bug|bugfix    : Creates a bugfix branch from the configured bugfix base
//
// Branches are automatically prefixed using values from `.dflow.yaml`
// under `branches.features`, `branches.releases`, or `branches.hotfixes`.
//
// This command performs the following steps:
//  1. Checks out the appropriate base branch
//  2. Pulls the latest changes from origin
//  3. Creates and checks out the new branch
//  4. Prompts the user to push the new branch to origin
//
// Example usage:
//
//	dflow start feat login-form
//	dflow start release v1.0.0
//	dflow start hotfix urgent-patch
//	dflow start bug bug-on-uat-detected
//
// If arguments are missing, help text is shown instead.
var StartCmd = &cobra.Command{
	Use:   "start [type] [name]",
	Short: "Create and switch to a new feature, release, hotfix, or bugfix branch",
	Long: `Start a new Git branch following the dflow branching model.
	
  Valid types:
    - feat|feature	: Starts a new feature branch from the configured feature base branch
    - release	: Starts a new release branch from the configured release base branch
    - fix|hot|hotfix	: Starts a new hotfix branch from the configured hotfix base branch
    - bug|bugfix	: Starts a new bugfix branch from the configured bugfix base branch

  Examples:
    dflow start feat login-form
    dflow start release v1.0.0
    dflow start hotfix urgent-patch
    dflow start bug bug-on-uat-detected

  The new branch will be created using the appropriate prefix (e.g., feature/, release/, hotfix/, bugfix/)
  and based on the corresponding base branch defined in your .dflow.yaml configuration.`,
	DisableFlagParsing: true,

	Args: cobra.ArbitraryArgs,
	RunE: validators.WithChecks(false, func(cmd *cobra.Command, args []string) error {

		if len(args) < 2 {
			_ = cmd.Help()
			return nil
		}

		branchType, err := utils.ParseBranchType(args[0])
		if err != nil {
			utils.Error("Unknown type. Use: feat, release, hotfix, bugfix")
			return nil
		}

		//normalize name of branch, change "word with word" or multiple void spaaces to "word-with-word"
		branchNameParts := strings.Fields(strings.Join(args[1:], " "))
		branchName := strings.Join(branchNameParts, "-")

		if strings.HasPrefix(branchName, "-") {
			utils.Warn("Branch name appears to start with '-'.")
			utils.Info("If your branch name starts with '-', wrap it in quotes or use '--' to avoid flag parsing issues.")
		}

		cfg, err := utils.LoadConfig()
		if err != nil {
			utils.Error(err.Error())
			return nil
		}

		prefix, err := utils.GetBranchPrefix(cfg, branchType)
		if err != nil {
			utils.Error(err.Error())
			return nil
		}

		rule, err := utils.GetFlowRule(cfg, branchType)
		if err != nil {
			utils.Error(err.Error())
			return nil
		}
		base := rule.Base

		fullName := fmt.Sprintf("%s%s", prefix, branchName)

		if valid, reason := validators.IsValidGitBranchName(fullName); !valid {
			utils.Error("Invalid branch name '%s': %s", fullName, reason)
			return nil
		}

		if err := gitutils.Checkout(base); err != nil {
			utils.Error("Could not checkout base branch '%s'", base)
			return nil
		}

		if err := gitutils.Pull(); err != nil {
			utils.Error("Failed to pull latest changes from '%s'", base)
			return nil
		}

		if err := gitutils.CheckoutNew(fullName); err != nil {
			utils.Error("Failed to create branch '%s'", fullName)
			return nil
		}

		utils.Success("Created and switched to branch '%s' from '%s'", fullName, base)

		// Ask to push
		var pushBranch bool
		err = survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Do you want to publish '%s' to origin?", fullName),
			Default: true,
		}, &pushBranch)
		if err != nil {
			fmt.Println("⚠️  Skipping push...")
			return nil
		}

		if pushBranch {
			if err := gitutils.PushBranch(fullName); err != nil {
				utils.Error("Failed to push branch '%s': %v", fullName, err)
				return err
			}
		}

		return nil
	}),
}

func init() {
	StartCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return []string{
				"feat\tAlias for 'feature'",
				"feature\tStart a new feature branch",
				"release\tStart a new release branch",
				"hot\tAlias for 'hotfix'",
				"hotfix\tStart a new hotfix branch",
				"bugfix\tStart a new bugfix branch",
				"bug\tAlias for 'bugfix'",
			}, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}
