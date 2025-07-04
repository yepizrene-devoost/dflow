package initcmd

import (
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/spf13/cobra"
	"github.com/yepizrene-devoost/dflow/cmd/gitutils"
	"github.com/yepizrene-devoost/dflow/cmd/utils"
)

var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize your dflow branching configuration",
	Run: func(cmd *cobra.Command, args []string) {

		var mainBranch, developBranch, uatBranch string

		err := survey.AskOne(&survey.Input{Message: "Main branch name:", Default: "main"}, &mainBranch, survey.WithValidator(survey.Required))
		if err == terminal.InterruptErr {
			fmt.Println("\n🚫 Cancelled by user.")
			os.Exit(1)
		} else if err != nil {
			fmt.Println("❌ Error:", err)
			return
		}

		err = survey.AskOne(&survey.Input{Message: "Development branch name:", Default: "develop"}, &developBranch, survey.WithValidator(survey.Required))
		if err == terminal.InterruptErr {
			fmt.Println("\n🚫 Cancelled by user.")
			os.Exit(1)
		} else if err != nil {
			fmt.Println("❌ Error:", err)
			return
		}

		err = survey.AskOne(&survey.Input{Message: "UAT branch name:", Default: "uat"}, &uatBranch, survey.WithValidator(survey.Required))
		if err == terminal.InterruptErr {
			fmt.Println("\n🚫 Cancelled by user.")
			os.Exit(1)
		} else if err != nil {
			fmt.Println("❌ Error:", err)
			return
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
			return
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
	},
}
