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
	"github.com/yepizrene-devoost/dflow/pkg/flow"
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
//
//  1. Fails fast when no terminal is available and neither --push nor --no-push
//     was given, before loading config or touching the repository
//  2. Checks out the appropriate base branch
//  3. Pulls the latest changes from origin
//  4. Creates and checks out the new branch
//  5. Prompts the user to push the new branch to origin
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
  and based on the corresponding base branch defined in your .dflow.yaml configuration.

  Use --from to base the new branch on another work branch (chained/stacked branches) instead.
  Use --push or --no-push to publish non-interactively (useful for agents and scripts).`,
	Args: cobra.ArbitraryArgs,
	RunE: validators.WithChecks(false, func(cmd *cobra.Command, args []string) error {

		if len(args) < 2 {
			_ = cmd.Help()
			return fmt.Errorf("missing arguments: expected `dflow start <type> <name>`")
		}

		pushFlag, _ := cmd.Flags().GetBool("push")
		noPushFlag, _ := cmd.Flags().GetBool("no-push")
		if pushFlag && noPushFlag {
			return fmt.Errorf("--push and --no-push cannot be used together")
		}

		// Fail fast, before loading config or touching the repository. The publish
		// prompt can never be answered without a terminal, and a handler that
		// exits non-zero must not have created a branch first: a caller that sees
		// a failure and retries would otherwise hit "branch already exists".
		if !pushFlag && !noPushFlag && !utils.IsInteractive() {
			return utils.NonInteractiveError(
				"prompt to publish the new branch",
				"pass --push to publish or --no-push to keep it local",
			)
		}

		branchType, err := flow.ParseBranchType(args[0])
		if err != nil {
			return fmt.Errorf("Unknown type. Use: feat, release, hotfix, bugfix")
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
			return err
		}

		prefix, err := flow.GetBranchPrefix(cfg, branchType)
		if err != nil {
			return err
		}

		rule, err := flow.GetFlowRule(cfg, branchType)
		if err != nil {
			return err
		}

		fromBranch, _ := cmd.Flags().GetString("from")
		base := rule.Base
		if fromBranch != "" {
			base = fromBranch
			for _, primary := range []string{cfg.Branches.Main, cfg.Branches.Develop, cfg.Branches.Uat} {
				if primary != "" && fromBranch == primary {
					utils.Warn("'%s' is a primary branch; chained branches are usually based on another work branch", fromBranch)
					break
				}
			}
		}

		fullName := fmt.Sprintf("%s%s", prefix, branchName)

		if valid, reason := validators.IsValidGitBranchName(fullName); !valid {
			return fmt.Errorf("Invalid branch name '%s': %s", fullName, reason)
		}

		// Capture the caller's branch before any checkout, so a failure that has
		// not created the new branch can put them back where they started. The
		// contract for a non-zero exit is: the repository is either unchanged, or
		// the message states explicitly what was created and left behind. A
		// failure must never silently abandon the caller on another branch.
		originalBranch, err := gitutils.CurrentBranch()
		if err != nil {
			return err
		}

		restoreOriginalBranch := func(cause error) error {
			if originalBranch == "" {
				return cause
			}
			if err := gitutils.CheckoutExistingBranch(originalBranch); err != nil {
				current, currentErr := gitutils.CurrentBranch()
				if currentErr != nil || current == "" {
					current = "an unknown branch"
				}
				// Keep the original error as the reported failure: the restore is a
				// best-effort repair, and the warning names where the caller ended up.
				utils.Warn("Could not restore the original branch '%s' (%v); the caller is now on '%s'.", originalBranch, err, current)
			}
			return cause
		}

		if fromBranch != "" {
			if err := gitutils.CheckoutBranch(fromBranch); err != nil {
				return restoreOriginalBranch(err)
			}
		} else {
			if err := gitutils.Checkout(base); err != nil {
				return restoreOriginalBranch(fmt.Errorf("Could not checkout base branch '%s'", base))
			}

			if err := gitutils.Pull(); err != nil {
				return restoreOriginalBranch(fmt.Errorf("Failed to pull latest changes from '%s'", base))
			}
		}

		if err := gitutils.CheckoutNew(fullName); err != nil {
			// The new branch was not created, so the repository can still be left
			// exactly as the caller found it.
			return restoreOriginalBranch(fmt.Errorf("Failed to create branch '%s'", fullName))
		}

		utils.Success("Created and switched to branch '%s' from '%s'", fullName, base)

		pushBranch := pushFlag
		if !pushFlag && !noPushFlag {
			if err := survey.AskOne(&survey.Confirm{
				Message: fmt.Sprintf("Do you want to publish '%s' to origin?", fullName),
				Default: true,
			}, &pushBranch); err != nil {
				// The branch is already created, so the primary work succeeded.
				// Report the skipped side effect honestly and exit 0; returning an
				// error here would claim failure for work that partly happened.
				utils.Warn("Branch '%s' was created, but the publish prompt could not be answered; the push was skipped.", fullName)
				utils.Info("Publish it later with: git push -u origin %s", fullName)
				return nil
			}
		}

		if pushBranch {
			if err := gitutils.PushBranch(fullName); err != nil {
				// The branch exists now: never delete it, and say so, so a caller
				// that sees a non-zero exit does not blindly retry into an
				// "already exists" failure. PushBranch already names the branch and
				// the operation, so only the consequence is added here.
				return fmt.Errorf("%v; the branch '%s' was created and remains", err, fullName)
			}
		}

		return nil
	}),
}

func init() {
	StartCmd.Flags().String("from", "", "existing work branch to base the new branch on (chained/stacked branch)")
	StartCmd.Flags().Bool("push", false, "push the new branch to origin without prompting")
	StartCmd.Flags().Bool("no-push", false, "skip pushing the new branch to origin")

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
