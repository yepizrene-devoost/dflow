package commands

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/yepizrene-devoost/dflow/cmd/gitutils"
	"github.com/yepizrene-devoost/dflow/cmd/utils"
	"github.com/yepizrene-devoost/dflow/pkg/validators"
)

// FinishCmd merges the current dflow work branch into all configured auto targets
// and reports any remaining manual targets that must be handled through PR flow.
var FinishCmd = &cobra.Command{
	Use:   "finish",
	Short: "Finish the current dflow branch using the configured merge rules",
	Long: `Finish the current dflow branch by resolving its configured finish targets.

For each target branch configured with merge_mode=auto, dflow will:
  - fetch updates from origin
  - checkout and sync the target branch
  - merge the current work branch into that target
  - push the updated target branch to origin

Targets configured with merge_mode=manual are not merged automatically.
Instead, dflow reports them so the team can complete them through the usual
pull request or manual review flow.`,
	Example: `  dflow finish
  dflow finish --dry-run`,
	Args: cobra.NoArgs,
	RunE: validators.WithChecks(false, func(cmd *cobra.Command, args []string) error {
		dryRun, err := cmd.Flags().GetBool("dry-run")
		if err != nil {
			utils.Error(err.Error())
			return nil
		}

		if err := gitutils.EnsureWorkingTreeClean(); err != nil {
			utils.Error(err.Error())
			return nil
		}

		if gitutils.MergeInProgress() {
			utils.Error("A merge is already in progress. Resolve or abort it before running `dflow finish`.")
			return nil
		}

		cfg, err := utils.LoadConfig()
		if err != nil {
			utils.Error(err.Error())
			return nil
		}

		currentBranch, err := gitutils.CurrentBranch()
		if err != nil {
			utils.Error(err.Error())
			return nil
		}

		plan, err := utils.ResolveFinishPlan(cfg, currentBranch)
		if err != nil {
			utils.Error(err.Error())
			return nil
		}

		autoTargets := plan.AutoTargets()
		manualTargets := plan.ManualTargets()

		utils.Info("Finishing branch '%s' (%s)", plan.CurrentBranch, plan.BranchType)
		utils.Info("Auto targets: %s", formatBranchList(autoTargets))
		utils.Info("Manual targets: %s", formatBranchList(manualTargets))

		if len(autoTargets) == 0 {
			utils.Warn("No auto targets are configured for '%s'.", plan.CurrentBranch)
			if len(manualTargets) > 0 {
				utils.Info("Manual follow-up required for: %s", strings.Join(manualTargets, ", "))
			}
			return nil
		}

		returnBranch := plan.Base
		if returnBranch == "" {
			returnBranch = plan.CurrentBranch
		}

		if dryRun {
			utils.Info("Dry run enabled. No branches will be checked out, merged, or pushed.")
			utils.Info("Would return to branch: %s", returnBranch)
			for _, target := range autoTargets {
				utils.Info("Would merge '%s' into '%s' and push the target branch.", plan.CurrentBranch, target)
			}
			if len(manualTargets) > 0 {
				utils.Warn("Manual follow-up still required for: %s", strings.Join(manualTargets, ", "))
			}
			return nil
		}

		if err := gitutils.FetchOrigin(); err != nil {
			utils.Error(err.Error())
			return nil
		}

		var mergedTargets []string
		allSucceeded := false
		defer func() {
			if allSucceeded {
				if err := gitutils.CheckoutExistingBranch(returnBranch); err != nil {
					utils.Warn("Finished all auto merges, but could not switch to '%s': %v", returnBranch, err)
				}
			}
		}()

		for _, target := range autoTargets {
			utils.Info("Processing auto target '%s'...", target)

			if err := gitutils.PullBranch(target); err != nil {
				utils.Error(err.Error())
				return nil
			}

			if err := gitutils.MergeBranchIntoCurrent(plan.CurrentBranch); err != nil {
				if gitutils.MergeInProgress() {
					utils.Error("Merge conflict while merging '%s' into '%s'. Resolve or abort the merge on '%s' and try again.", plan.CurrentBranch, target, target)
					return nil
				}

				utils.Error(err.Error())
				return nil
			}

			if err := gitutils.PushBranchUpdate(target); err != nil {
				utils.Error(err.Error())
				return nil
			}

			mergedTargets = append(mergedTargets, target)
			utils.Success("Merged '%s' into '%s'", plan.CurrentBranch, target)
		}

		allSucceeded = true

		if len(mergedTargets) > 0 {
			utils.Success("Finished auto merges for '%s': %s", plan.CurrentBranch, strings.Join(mergedTargets, ", "))
		}
		utils.Info("Current branch after finish: %s", returnBranch)
		if len(manualTargets) > 0 {
			utils.Warn("Manual follow-up still required for: %s", strings.Join(manualTargets, ", "))
		}

		return nil
	}),
}

func formatBranchList(branches []string) string {
	if len(branches) == 0 {
		return "none"
	}
	return strings.Join(branches, ", ")
}

func init() {
	FinishCmd.Flags().Bool("dry-run", false, "Show the finish plan without performing any merge or push")
}
