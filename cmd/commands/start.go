package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yepizrene-devoost/dflow/cmd/gitutils"
	"github.com/yepizrene-devoost/dflow/cmd/utils"
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
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			_ = cmd.Help()
			return
		}

		branchType := args[0]
		branchName := args[1]

		cfg, err := utils.LoadConfig()
		if err != nil {
			utils.Error(err.Error())
			return
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
			return
		}

		fullName := fmt.Sprintf("%s%s", prefix, branchName)

		// Verificar y hacer checkout de la base
		if err := gitutils.Checkout(base); err != nil {
			utils.Error(fmt.Sprintf("Could not checkout base branch '%s'", base))
			return
		}

		if err := gitutils.Pull(); err != nil {
			utils.Error(fmt.Sprintf("Failed to pull latest changes from '%s'", base))
			return
		}

		// Crear la nueva rama desde base
		if err := gitutils.CheckoutNew(fullName); err != nil {
			utils.Error(fmt.Sprintf("Failed to create branch '%s'", fullName))
			return
		}

		utils.Success(fmt.Sprintf("Created and switched to branch '%s' from '%s'", fullName, base))
	},
}
