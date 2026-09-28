package utils

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yepizrene-devoost/dflow/pkg/flow"
)

func testSaveConfigConfig() *flow.Config {
	cfg := &flow.Config{}
	cfg.Branches.Main = "main"
	cfg.Branches.Develop = "develop"
	cfg.Branches.Uat = "uat"
	cfg.Branches.Features = "feature/"
	cfg.Branches.Releases = "release/"
	cfg.Branches.Hotfixes = "hotfix/"
	cfg.Branches.Bugfixes = "bugfix/"
	cfg.Flow.Feature.Base = "develop"
	cfg.Flow.Feature.FinishTargets = []string{"develop"}
	cfg.Workflow.DefaultMergeMode = flow.MergeModeManual
	return cfg
}

func TestSaveConfigRejectsNilConfig(t *testing.T) {
	if err := SaveConfig(nil); err == nil || !strings.Contains(err.Error(), "nil config") {
		t.Fatalf("SaveConfig(nil) error = %v, want explicit nil-config error", err)
	}
}

func TestSaveConfigWritesInvalidConfigForLoadConfigValidation(t *testing.T) {
	dir := t.TempDir()
	if output, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, output)
	}
	t.Setenv("DFLOW_CWD", dir)

	cfg := testSaveConfigConfig()
	cfg.Workflow.BranchRules = map[string]flow.WorkflowBranchRule{
		"develop": {MergeMode: "pr"},
	}
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v, want invalid config to be written", err)
	}

	if _, err := LoadConfig(); err == nil || !strings.Contains(err.Error(), "workflow.branch_rules.develop.merge_mode") {
		t.Fatalf("LoadConfig() error = %v, want path-specific validation error", err)
	}
}

func TestSaveConfigRoundTripsPartialFixtureConfig(t *testing.T) {
	dir := t.TempDir()
	if output, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\\n%s", err, output)
	}
	t.Setenv("DFLOW_CWD", dir)

	cfg := testSaveConfigConfig()
	cfg.Workflow.DefaultMergeMode = ""
	cfg.Workflow.BranchRules = nil
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if loaded.Flow.Feature.Base != cfg.Flow.Feature.Base || strings.Join(loaded.Flow.Feature.FinishTargets, ",") != strings.Join(cfg.Flow.Feature.FinishTargets, ",") {
		t.Fatalf("loaded feature rule = %+v, want %+v", loaded.Flow.Feature, cfg.Flow.Feature)
	}
	if loaded.Workflow.DefaultMergeMode != "" || len(loaded.Workflow.BranchRules) != 0 {
		t.Fatalf("loaded workflow = %+v, want unset merge mode and empty branch rules", loaded.Workflow)
	}
}

func TestSaveConfigKeepsExistingTargetWhenRenameFails(t *testing.T) {
	dir := t.TempDir()
	if output, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\\n%s", err, output)
	}
	t.Setenv("DFLOW_CWD", dir)

	path := filepath.Join(dir, ".dflow.yaml")
	if err := os.Mkdir(path, 0755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(path, "previous-target")
	if err := os.WriteFile(marker, []byte("intact"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := SaveConfig(testSaveConfigConfig()); err == nil || !strings.Contains(err.Error(), "error replacing .dflow.yaml") {
		t.Fatalf("SaveConfig() error = %v, want atomic rename failure", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("previous target was removed: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("previous target changed type: mode=%s", info.Mode())
	}
	contents, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("previous target contents unavailable: %v", err)
	}
	if string(contents) != "intact" {
		t.Fatalf("previous target contents changed: %q", contents)
	}
}
