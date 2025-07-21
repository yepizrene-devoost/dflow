// Package commands provides the CLI subcommands for dflow, enabling users to manage
// Git branching workflows using a consistent, configurable model.
//
// The `start` command creates and checks out a new branch based on the type (feature,
// release, or hotfix) defined in the `.dflow.yaml` configuration.
//
// It determines the correct base branch and prefix, creates the new branch locally,
// pulls the latest changes from the base branch, and offers to push it to the remote.
//
// Valid types include:
//   - feat|feature:  starts from `feature_base`
//   - release:       starts from `release_base`
//   - hotfix:        starts from `hotfix_base`
//
// Example usage:
//
//	dflow start feat login-form
//	dflow start release v1.0.0
//	dflow start hotfix urgent-patch
package commands

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/yepizrene-devoost/dflow/cmd/gitutils"
	"github.com/yepizrene-devoost/dflow/cmd/utils"
	"github.com/yepizrene-devoost/dflow/pkg/validators"
)

var StartCmd = &cobra.Command{
	Use:   "start [type] [name]",
	Short: "Create and switch to a new feature, release, or hotfix branch",
	Long: `Start a new Git branch following the dflow branching model.
	
  Valid types:
    - feat|feature : Starts a new feature branch from the configured 'feature_base'
    - release      : Starts a new release branch from the configured 'release_base'
    - hotfix       : Starts a new hotfix branch from the configured 'hotfix_base'

  Examples:
    dflow start feat login-form
    dflow start release v1.0.0
    dflow start hotfix urgent-patch

  The new branch will be created using the appropriate prefix (e.g., feature/, release/, hotfix/)
  and based on the corresponding base branch defined in your .dflow.yaml configuration.`,

	Args: cobra.MaximumNArgs(2),
	RunE: validators.WithChecks(false, func(cmd *cobra.Command, args []string) error {

		if len(args) != 2 {
			_ = cmd.Help()
			return nil
		}

		branchType := args[0]
		branchName := args[1]

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
		case "hotfix":
			prefix = cfg.Branches.Hotfixes
			base = cfg.Flow.HotfixBase
		default:
			utils.Error("Unknown type. Use: feat, release, hotfix")
			return nil
		}

		fullName := fmt.Sprintf("%s%s", prefix, branchName)

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
			utils.Success("Branch '%s' pushed to origin", fullName)
		}

		return nil
	}),
}
