package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yepizrene-devoost/dflow/cmd/utils"
)

// TestSaveAndLoadConfig verifies that the dflow configuration can be saved
// to a .dflow.yaml file and loaded back correctly. It checks that the file
// is created and that loaded values match the original configuration.
func TestSaveAndLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("DFLOW_CWD", tmpDir)
	defer os.Unsetenv("DFLOW_CWD")

	original := &utils.Config{}
	original.Branches.Main = "main"
	original.Branches.Develop = "develop"
	original.Branches.Uat = "uat"
	original.Branches.Features = "feature/"
	original.Branches.Releases = "release/"
	original.Branches.Hotfixes = "hotfix/"

	original.Flow.FeatureBase = "uat"
	original.Flow.FeatureMerge = "develop"
	original.Flow.ReleaseBase = "uat"
	original.Flow.HotfixBase = "main"

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
}
