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

// statusReport is the machine-readable report of `dflow status`.
//
// Its targets field uses the shared targetView shape declared in report.go.
//
// Field order is the document order encoding/json produces, and it matches the
// keys the human-readable mode prints.
type statusReport struct {
	Branch           string       `json:"branch"`
	BranchType       string       `json:"branch_type"`
	Base             string       `json:"base"`
	Targets          []targetView `json:"targets"`
	WorkingTreeClean bool         `json:"working_tree_clean"`
	MergeInProgress  bool         `json:"merge_in_progress"`
	HasOrigin        bool         `json:"has_origin"`
}

// StatusCmd reports the state a non-interactive caller needs before it decides
// what to do next.
//
// It answers one question without mutating anything: which branch is checked
// out, whether dflow recognises it, where it would be finished, and whether Git
// is in a state where a finish could even run.
var StatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Report the current branch, its dflow type, and the resolved finish plan",
	Long: `Report the current dflow state without changing anything.

The report contains the checked out branch, the work type detected from the
configured prefixes, the resolved base branch, every finish target with its
effective merge mode, and whether the working tree is clean, a merge is already
in progress, and an 'origin' remote exists.

When the current branch does not match any configured dflow prefix the report
still succeeds: branch_type and base are empty and targets is an empty list,
which means "this is not a work branch".

Pass --json to receive the same report as a single machine-readable JSON
document. In that mode stdout carries the document and nothing else.`,
	Example: `  dflow status
  dflow status --json`,
	Args: cobra.NoArgs,
	// The format must be decided before the WithChecks wrapper inside RunE runs,
	// so a failure such as a missing .dflow.yaml is reported in the requested
	// format instead of as styled text. Cobra runs PersistentPreRun, then
	// PreRunE, then RunE.
	PreRunE: func(cmd *cobra.Command, args []string) error {
		jsonOutput, err := cmd.Flags().GetBool("json")
		if err != nil {
			return err
		}
		if jsonOutput {
			utils.SetFormat(utils.FormatJSON)
		}
		return nil
	},
	RunE: validators.WithChecks(false, func(cmd *cobra.Command, args []string) error {
		report, err := collectStatus()
		if err != nil {
			return err
		}

		if utils.CurrentFormat() == utils.FormatJSON {
			return utils.EmitJSON(report)
		}

		utils.Plain("branch: %s", report.Branch)
		utils.Plain("branch_type: %s", report.BranchType)
		utils.Plain("base: %s", report.Base)
		utils.Plain("targets: %s", formatStatusTargets(report.Targets))
		utils.Plain("working_tree_clean: %t", report.WorkingTreeClean)
		utils.Plain("merge_in_progress: %t", report.MergeInProgress)
		utils.Plain("has_origin: %t", report.HasOrigin)
		return nil
	}),
}

// collectStatus gathers the report without mutating the repository.
//
// A branch that matches no configured prefix is not an error here: status is a
// query, so "not a work branch" is a successful answer with empty fields. Any
// other failure, such as a repository whose configured merge mode is invalid,
// still fails loudly.
func collectStatus() (*statusReport, error) {
	cfg, err := utils.LoadConfig()
	if err != nil {
		return nil, err
	}

	branch, err := gitutils.CurrentBranch()
	if err != nil {
		return nil, err
	}

	clean, err := gitutils.IsWorkingTreeClean()
	if err != nil {
		return nil, err
	}

	report := &statusReport{
		Branch:           branch,
		Targets:          []targetView{},
		WorkingTreeClean: clean,
		MergeInProgress:  gitutils.MergeInProgress(),
		HasOrigin:        gitutils.HasOriginRemote(),
	}

	if !flow.IsWorkBranch(cfg, branch) {
		return report, nil
	}

	plan, err := flow.ResolveFinishPlan(cfg, branch)
	if err != nil {
		return nil, err
	}

	report.BranchType = string(plan.BranchType)
	report.Base = plan.Base
	report.Targets = targetViews(plan.Targets)

	return report, nil
}

// formatStatusTargets renders the target list for the human-readable report as
// `branch (merge_mode)` pairs, so it stays greppable and recognisably the same
// data the JSON document carries.
func formatStatusTargets(targets []targetView) string {
	if len(targets) == 0 {
		return "none"
	}

	parts := make([]string, 0, len(targets))
	for _, target := range targets {
		parts = append(parts, fmt.Sprintf("%s (%s)", target.Branch, target.MergeMode))
	}
	return strings.Join(parts, ", ")
}

func init() {
	StatusCmd.Flags().Bool("json", false, "Print the report as a single JSON document on stdout")
}
