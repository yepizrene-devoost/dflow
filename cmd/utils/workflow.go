package utils

import (
	"fmt"
	"strings"
)

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
		plan.Targets = append(plan.Targets, FinishTarget{
			Branch:    branch,
			MergeMode: GetMergeModeForBranch(cfg, branch),
		})
	}

	return plan, nil
}

// AutoTargets returns the branches that can be merged directly by dflow.
func (p FinishPlan) AutoTargets() []string {
	var branches []string
	for _, target := range p.Targets {
		if target.MergeMode == "auto" {
			branches = append(branches, target.Branch)
		}
	}
	return branches
}

// ManualTargets returns the branches that must be handled by PR/manual flow.
func (p FinishPlan) ManualTargets() []string {
	var branches []string
	for _, target := range p.Targets {
		if target.MergeMode == "manual" {
			branches = append(branches, target.Branch)
		}
	}
	return branches
}
