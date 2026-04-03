package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yepizrene-devoost/dflow/cmd/commands"
	"github.com/yepizrene-devoost/dflow/cmd/utils"
)

func TestFinishMergesAutoTargetsAndSkipsManualOnes(t *testing.T) {
	repoDir := initTempGitRepo(t)
	remoteDir := initBareGitRepo(t)

	runGit(t, repoDir, "checkout", "-b", "develop")
	writeFileAndCommit(t, repoDir, "app.txt", "develop base\n", "seed develop")

	runGit(t, repoDir, "remote", "add", "origin", remoteDir)
	runGit(t, repoDir, "push", "-u", "origin", "main")
	runGit(t, repoDir, "push", "-u", "origin", "develop")

	runGit(t, repoDir, "checkout", "-b", "feature/demo")
	writeFileAndCommit(t, repoDir, "app.txt", "feature work\n", "feature work")

	cfg := &utils.Config{}
	cfg.Branches.Main = "main"
	cfg.Branches.Develop = "develop"
	cfg.Branches.Uat = "uat"
	cfg.Branches.Features = "feature/"
	cfg.Branches.Releases = "release/"
	cfg.Branches.Hotfixes = "hotfix/"
	cfg.Branches.Bugfixes = "bugfix/"
	cfg.Flow.Feature.Base = "develop"
	cfg.Flow.Feature.FinishTargets = []string{"develop", "main"}
	cfg.Workflow.DefaultMergeMode = "manual"
	cfg.Workflow.BranchRules = map[string]utils.WorkflowBranchRule{
		"develop": {MergeMode: "auto"},
		"main":    {MergeMode: "manual"},
	}

	withWorkingDir(t, repoDir, func() {
		if err := utils.SaveConfig(cfg); err != nil {
			t.Fatalf("failed to save config: %v", err)
		}
		runGit(t, repoDir, "add", ".dflow.yaml")
		runGit(t, repoDir, "commit", "-m", "add dflow config")

		if err := commands.FinishCmd.RunE(commands.FinishCmd, []string{}); err != nil {
			t.Fatalf("FinishCmd returned error: %v", err)
		}

		currentBranch := currentBranchName(t, repoDir)
		if currentBranch != "develop" {
			t.Fatalf("expected to end on develop, got %q", currentBranch)
		}

		developContent := readFile(t, filepath.Join(repoDir, "app.txt"))
		if !strings.Contains(developContent, "feature work") {
			t.Fatalf("expected develop to contain merged feature changes, got %q", developContent)
		}

		runGit(t, repoDir, "checkout", "main")
		mainContent := readFile(t, filepath.Join(repoDir, ".gitkeep"))
		if !strings.Contains(mainContent, "seed") {
			t.Fatalf("expected main branch to stay unchanged")
		}
		if _, err := os.Stat(filepath.Join(repoDir, "app.txt")); !os.IsNotExist(err) {
			t.Fatalf("expected manual target main to skip merge and keep app.txt absent")
		}
	})
}

func TestFinishDryRunDoesNotModifyBranches(t *testing.T) {
	repoDir := initTempGitRepo(t)
	remoteDir := initBareGitRepo(t)

	runGit(t, repoDir, "checkout", "-b", "develop")
	writeFileAndCommit(t, repoDir, "app.txt", "develop base\n", "seed develop")

	runGit(t, repoDir, "remote", "add", "origin", remoteDir)
	runGit(t, repoDir, "push", "-u", "origin", "main")
	runGit(t, repoDir, "push", "-u", "origin", "develop")

	runGit(t, repoDir, "checkout", "-b", "feature/demo")
	writeFileAndCommit(t, repoDir, "app.txt", "feature work\n", "feature work")

	cfg := &utils.Config{}
	cfg.Branches.Main = "main"
	cfg.Branches.Develop = "develop"
	cfg.Branches.Uat = "uat"
	cfg.Branches.Features = "feature/"
	cfg.Branches.Releases = "release/"
	cfg.Branches.Hotfixes = "hotfix/"
	cfg.Branches.Bugfixes = "bugfix/"
	cfg.Flow.Feature.Base = "develop"
	cfg.Flow.Feature.FinishTargets = []string{"develop", "main"}
	cfg.Workflow.DefaultMergeMode = "manual"
	cfg.Workflow.BranchRules = map[string]utils.WorkflowBranchRule{
		"develop": {MergeMode: "auto"},
		"main":    {MergeMode: "manual"},
	}

	withWorkingDir(t, repoDir, func() {
		if err := utils.SaveConfig(cfg); err != nil {
			t.Fatalf("failed to save config: %v", err)
		}
		runGit(t, repoDir, "add", ".dflow.yaml")
		runGit(t, repoDir, "commit", "-m", "add dflow config")

		if err := commands.FinishCmd.Flags().Set("dry-run", "true"); err != nil {
			t.Fatalf("failed to enable dry-run flag: %v", err)
		}
		defer func() {
			_ = commands.FinishCmd.Flags().Set("dry-run", "false")
		}()

		if err := commands.FinishCmd.RunE(commands.FinishCmd, []string{}); err != nil {
			t.Fatalf("FinishCmd returned error: %v", err)
		}

		currentBranch := currentBranchName(t, repoDir)
		if currentBranch != "feature/demo" {
			t.Fatalf("expected dry-run to keep current branch on feature/demo, got %q", currentBranch)
		}

		runGit(t, repoDir, "checkout", "develop")
		developContent := readFile(t, filepath.Join(repoDir, "app.txt"))
		if strings.Contains(developContent, "feature work") {
			t.Fatalf("expected dry-run to skip merge into develop, got %q", developContent)
		}
	})
}

func currentBranchName(t *testing.T, repoDir string) string {
	t.Helper()

	output := runGitOutput(t, repoDir, "rev-parse", "--abbrev-ref", "HEAD")
	return strings.TrimSpace(output)
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file %q: %v", path, err)
	}

	return string(content)
}
