// Package commands defines the end-user subcommands that make up the dflow CLI.
//
// The package contains interactive and non-interactive commands for initializing
// repositories, starting and finishing work branches, deleting branches, and
// managing local dflow metadata stored in Git config.
package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/yepizrene-devoost/dflow/cmd/gitutils"
	"github.com/yepizrene-devoost/dflow/cmd/utils"
	"github.com/yepizrene-devoost/dflow/pkg/agent"
	"github.com/yepizrene-devoost/dflow/pkg/flow"
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
// When the project already has a `.dflow.yaml`, `init` refuses to run so a
// hand-edited team contract is never overwritten by accident. Pass `--force` to
// regenerate it deliberately.
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
      - Features start from Develop and can be promoted to Develop and UAT
      - Releases start from UAT and can be promoted to Main and Develop
      - Bugfixes start from UAT and sync back to UAT and Develop
      - Hotfixes start from Main and sync back to Main, Develop, and UAT
    - Ensure the specified branches exist locally.
    - Ask whether to push those base branches to origin.

  The resulting .dflow.yaml is stored in the project root and used by all dflow commands.

  If .dflow.yaml already exists, init refuses to run instead of overwriting it.
  Pass --force to regenerate the file deliberately. --force only authorizes the
  overwrite: init still runs interactively and requires a terminal.

  Example:
    dflow init
    dflow init --force

  This command is meant to be run once per project when setting up the dflow branching model.`,
	Example: `  dflow init
  dflow init --force`,
	RunE: validators.WithChecks(true, func(cmd *cobra.Command, args []string) error {
		force, err := cmd.Flags().GetBool("force")
		if err != nil {
			return err
		}
		if !force {
			if err := validators.EnsureDflowNotInitialized(); err != nil {
				return err
			}
		}

		if !utils.IsInteractive() {
			return fmt.Errorf("dflow init is interactive and requires a terminal")
		}

		var mainBranch, developBranch, uatBranch string

		err = survey.AskOne(&survey.Input{Message: "Main branch name:", Default: "main"}, &mainBranch, survey.WithValidator(survey.Required))
		if err != nil {
			return err
		}

		err = survey.AskOne(&survey.Input{Message: "Development branch name:", Default: "develop"}, &developBranch, survey.WithValidator(survey.Required))
		if err != nil {
			return err
		}

		err = survey.AskOne(&survey.Input{Message: "UAT branch name:", Default: "uat"}, &uatBranch, survey.WithValidator(survey.Required))
		if err != nil {
			return err
		}

		// 🌟 merge modes explain
		utils.Plain("")
		utils.Icon("🔧", "Dflow supports two types of merge modes:")
		utils.Plain("   - manual: you open Pull Requests and merge via your platform (e.g. GitHub, GitLab).")
		utils.Plain("   - auto: dflow merges branches directly using Git commands (no PRs needed).")

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
			return err
		}

		var defaultMode, inverseMode string
		if mergeModeOption == "auto (direct merge from CLI)" {
			defaultMode = "auto"
			inverseMode = "manual"
		} else {
			defaultMode = "manual"
			inverseMode = "auto"
		}

		cfg := flow.Config{}
		cfg.Branches.Main = mainBranch
		cfg.Branches.Develop = developBranch
		cfg.Branches.Uat = uatBranch
		cfg.Branches.Features = "feature/"
		cfg.Branches.Releases = "release/"
		cfg.Branches.Hotfixes = "hotfix/"
		cfg.Branches.Bugfixes = "bugfix/"

		cfg.Flow.Feature.Base = developBranch
		cfg.Flow.Feature.FinishTargets = uniqueBranchNames(developBranch, uatBranch)
		cfg.Flow.Release.Base = uatBranch
		cfg.Flow.Release.FinishTargets = uniqueBranchNames(mainBranch, developBranch)
		cfg.Flow.Hotfix.Base = mainBranch
		cfg.Flow.Hotfix.FinishTargets = uniqueBranchNames(mainBranch, developBranch, uatBranch)
		cfg.Flow.Bugfix.Base = uatBranch
		cfg.Flow.Bugfix.FinishTargets = uniqueBranchNames(uatBranch, developBranch)

		cfg.Workflow.DefaultMergeMode = defaultMode
		cfg.Workflow.BranchRules = make(map[string]flow.WorkflowBranchRule)

		// 🎯 ask exceptions at the default mode
		var exceptionBranches []string
		allBranches := uniqueBranchNames(mainBranch, developBranch, uatBranch)

		err = survey.AskOne(&survey.MultiSelect{
			Message: fmt.Sprintf("Which branches should behave differently from the default '%s' mode?", defaultMode),
			Options: allBranches,
			Help:    fmt.Sprintf("Select the branches that require '%s' instead of the default '%s'", inverseMode, defaultMode),
		}, &exceptionBranches)
		if err != nil {
			return err
		}

		for _, branch := range allBranches {
			cfg.Workflow.BranchRules[branch] = flow.WorkflowBranchRule{MergeMode: defaultMode}
		}

		for _, branch := range exceptionBranches {
			cfg.Workflow.BranchRules[branch] = flow.WorkflowBranchRule{MergeMode: inverseMode}
		}

		if err := utils.SaveConfig(&cfg); err != nil {
			return err
		}
		utils.Success("Created .dflow.yaml")

		// 📋 print summary
		utils.Plain("")
		utils.Success("Merge behavior summary:")
		utils.Plain("   Default mode: %s", defaultMode)
		for _, branch := range allBranches {
			utils.Plain("   - %s: %s", branch, flow.GetMergeModeForBranch(&cfg, branch))
		}
		utils.Plain("")

		// 🌱 verify if base branches exists
		if err := gitutils.CheckOrCreateBranch(mainBranch); err != nil {
			return err
		}
		if err := gitutils.CheckOrCreateBranch(developBranch); err != nil {
			return err
		}
		if err := gitutils.CheckOrCreateBranch(uatBranch); err != nil {
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
			return err
		}

		if pushConfirm {
			if err := gitutils.PushBranch(mainBranch); err != nil {
				return fmt.Errorf("Failed to push '%s': %v", mainBranch, err)
			}
			if err := gitutils.PushBranch(developBranch); err != nil {
				return fmt.Errorf("Failed to push '%s': %v", developBranch, err)
			}
			if err := gitutils.PushBranch(uatBranch); err != nil {
				return fmt.Errorf("Failed to push '%s': %v", uatBranch, err)
			}
		}

		// 📝 generate agent workflow file
		var generateAgent bool
		if err := survey.AskOne(&survey.Confirm{
			Message: "Generate an agent workflow file for AI coding assistants?",
			Default: true,
		}, &generateAgent); err != nil {
			fmt.Fprintf(os.Stderr, "Prompt failed: %v\n", err)
			os.Exit(1)
		}

		if generateAgent {
			doc := agent.GenerateAgentDoc(&cfg)
			agentPath := ".agents/workflows/dflow.md"

			if err := os.MkdirAll(filepath.Dir(agentPath), 0755); err != nil {
				return fmt.Errorf("failed to create agent workflow directory: %w", err)
			}
			if err := os.WriteFile(agentPath, doc, 0644); err != nil {
				return fmt.Errorf("failed to write agent workflow: %w", err)
			}
			utils.Success("Generated agent workflow: %s", agentPath)

			// ensure AGENTS.md references the workflow file
			agentsRef := "## dflow Workflow\nRead `" + agentPath + "` for branch types, merge rules, and finish flow.\n"
			if err := ensureAgentsMdReference(agentsRef); err != nil {
				utils.Warn("Could not update AGENTS.md: %v", err)
			} else {
				utils.Success("Updated AGENTS.md with dflow workflow reference")
			}
		}

		utils.Icon("🎉", "dflow is ready! Use `dflow start` to begin a new branch.")
		return nil
	}),
}

func init() {
	InitCmd.Flags().Bool("force", false, "Regenerate .dflow.yaml even if the project is already initialized")
}

// ensureAgentsMdReference appends a section to AGENTS.md if it does not already
// contain the dflow workflow reference. If AGENTS.md does not exist, it is created.
func ensureAgentsMdReference(ref string) error {
	path := "AGENTS.md"

	var existing []byte
	if data, err := os.ReadFile(path); err == nil {
		existing = data
	} else if !os.IsNotExist(err) {
		return err
	}

	if existing != nil && strings.Contains(string(existing), "dflow.md") {
		return nil // already present
	}

	var content []byte
	if len(existing) > 0 {
		content = append(existing, '\n')
		content = append(content, []byte(ref)...)
	} else {
		content = []byte(ref)
	}

	return os.WriteFile(path, content, 0644)
}

func uniqueBranchNames(branches ...string) []string {
	var unique []string
	for _, branch := range branches {
		if branch == "" || slices.Contains(unique, branch) {
			continue
		}
		unique = append(unique, branch)
	}
	return unique
}
