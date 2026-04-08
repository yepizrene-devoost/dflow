package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yepizrene-devoost/dflow/cmd/utils"
)

// TestSaveAndLoadConfig verifies that the dflow configuration can be saved
// to a .dflow.yaml file and loaded back correctly. It checks that the file
// is created and that loaded values match the original configuration.
func TestSaveAndLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFLOW_CWD", tmpDir)

	original := &utils.Config{}
	original.Branches.Main = "main"
	original.Branches.Develop = "develop"
	original.Branches.Uat = "uat"
	original.Branches.Features = "feature/"
	original.Branches.Releases = "release/"
	original.Branches.Hotfixes = "hotfix/"
	original.Branches.Bugfixes = "bugfix/"

	original.Flow.Feature.Base = "develop"
	original.Flow.Feature.FinishTargets = []string{"develop", "uat"}
	original.Flow.Release.Base = "uat"
	original.Flow.Release.FinishTargets = []string{"main", "develop"}
	original.Flow.Hotfix.Base = "main"
	original.Flow.Hotfix.FinishTargets = []string{"main", "develop", "uat"}
	original.Flow.Bugfix.Base = "uat"
	original.Flow.Bugfix.FinishTargets = []string{"uat", "develop"}

	original.Workflow.DefaultMergeMode = "manual"
	original.Workflow.BranchRules = map[string]utils.WorkflowBranchRule{
		"main":    {MergeMode: "manual"},
		"develop": {MergeMode: "auto"},
		"uat":     {MergeMode: "auto"},
	}

	if err := utils.SaveConfig(original); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Verifica que el archivo fue creado
	configPath := filepath.Join(tmpDir, ".dflow.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf(".dflow.yaml not found at: %s", configPath)
	}

	loaded, err := utils.LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.Branches.Main != original.Branches.Main {
		t.Errorf("expected main branch '%s', got '%s'", original.Branches.Main, loaded.Branches.Main)
	}

	if loaded.Flow.Release.Base != "uat" {
		t.Errorf("expected release base 'uat', got '%s'", loaded.Flow.Release.Base)
	}

	if len(loaded.Flow.Feature.FinishTargets) != 2 {
		t.Fatalf("expected 2 feature targets, got %d", len(loaded.Flow.Feature.FinishTargets))
	}

	if len(loaded.Flow.Hotfix.FinishTargets) != 3 {
		t.Fatalf("expected 3 hotfix targets, got %d", len(loaded.Flow.Hotfix.FinishTargets))
	}

	if utils.GetMergeModeForBranch(loaded, "develop") != "auto" {
		t.Errorf("expected develop merge mode 'auto', got '%s'", utils.GetMergeModeForBranch(loaded, "develop"))
	}
}

func TestLoadLegacyConfigFormat(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DFLOW_CWD", tmpDir)

	legacyConfig := `
branches:
  main: main
  develop: develop
  uat: uat
  features: feature/
  releases: release/
  hotfixes: hotfix/
  bugfixes: bugfix/
flow:
  feature_base: uat
  feature_merge: develop
  release_base: uat
  hotfix_base: main
  bugfix_base: uat
workflow:
  default_merge_mode: auto
  branch_rules:
    main: manual
`

	configPath := filepath.Join(tmpDir, ".dflow.yaml")
	if err := os.WriteFile(configPath, []byte(legacyConfig), 0644); err != nil {
		t.Fatalf("failed to write legacy config: %v", err)
	}

	loaded, err := utils.LoadConfig()
	if err != nil {
		t.Fatalf("failed to load legacy config: %v", err)
	}

	if loaded.Flow.Feature.Base != "uat" {
		t.Errorf("expected legacy feature base 'uat', got '%s'", loaded.Flow.Feature.Base)
	}

	if len(loaded.Flow.Feature.FinishTargets) != 1 || loaded.Flow.Feature.FinishTargets[0] != "develop" {
		t.Fatalf("expected feature finish target 'develop', got %v", loaded.Flow.Feature.FinishTargets)
	}

	if utils.GetMergeModeForBranch(loaded, "main") != "manual" {
		t.Errorf("expected main merge mode 'manual', got '%s'", utils.GetMergeModeForBranch(loaded, "main"))
	}

	saved, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read legacy config: %v", err)
	}

	if !strings.Contains(string(saved), "feature_base") {
		t.Fatalf("expected fixture to keep legacy content before save")
	}
}
