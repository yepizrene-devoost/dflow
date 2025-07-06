package commands

import (
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/spf13/cobra"
	"github.com/yepizrene-devoost/dflow/cmd/gitutils"
	"github.com/yepizrene-devoost/dflow/cmd/utils"
	"github.com/yepizrene-devoost/dflow/pkg/validators"
)

var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize your dflow branching configuration",
	Long: `Initialize your dflow branching configuration and generate a .dflow.yaml file.

  This interactive setup will guide you through the following steps:

    - Prompt for the names of your main, develop, and UAT branches.
    - Create a .dflow.yaml file with default prefixes:
      - feature/  → for feature branches
      - release/  → for release branches
      - hotfix/   → for hotfix branches
    - Set flow rules:
      - Features start from UAT and merge to Develop
      - Releases start from UAT
      - Hotfixes start from Main
    - Ensure the specified branches exist locally.
    - Ask whether to push those base branches to origin.

  The resulting .dflow.yaml is stored in the project root and used by all dflow commands.

  Example:
    dflow init

  This command is meant to be run once per project when setting up the dflow branching model.`,
	RunE: validators.WithChecks(true, func(cmd *cobra.Command, args []string) error {

		var mainBranch, developBranch, uatBranch string

		err := survey.AskOne(&survey.Input{Message: "Main branch name:", Default: "main"}, &mainBranch, survey.WithValidator(survey.Required))
		if err == terminal.InterruptErr {
			fmt.Println("\n🚫 Cancelled by user.")
			os.Exit(1)
		} else if err != nil {
			fmt.Println("❌ Error:", err)
			return nil
		}

		err = survey.AskOne(&survey.Input{Message: "Development branch name:", Default: "develop"}, &developBranch, survey.WithValidator(survey.Required))
		if err == terminal.InterruptErr {
			fmt.Println("\n🚫 Cancelled by user.")
			os.Exit(1)
		} else if err != nil {
			fmt.Println("❌ Error:", err)
			return nil
		}

		err = survey.AskOne(&survey.Input{Message: "UAT branch name:", Default: "uat"}, &uatBranch, survey.WithValidator(survey.Required))
		if err == terminal.InterruptErr {
			fmt.Println("\n🚫 Cancelled by user.")
			os.Exit(1)
		} else if err != nil {
			fmt.Println("❌ Error:", err)
			return nil
		}

		cfg := utils.Config{}
		cfg.Branches.Main = mainBranch
		cfg.Branches.Develop = developBranch
		cfg.Branches.Uat = uatBranch
		cfg.Branches.Features = "feature/"
		cfg.Branches.Releases = "release/"
		cfg.Branches.Hotfixes = "hotfix/"

		cfg.Flow.FeatureBase = uatBranch
		cfg.Flow.FeatureMerge = developBranch
		cfg.Flow.ReleaseBase = uatBranch
		cfg.Flow.HotfixBase = mainBranch

		if err := utils.SaveConfig(&cfg); err != nil {
			utils.Error(err.Error())
			return nil
		}
		utils.Success("Created .dflow.yaml")

		gitutils.CheckOrCreateBranch(mainBranch)
		gitutils.CheckOrCreateBranch(developBranch)
		gitutils.CheckOrCreateBranch(uatBranch)

		var pushConfirm bool
		survey.AskOne(&survey.Confirm{
			Message: "Do you want to push the base branches to 'origin'?",
			Default: true,
		}, &pushConfirm)

		if pushConfirm {
			gitutils.PushBranch(mainBranch)
			gitutils.PushBranch(developBranch)
			gitutils.PushBranch(uatBranch)
		}

		utils.Success("🎉 dflow is ready! Use `dflow start` to begin a new branch.")
		return nil
	}),
}
