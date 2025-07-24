// Package commands provides the CLI subcommands for dflow, enabling users to manage
// Git branching workflows using a consistent, configurable model.
//
// This includes project-local configuration commands under `dflow config`,
// allowing users to set and retrieve metadata such as author name and email
// for use in changelogs and other automated processes.
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
//   - feat|feature : Creates a feature branch from `flow.feature_base`
//   - release      : Creates a release branch from `flow.release_base`
//   - fix|hot|hotfix       : Creates a hotfix branch from `flow.hotfix_base`
//   - bug|bugfix       : Creates a bugfix branch from `flow.bugfix_base`
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
	Short: "Create and switch to a new feature, release, or hotfix branch",
	Long: `Start a new Git branch following the dflow branching model.
	
  Valid types:
    - feat|feature	: Starts a new feature branch from the configured 'feature_base'
    - release	: Starts a new release branch from the configured 'release_base'
    - fix|hot|hotfix	: Starts a new hotfix branch from the configured 'hotfix_base'
    - bug|bugfix	: Starts a new bugfix branch from the configured 'bugfix_base'

  Examples:
    dflow start feat login-form
    dflow start release v1.0.0
    dflow start hotfix urgent-patch
    dflow start bug bug-on-uat-detected

  The new branch will be created using the appropriate prefix (e.g., feature/, release/, hotfix/, bugfix/)
  and based on the corresponding base branch defined in your .dflow.yaml configuration.`,
	DisableFlagParsing: true,

	Args: cobra.MinimumNArgs(2),
	RunE: validators.WithChecks(false, func(cmd *cobra.Command, args []string) error {

		if len(args) < 2 {
			_ = cmd.Help()
			return nil
		}

		branchType := args[0]

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

		var prefix, base string

		switch branchType {
		case "feat", "feature":
			prefix = cfg.Branches.Features
			base = cfg.Flow.FeatureBase
		case "release":
			prefix = cfg.Branches.Releases
			base = cfg.Flow.ReleaseBase
		case "hot", "hotfix":
			prefix = cfg.Branches.Hotfixes
			base = cfg.Flow.HotfixBase
		case "bug", "bugfix":
			prefix = cfg.Branches.Bugfixes
			base = cfg.Flow.BugfixBase
		default:
			utils.Error("Unknown type. Use: feat, release, hotfix")
			return nil
		}

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
				"feat\tAlias for 'feat'",
				"feature\tStart a new feature branch",
				"release\tStart a new release branch",
				"hot\tAlias for 'hotfix'",
				"hotfix\tStart a new hotfix branch",
				"bug\tStart a new bugfix branch",
				"bugfix\tAlias for 'bug'",
			}, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}
