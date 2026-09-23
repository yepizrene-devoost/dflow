package flow

import (
	"fmt"
	"strings"
)

// Merge modes. These are the only values workflow.default_merge_mode and
// workflow.branch_rules.<branch>.merge_mode may take.
const (
	// MergeModeAuto means dflow merges and pushes the target itself.
	MergeModeAuto = "auto"
	// MergeModeManual means the target is left for PR/manual review flow.
	MergeModeManual = "manual"
)

// IsValidMergeMode reports whether mode is one of the supported merge modes.
func IsValidMergeMode(mode string) bool {
	return mode == MergeModeAuto || mode == MergeModeManual
}

// BranchType identifies the supported dflow working branch categories.
type BranchType string

const (
	BranchTypeFeature BranchType = "feature"
	BranchTypeRelease BranchType = "release"
	BranchTypeHotfix  BranchType = "hotfix"
	BranchTypeBugfix  BranchType = "bugfix"
)

// FinishTarget describes one destination branch together with its merge mode.
type FinishTarget struct {
	Branch    string
	MergeMode string
}

// FinishPlan contains the resolved finish configuration for a working branch.
type FinishPlan struct {
	CurrentBranch string
	BranchType    BranchType
	Base          string
	Targets       []FinishTarget
}

// ParseBranchType converts CLI aliases into canonical branch types.
func ParseBranchType(input string) (BranchType, error) {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "feat", "feature":
		return BranchTypeFeature, nil
	case "release":
		return BranchTypeRelease, nil
	case "hot", "hotfix", "fix":
		return BranchTypeHotfix, nil
	case "bug", "bugfix":
		return BranchTypeBugfix, nil
	default:
		return "", fmt.Errorf("unknown branch type %q", input)
	}
}

// DetectBranchType infers the branch type from the configured branch prefixes.
func DetectBranchType(cfg *Config, branchName string) (BranchType, error) {
	type branchMatcher struct {
		branchType BranchType
		prefix     string
	}

	matchers := []branchMatcher{
		{branchType: BranchTypeFeature, prefix: cfg.Branches.Features},
		{branchType: BranchTypeRelease, prefix: cfg.Branches.Releases},
		{branchType: BranchTypeHotfix, prefix: cfg.Branches.Hotfixes},
		{branchType: BranchTypeBugfix, prefix: cfg.Branches.Bugfixes},
	}

	for _, matcher := range matchers {
		if matcher.prefix != "" && strings.HasPrefix(branchName, matcher.prefix) {
			return matcher.branchType, nil
		}
	}

	return "", fmt.Errorf("branch %q does not match any configured dflow prefix", branchName)
}

// IsWorkBranch reports whether branch matches one of the configured dflow
// prefixes, so dflow can plan a finish for it.
//
// It is the predicate form of DetectBranchType: a caller that only asks
// "is this a work branch?" states that question directly instead of
// discarding a detected type. It delegates to DetectBranchType so the two can
// never drift apart.
func IsWorkBranch(cfg *Config, branchName string) bool {
	_, err := DetectBranchType(cfg, branchName)
	return err == nil
}

// GetBranchPrefix returns the configured prefix for the given branch type.
func GetBranchPrefix(cfg *Config, branchType BranchType) (string, error) {
	switch branchType {
	case BranchTypeFeature:
		return cfg.Branches.Features, nil
	case BranchTypeRelease:
		return cfg.Branches.Releases, nil
	case BranchTypeHotfix:
		return cfg.Branches.Hotfixes, nil
	case BranchTypeBugfix:
		return cfg.Branches.Bugfixes, nil
	default:
		return "", fmt.Errorf("unsupported branch type %q", branchType)
	}
}

// GetFlowRule returns the configured flow rule for a canonical branch type.
func GetFlowRule(cfg *Config, branchType BranchType) (BranchFlowRule, error) {
	switch branchType {
	case BranchTypeFeature:
		return cfg.Flow.Feature, nil
	case BranchTypeRelease:
		return cfg.Flow.Release, nil
	case BranchTypeHotfix:
		return cfg.Flow.Hotfix, nil
	case BranchTypeBugfix:
		return cfg.Flow.Bugfix, nil
	default:
		return BranchFlowRule{}, fmt.Errorf("unsupported branch type %q", branchType)
	}
}

// ResolveFinishPlan builds a finish plan for the given working branch.
func ResolveFinishPlan(cfg *Config, currentBranch string) (*FinishPlan, error) {
	branchType, err := DetectBranchType(cfg, currentBranch)
	if err != nil {
		return nil, err
	}

	rule, err := GetFlowRule(cfg, branchType)
	if err != nil {
		return nil, err
	}

	targetNames := uniqueStrings(rule.FinishTargets)
	if len(targetNames) == 0 {
		return nil, fmt.Errorf("branch type %q has no configured finish targets", branchType)
	}

	plan := &FinishPlan{
		CurrentBranch: currentBranch,
		BranchType:    branchType,
		Base:          rule.Base,
		Targets:       make([]FinishTarget, 0, len(targetNames)),
	}

	for _, branch := range targetNames {
		mergeMode := GetMergeModeForBranch(cfg, branch)
		if !IsValidMergeMode(mergeMode) {
			return nil, invalidMergeModeError(branch, mergeMode)
		}

		plan.Targets = append(plan.Targets, FinishTarget{
			Branch:    branch,
			MergeMode: mergeMode,
		})
	}

	return plan, nil
}

// invalidMergeModeError describes an effective merge mode that dflow cannot act
// on.
//
// An empty value is not a typo but missing configuration, so that message points
// at both places the value could be set. A non-empty value is mistyped, so the
// message names the offending value and the accepted ones.
func invalidMergeModeError(branch, mergeMode string) error {
	if mergeMode == "" {
		return fmt.Errorf(
			"branch %q has merge mode %q, which is unset: set workflow.default_merge_mode or the workflow.branch_rules entry for %q to %q or %q",
			branch, mergeMode, branch, MergeModeAuto, MergeModeManual,
		)
	}

	return fmt.Errorf(
		"branch %q has invalid merge mode %q: use %q or %q",
		branch, mergeMode, MergeModeAuto, MergeModeManual,
	)
}

// AutoTargets returns the branches that can be merged directly by dflow.
func (p FinishPlan) AutoTargets() []string {
	var branches []string
	for _, target := range p.Targets {
		if target.MergeMode == MergeModeAuto {
			branches = append(branches, target.Branch)
		}
	}
	return branches
}

// ManualTargets returns the branches that must be handled by PR/manual flow.
func (p FinishPlan) ManualTargets() []string {
	var branches []string
	for _, target := range p.Targets {
		if target.MergeMode == MergeModeManual {
			branches = append(branches, target.Branch)
		}
	}
	return branches
}
