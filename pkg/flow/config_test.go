package flow

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func validConfig() Config {
	var cfg Config
	cfg.Branches.Main = "main"
	cfg.Branches.Develop = "develop"
	cfg.Branches.Uat = "uat"
	cfg.Branches.Features = "feature/"
	cfg.Branches.Releases = "release/"
	cfg.Branches.Hotfixes = "hotfix/"
	cfg.Branches.Bugfixes = "bugfix/"
	cfg.Flow.Feature = BranchFlowRule{Base: "develop", FinishTargets: []string{"develop", "uat"}}
	cfg.Flow.Release = BranchFlowRule{Base: "uat", FinishTargets: []string{"main"}}
	cfg.Flow.Hotfix = BranchFlowRule{Base: "main", FinishTargets: []string{"main"}}
	cfg.Flow.Bugfix = BranchFlowRule{Base: "uat", FinishTargets: []string{"uat"}}
	cfg.Workflow.DefaultMergeMode = MergeModeManual
	cfg.Workflow.BranchRules = map[string]WorkflowBranchRule{"main": {MergeMode: MergeModeManual}}
	return cfg
}

func TestConfigValidateReportsPathSpecificErrors(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantValue string
		edit      func(*Config)
	}{
		{"missing main", "branches.main", "", func(c *Config) { c.Branches.Main = "" }},
		{"missing prefix", "branches.features", "", func(c *Config) { c.Branches.Features = " " }},
		{"overlapping prefixes", "branches", "", func(c *Config) { c.Branches.Releases = "feature/" }},
		{"missing base", "flow.feature.base", "", func(c *Config) { c.Flow.Feature.Base = "" }},
		{"blank target", "flow.feature.finish_targets[1]", "", func(c *Config) { c.Flow.Feature.FinishTargets[1] = " " }},
		{"duplicate target", "flow.feature.finish_targets", "", func(c *Config) { c.Flow.Feature.FinishTargets = []string{"develop", "develop"} }},
		{"invalid default mode", "workflow.default_merge_mode", "", func(c *Config) { c.Workflow.DefaultMergeMode = "sometimes" }},
		{"invalid branch mode", "workflow.branch_rules.main.merge_mode", "pr", func(c *Config) { c.Workflow.BranchRules["main"] = WorkflowBranchRule{MergeMode: "pr"} }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := validConfig()
			test.edit(&cfg)
			err := cfg.Validate()
			if err == nil || !strings.Contains(err.Error(), test.path) {
				t.Fatalf("Validate() error = %v, want path %q", err, test.path)
			}
			if test.wantValue != "" && !strings.Contains(err.Error(), test.wantValue) {
				t.Fatalf("Validate() error = %v, want invalid value %q", err, test.wantValue)
			}
		})
	}
}

func TestConfigValidateAcceptsValidConfig(t *testing.T) {
	if err := validConfig().Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestConfigValidateAllowsAbsentFlowRules(t *testing.T) {
	cfg := validConfig()
	cfg.Flow.Release = BranchFlowRule{}
	cfg.Flow.Hotfix = BranchFlowRule{}
	cfg.Flow.Bugfix = BranchFlowRule{}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want absent flow rules to be allowed: %v", err, err)
	}
}

func TestConfigValidateAllowsZeroValuedEmittedRulesAndUnsetWorkflow(t *testing.T) {
	original := validConfig()
	original.Flow.Release = BranchFlowRule{}
	original.Flow.Hotfix = BranchFlowRule{}
	original.Flow.Bugfix = BranchFlowRule{}
	original.Workflow.DefaultMergeMode = ""
	original.Workflow.BranchRules = nil

	data, err := yaml.Marshal(&original)
	if err != nil {
		t.Fatal(err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want zero-valued emitted rules and unset workflow to be valid", err)
	}
}

func TestConfigValidateRejectsPartiallySpecifiedFlowRules(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Config)
		path string
	}{
		{
			name: "missing base",
			edit: func(cfg *Config) {
				cfg.Flow.Release = BranchFlowRule{FinishTargets: []string{"main"}}
			},
			path: "flow.release.base",
		},
		{
			name: "missing targets",
			edit: func(cfg *Config) {
				cfg.Flow.Release = BranchFlowRule{Base: "uat"}
			},
			path: "flow.release.finish_targets",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := validConfig()
			test.edit(&cfg)
			if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), test.path) {
				t.Fatalf("Validate() error = %v, want path %q", err, test.path)
			}
		})
	}
}

func TestConfigYAMLRejectsMalformedAndContradictoryValues(t *testing.T) {
	tests := []string{
		"flow:\n  feature: nope\n",
		"flow:\n  feature:\n    base: develop\n    finish_targets: [develop]\n  feature_base: release\n",
		"workflow:\n  branch_rules:\n    main: 42\n",
		"workflow:\n  branch_rules:\n    main:\n      merge_mode: 42\n",
	}
	for _, input := range tests {
		var cfg Config
		if err := yaml.Unmarshal([]byte(input), &cfg); err == nil {
			t.Errorf("yaml.Unmarshal(%q) succeeded, want error", input)
		}
	}
}

func TestConfigYAMLDecodesLegacyRulesDeterministically(t *testing.T) {
	input := `flow:
  feature_base: develop
  feature_merge: uat
  release_base: uat
  release_merge: main
  hotfix_base: main
  hotfix_merge: develop
  bugfix_base: uat
  bugfix_merge: develop
workflow:
  default_merge_mode: manual
  branch_rules:
    main: auto
`
	var cfg Config
	if err := yaml.Unmarshal([]byte(input), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Flow.Release.Base != "uat" || len(cfg.Flow.Release.FinishTargets) != 1 || cfg.Flow.Release.FinishTargets[0] != "main" {
		t.Fatalf("unexpected legacy release rule: %+v", cfg.Flow.Release)
	}
	if cfg.Flow.Hotfix.FinishTargets[0] != "develop" || cfg.Flow.Bugfix.FinishTargets[0] != "develop" {
		t.Fatalf("unexpected legacy finish targets: hotfix=%v bugfix=%v", cfg.Flow.Hotfix.FinishTargets, cfg.Flow.Bugfix.FinishTargets)
	}
	if cfg.Workflow.BranchRules["main"].MergeMode != "auto" {
		t.Fatalf("unexpected legacy branch rule: %+v", cfg.Workflow.BranchRules)
	}
}
