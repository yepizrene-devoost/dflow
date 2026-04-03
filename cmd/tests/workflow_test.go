package tests

import (
	"testing"

	"github.com/yepizrene-devoost/dflow/cmd/utils"
)

func TestParseBranchTypeAliases(t *testing.T) {
	testCases := []struct {
		input string
		want  utils.BranchType
	}{
		{input: "feat", want: utils.BranchTypeFeature},
		{input: "feature", want: utils.BranchTypeFeature},
		{input: "release", want: utils.BranchTypeRelease},
		{input: "hot", want: utils.BranchTypeHotfix},
		{input: "hotfix", want: utils.BranchTypeHotfix},
		{input: "fix", want: utils.BranchTypeHotfix},
		{input: "bug", want: utils.BranchTypeBugfix},
		{input: "bugfix", want: utils.BranchTypeBugfix},
	}

	for _, tc := range testCases {
		got, err := utils.ParseBranchType(tc.input)
		if err != nil {
			t.Fatalf("ParseBranchType(%q) returned error: %v", tc.input, err)
		}
		if got != tc.want {
			t.Fatalf("ParseBranchType(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestResolveFinishPlanWithDistinctTargets(t *testing.T) {
	cfg := &utils.Config{}
	cfg.Branches.Main = "main"
	cfg.Branches.Develop = "develop"
	cfg.Branches.Uat = "uat"
	cfg.Branches.Features = "feature/"
	cfg.Branches.Releases = "release/"
	cfg.Branches.Hotfixes = "hotfix/"
	cfg.Branches.Bugfixes = "bugfix/"

	cfg.Flow.Release.Base = "develop"
	cfg.Flow.Release.FinishTargets = []string{"uat", "main", "develop"}

	cfg.Workflow.DefaultMergeMode = "manual"
	cfg.Workflow.BranchRules = map[string]utils.WorkflowBranchRule{
		"develop": {MergeMode: "auto"},
		"uat":     {MergeMode: "auto"},
		"main":    {MergeMode: "manual"},
	}

	plan, err := utils.ResolveFinishPlan(cfg, "release/1.2.0")
	if err != nil {
		t.Fatalf("ResolveFinishPlan returned error: %v", err)
	}

	if plan.BranchType != utils.BranchTypeRelease {
		t.Fatalf("expected branch type %q, got %q", utils.BranchTypeRelease, plan.BranchType)
	}

	if plan.Base != "develop" {
		t.Fatalf("expected base %q, got %q", "develop", plan.Base)
	}

	autoTargets := plan.AutoTargets()
	if len(autoTargets) != 2 || autoTargets[0] != "uat" || autoTargets[1] != "develop" {
		t.Fatalf("unexpected auto targets: %v", autoTargets)
	}

	manualTargets := plan.ManualTargets()
	if len(manualTargets) != 1 || manualTargets[0] != "main" {
		t.Fatalf("unexpected manual targets: %v", manualTargets)
	}
}

func TestResolveFinishPlanDeduplicatesTargets(t *testing.T) {
	cfg := &utils.Config{}
	cfg.Branches.Main = "main"
	cfg.Branches.Develop = "develop"
	cfg.Branches.Uat = "develop"
	cfg.Branches.Features = "feature/"
	cfg.Branches.Releases = "release/"
	cfg.Branches.Hotfixes = "hotfix/"
	cfg.Branches.Bugfixes = "bugfix/"

	cfg.Flow.Release.Base = "develop"
	cfg.Flow.Release.FinishTargets = []string{"develop", "main", "develop"}

	cfg.Workflow.DefaultMergeMode = "manual"
	cfg.Workflow.BranchRules = map[string]utils.WorkflowBranchRule{
		"develop": {MergeMode: "auto"},
		"main":    {MergeMode: "manual"},
	}

	plan, err := utils.ResolveFinishPlan(cfg, "release/1.2.0")
	if err != nil {
		t.Fatalf("ResolveFinishPlan returned error: %v", err)
	}

	if len(plan.Targets) != 2 {
		t.Fatalf("expected 2 unique finish targets, got %d (%v)", len(plan.Targets), plan.Targets)
	}
}

func TestDetectBranchTypeRejectsUnknownBranch(t *testing.T) {
	cfg := &utils.Config{}
	cfg.Branches.Features = "feature/"
	cfg.Branches.Releases = "release/"
	cfg.Branches.Hotfixes = "hotfix/"
	cfg.Branches.Bugfixes = "bugfix/"

	if _, err := utils.DetectBranchType(cfg, "develop"); err == nil {
		t.Fatalf("expected error for branch without dflow prefix")
	}
}
