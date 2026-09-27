package commands

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/yepizrene-devoost/dflow/pkg/flow"
)

func TestInitApplyRejectsInvalidPrimaryBranchBeforeMutation(t *testing.T) {
	draft := validInitDraft()
	draft.config.Branches.Develop = "bad name"
	draft.branches = []string{"main", "bad name", "uat"}
	ops, calls := recordingInitOperations()
	ops.validateBranch = func(branch string) error {
		*calls = append(*calls, "validate:"+branch)
		if branch == "bad name" {
			return errors.New("fatal: 'bad name' is not a valid branch name")
		}
		return nil
	}

	err := applyInitDraft(draft, ops)
	if err == nil || !strings.Contains(err.Error(), `invalid development branch "bad name"`) {
		t.Fatalf("applyInitDraft() error = %v, want invalid development branch", err)
	}
	assertNoInitMutations(t, *calls)
}

func TestInitApplyValidatesConfigBeforeMutation(t *testing.T) {
	draft := validInitDraft()
	draft.config.Branches.Releases = draft.config.Branches.Features
	ops, calls := recordingInitOperations()

	err := applyInitDraft(draft, ops)
	if err == nil || !strings.Contains(err.Error(), "validate onboarding configuration") {
		t.Fatalf("applyInitDraft() error = %v, want configuration validation error", err)
	}
	assertNoInitMutations(t, *calls)
}

func TestInitApplyValidatesAllPrimaryRefsBeforeMutation(t *testing.T) {
	draft := validInitDraft()
	ops, calls := recordingInitOperations()

	if err := applyInitDraft(draft, ops); err != nil {
		t.Fatalf("applyInitDraft() error = %v", err)
	}
	wantPrefix := []string{"validate:main", "validate:develop", "validate:uat", "save"}
	if got := (*calls)[:len(wantPrefix)]; !reflect.DeepEqual(got, wantPrefix) {
		t.Fatalf("initial calls = %v, want validation before first mutation %v", got, wantPrefix)
	}
}

func TestInitApplyUsesCollectedAnswersBeforeLegacyMutations(t *testing.T) {
	draft := validInitDraft()
	draft.push = true
	draft.generateAgent = true
	ops, calls := recordingInitOperations()

	if err := applyInitDraft(draft, ops); err != nil {
		t.Fatalf("applyInitDraft() error = %v", err)
	}
	want := []string{
		"validate:main", "validate:develop", "validate:uat",
		"save",
		"ensure:main", "ensure:develop", "ensure:uat",
		"push:main", "push:develop", "push:uat",
		"agent",
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("calls = %v, want collected decisions applied in legacy order %v", *calls, want)
	}
}

func TestInitBranchValidationUsesGitRefRules(t *testing.T) {
	for _, tc := range []struct {
		name    string
		branch  string
		wantErr bool
	}{
		{name: "ordinary branch", branch: "release/1.2"},
		{name: "contains space", branch: "bad name", wantErr: true},
		{name: "starts with dash", branch: "-bad", wantErr: true},
		{name: "contains dot sequence", branch: "bad..name", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateInitBranch(tc.branch)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateInitBranch(%q) error = %v, wantErr %v", tc.branch, err, tc.wantErr)
			}
		})
	}
}

func validInitDraft() initDraft {
	cfg := flow.Config{}
	cfg.Branches.Main, cfg.Branches.Develop, cfg.Branches.Uat = "main", "develop", "uat"
	cfg.Branches.Features, cfg.Branches.Releases = "feature/", "release/"
	cfg.Branches.Hotfixes, cfg.Branches.Bugfixes = "hotfix/", "bugfix/"
	cfg.Flow.Feature = flow.BranchFlowRule{Base: "develop", FinishTargets: []string{"develop", "uat"}}
	cfg.Flow.Release = flow.BranchFlowRule{Base: "uat", FinishTargets: []string{"main", "develop"}}
	cfg.Flow.Hotfix = flow.BranchFlowRule{Base: "main", FinishTargets: []string{"main", "develop", "uat"}}
	cfg.Flow.Bugfix = flow.BranchFlowRule{Base: "uat", FinishTargets: []string{"uat", "develop"}}
	cfg.Workflow.DefaultMergeMode = "manual"
	cfg.Workflow.BranchRules = map[string]flow.WorkflowBranchRule{
		"main": {MergeMode: "manual"}, "develop": {MergeMode: "manual"}, "uat": {MergeMode: "manual"},
	}
	return initDraft{config: cfg, branches: []string{"main", "develop", "uat"}}
}

func recordingInitOperations() (initOperations, *[]string) {
	calls := []string{}
	return initOperations{
		validateBranch: func(branch string) error { calls = append(calls, "validate:"+branch); return nil },
		ensureBranch:   func(branch string) error { calls = append(calls, "ensure:"+branch); return nil },
		pushBranch:     func(branch string) error { calls = append(calls, "push:"+branch); return nil },
		generateAgentFiles: func(*flow.Config) error {
			calls = append(calls, "agent")
			return nil
		},
		saveConfig: func(*flow.Config) error { calls = append(calls, "save"); return nil },
	}, &calls
}

func assertNoInitMutations(t *testing.T, calls []string) {
	t.Helper()
	for _, call := range calls {
		if !strings.HasPrefix(call, "validate:") {
			t.Fatalf("invalid draft mutated state through calls %v", calls)
		}
	}
}
