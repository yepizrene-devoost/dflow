package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yepizrene-devoost/dflow/cmd/gitutils"
	"github.com/yepizrene-devoost/dflow/cmd/utils"
	"github.com/yepizrene-devoost/dflow/pkg/flow"
	"github.com/yepizrene-devoost/dflow/pkg/validators"
)

// FinishCmd merges the current dflow work branch into all configured auto targets
// and reports any remaining manual targets that must be handled through PR flow.
var FinishCmd = &cobra.Command{
	Use:   "finish",
	Short: "Finish the current dflow branch using the configured merge rules",
	Long: `Finish the current dflow branch by resolving its configured finish targets.

Before running any merge, dflow requires a clean working tree and verifies that
no merge is already in progress.

For each target branch configured with merge_mode=auto, dflow will:
  - fetch updates from origin
  - checkout and sync the target branch
  - merge the current work branch into that target
  - push the updated target branch to origin

Targets configured with merge_mode=manual are not merged automatically.
Instead, dflow reports them so the team can complete them through the usual
pull request or manual review flow.

After all automatic merges succeed, dflow switches back to the configured base
branch for the finished work type. The source branch is not deleted automatically
unless you explicitly pass --delete and no manual targets remain.

Use --dry-run to inspect the finish plan without fetching, merging, or pushing.`,
	Example: `  dflow finish
  dflow finish --dry-run
  dflow finish --delete
  # Merges auto targets, returns to the configured base branch, and deletes the source branch when no manual targets remain`,
	Args: cobra.NoArgs,
	// The format is decided before the WithChecks wrapper inside RunE runs, so a
	// pre-handler failure is reported in the requested format. --json only makes
	// sense for a plan preview: a mutating finish has no JSON report, so the
	// combination is rejected here before anything is loaded or touched.
	PreRunE: func(cmd *cobra.Command, args []string) error {
		jsonOutput, err := cmd.Flags().GetBool("json")
		if err != nil {
			return err
		}
		if !jsonOutput {
			return nil
		}

		// Set the format first: a caller that passed --json must receive the
		// rejection as a JSON document, not as styled text.
		utils.SetFormat(utils.FormatJSON)

		dryRun, err := cmd.Flags().GetBool("dry-run")
		if err != nil {
			return err
		}
		if !dryRun {
			return fmt.Errorf("--json requires --dry-run: a mutating finish has no JSON report, so --json would only hide the plan. Run `dflow finish --dry-run --json`")
		}

		return nil
	},
	RunE: validators.WithChecks(false, func(cmd *cobra.Command, args []string) error {
		dryRun, err := cmd.Flags().GetBool("dry-run")
		if err != nil {
			return err
		}
		deleteBranch, err := cmd.Flags().GetBool("delete")
		if err != nil {
			return err
		}

		if err := gitutils.EnsureWorkingTreeClean(); err != nil {
			return err
		}

		if gitutils.MergeInProgress() {
			return fmt.Errorf("A merge is already in progress. Resolve or abort it before running `dflow finish`.")
		}

		cfg, err := utils.LoadConfig()
		if err != nil {
			return err
		}

		currentBranch, err := gitutils.CurrentBranch()
		if err != nil {
			return err
		}

		plan, err := flow.ResolveFinishPlan(cfg, currentBranch)
		if err != nil {
			return err
		}

		autoTargets := plan.AutoTargets()
		manualTargets := plan.ManualTargets()

		returnBranch := plan.Base
		if returnBranch == "" {
			returnBranch = plan.CurrentBranch
		}

		// In JSON mode the resolved plan is the only thing stdout carries, so it
		// is emitted before any human line. It is only reachable with --dry-run
		// (enforced in PreRunE), so this path never fetches, merges or pushes.
		if utils.CurrentFormat() == utils.FormatJSON {
			return utils.EmitJSON(finishPlanReport{
				CurrentBranch:   plan.CurrentBranch,
				BranchType:      string(plan.BranchType),
				Base:            plan.Base,
				ReturnBranch:    returnBranch,
				Targets:         targetViews(plan.Targets),
				AutoTargets:     nonNilStrings(autoTargets),
				ManualTargets:   nonNilStrings(manualTargets),
				DeleteRequested: deleteBranch,
				DryRun:          true,
			})
		}

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

		if dryRun {
			utils.Info("Dry run enabled. No branches will be checked out, merged, or pushed.")
			utils.Info("Would return to branch: %s", returnBranch)
			for _, target := range autoTargets {
				utils.Info("Would merge '%s' into '%s' and push the target branch.", plan.CurrentBranch, target)
			}
			if deleteBranch {
				if len(manualTargets) == 0 {
					utils.Info("Would delete '%s' locally and remotely after a successful finish.", plan.CurrentBranch)
				} else {
					utils.Warn("Would skip deleting '%s' because manual follow-up is still required.", plan.CurrentBranch)
				}
			}
			if len(manualTargets) > 0 {
				utils.Warn("Manual follow-up still required for: %s", strings.Join(manualTargets, ", "))
			}
			return nil
		}

		if err := gitutils.FetchOrigin(); err != nil {
			return err
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
				return err
			}

			if err := gitutils.MergeBranchIntoCurrent(plan.CurrentBranch); err != nil {
				if gitutils.MergeInProgress() {
					return fmt.Errorf("Merge conflict while merging '%s' into '%s'. Resolve or abort the merge on '%s' and try again.", plan.CurrentBranch, target, target)
				}

				return err
			}

			if err := gitutils.PushBranchUpdate(target); err != nil {
				return err
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
		if deleteBranch {
			if len(manualTargets) > 0 {
				utils.Warn("Skipping delete for '%s' because manual follow-up is still required.", plan.CurrentBranch)
				return nil
			}

			if err := gitutils.Delete(plan.CurrentBranch); err != nil {
				return err
			}
			utils.Success("Deleted finished branch '%s'", plan.CurrentBranch)
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

// finishPlanReport is the machine-readable preview of `finish --dry-run --json`.
//
// It mirrors the human dry-run report: the branch being finished, where it
// would land, its targets with their effective merge modes, and whether a delete
// was requested. Field order is the document order encoding/json produces.
type finishPlanReport struct {
	CurrentBranch   string       `json:"current_branch"`
	BranchType      string       `json:"branch_type"`
	Base            string       `json:"base"`
	ReturnBranch    string       `json:"return_branch"`
	Targets         []targetView `json:"targets"`
	AutoTargets     []string     `json:"auto_targets"`
	ManualTargets   []string     `json:"manual_targets"`
	DeleteRequested bool         `json:"delete_requested"`
	DryRun          bool         `json:"dry_run"`
}

// nonNilStrings returns a non-nil empty slice for a nil input, so an empty
// target list marshals as `[]` rather than `null`.
func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func init() {
	FinishCmd.Flags().Bool("dry-run", false, "Show the finish plan without performing any merge or push")
	FinishCmd.Flags().Bool("delete", false, "Delete the finished branch locally and remotely after a successful finish when no manual targets remain")
	FinishCmd.Flags().Bool("json", false, "Print the --dry-run plan as a single JSON document on stdout; requires --dry-run")
}
