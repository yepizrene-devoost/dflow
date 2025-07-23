// Package commands provides the CLI subcommands for dflow, enabling users to manage
// Git branching workflows using a consistent, configurable model.
//
// This includes project-local configuration commands under `dflow config`,
// allowing users to set and retrieve metadata such as author name and email
// for use in changelogs and other automated processes.
package commands

import (
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/yepizrene-devoost/dflow/cmd/gitutils"
	"github.com/yepizrene-devoost/dflow/cmd/utils"
	"github.com/yepizrene-devoost/dflow/pkg/validators"
)

// InitCmd initializes the dflow configuration for the current Git project.
//
// This command is intended to be run once per project and will guide the user through
// an interactive setup process to generate a `.dflow.yaml` file with the following:
//
//   - Names for the main, develop, and UAT branches.
//   - Default merge behavior (manual via Pull Requests or automatic).
//   - Optional exceptions for specific branches to use a different merge mode.
//   - Prefixes for feature, release, bugfix and hotfix branches.
//   - Flow rules for each type of branch (feature, release, bugfix, hotfix).
//
// It also ensures the specified base branches exist locally, offering to create them
// if missing, and provides an option to push them to the remote origin.
//
// The `.dflow.yaml` file is stored at the root of the repository and is used by all
// subsequent dflow commands (`start`, `config`, `delete`, etc).
//
// Example:
//
//	dflow init
//
// This command is interactive and requires a terminal.
var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize your dflow branching configuration",
	Long: `Initialize your dflow branching configuration and generate a .dflow.yaml file.

  This interactive setup will guide you through the following steps:

    - Prompt for the names of your main, develop, and UAT branches.
    - Choose your default merge mode and any branch-specific exceptions.
    - Create a .dflow.yaml file with default prefixes:
      - feature/  → for feature branches
      - release/  → for release branches
      - hotfix/   → for hotfix branches
      - bugfix/   → for bugfix branches
    - Set flow rules:
      - Features start from UAT and merge to Develop
      - Releases start from UAT
      - Bugfixes start from UAT
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
		if err != nil {
			utils.Error(err.Error())
			return nil
		}

		err = survey.AskOne(&survey.Input{Message: "Development branch name:", Default: "develop"}, &developBranch, survey.WithValidator(survey.Required))
		if err != nil {
			utils.Error(err.Error())
			return nil
		}

		err = survey.AskOne(&survey.Input{Message: "UAT branch name:", Default: "uat"}, &uatBranch, survey.WithValidator(survey.Required))
		if err != nil {
			utils.Error(err.Error())
			return nil
		}

		// 🌟 merge modes explain
		fmt.Println("\n🔧 Dflow supports two types of merge modes:")
		fmt.Println("   - manual: you open Pull Requests and merge via your platform (e.g. GitHub, GitLab).")
		fmt.Println("   - auto: dflow merges branches directly using Git commands (no PRs needed).")

		var mergeModeOption string
		err = survey.AskOne(&survey.Select{
			Message: "How do you manage merges by default in this project?",
			Options: []string{
				"manual (via Pull Requests)",
				"auto (direct merge from CLI)",
			},
			Default: "manual (via Pull Requests)",
		}, &mergeModeOption)
		if err != nil {
			utils.Error(err.Error())
			return nil
		}

		var defaultMode, inverseMode string
		if mergeModeOption == "auto (direct merge from CLI)" {
			defaultMode = "auto"
			inverseMode = "manual"
		} else {
			defaultMode = "manual"
			inverseMode = "auto"
		}

		cfg := utils.Config{}
		cfg.Branches.Main = mainBranch
		cfg.Branches.Develop = developBranch
		cfg.Branches.Uat = uatBranch
		cfg.Branches.Features = "feature/"
		cfg.Branches.Releases = "release/"
		cfg.Branches.Hotfixes = "hotfix/"
		cfg.Branches.Bugfixes = "bugfix/"

		cfg.Flow.FeatureBase = uatBranch
		cfg.Flow.FeatureMerge = developBranch
		cfg.Flow.ReleaseBase = uatBranch
		cfg.Flow.HotfixBase = mainBranch
		cfg.Flow.BugfixBase = uatBranch

		cfg.Workflow.DefaultMergeMode = defaultMode
		cfg.Workflow.BranchRules = make(map[string]string)

		// 🎯 ask exceptions at the default mode
		var exceptionBranches []string
		allBranches := []string{mainBranch, developBranch, uatBranch}

		err = survey.AskOne(&survey.MultiSelect{
			Message: fmt.Sprintf("Which branches should behave differently from the default '%s' mode?", defaultMode),
			Options: allBranches,
			Help:    fmt.Sprintf("Select the branches that require '%s' instead of the default '%s'", inverseMode, defaultMode),
		}, &exceptionBranches)
		if err != nil {
			utils.Error(err.Error())
			return nil
		}

		for _, branch := range exceptionBranches {
			cfg.Workflow.BranchRules[branch] = inverseMode
		}

		if err := utils.SaveConfig(&cfg); err != nil {
			utils.Error(err.Error())
			return nil
		}
		utils.Success("Created .dflow.yaml")

		// 📋 print summary
		fmt.Println("\n✅ Merge behavior summary:")
		fmt.Printf("   Default mode: %s\n", defaultMode)
		if len(exceptionBranches) > 0 {
			fmt.Printf("   Exceptions (%s): %v\n", inverseMode, exceptionBranches)
		} else {
			fmt.Println("   No branch exceptions defined.")
		}
		fmt.Println()

		// 🌱 verify if base branches exists
		if err := gitutils.CheckOrCreateBranch(mainBranch); err != nil {
			utils.Error(err.Error())
			return err
		}
		if err := gitutils.CheckOrCreateBranch(developBranch); err != nil {
			utils.Error(err.Error())
			return err
		}
		if err := gitutils.CheckOrCreateBranch(uatBranch); err != nil {
			utils.Error(err.Error())
			return err
		}

		// 🚀 confirm push of branches
		var pushConfirm bool

		if err := survey.AskOne(&survey.Confirm{
			Message: "Do you want to push the base branches to 'origin'?",
			Default: true,
		}, &pushConfirm); err != nil {
			fmt.Fprintf(os.Stderr, "Prompt failed: %v\n", err)
			os.Exit(1)
		}

		if err != nil {
			utils.Error(err.Error())
			return nil
		}

		if pushConfirm {
			if err := gitutils.PushBranch(mainBranch); err != nil {
				utils.Error("Failed to push '%s': %v", mainBranch, err)
				return err
			}
			if err := gitutils.PushBranch(developBranch); err != nil {
				utils.Error("Failed to push '%s': %v", developBranch, err)
				return err
			}
			if err := gitutils.PushBranch(uatBranch); err != nil {
				utils.Error("Failed to push '%s': %v", uatBranch, err)
				return err
			}
		}

		utils.Success("dflow is ready! Use `dflow start` to begin a new branch.", "🎉")
		return nil
	}),
}
