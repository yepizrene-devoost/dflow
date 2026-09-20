package tests

import (
	"strings"
	"testing"

	"github.com/yepizrene-devoost/dflow/cmd/commands"
	"github.com/yepizrene-devoost/dflow/cmd/utils"
)

func setupStartRepo(t *testing.T) string {
	t.Helper()

	repoDir := initTempGitRepo(t)
	remoteDir := initBareGitRepo(t)

	runGit(t, repoDir, "checkout", "-b", "develop")
	writeFileAndCommit(t, repoDir, "app.txt", "develop base\n", "seed develop")

	runGit(t, repoDir, "remote", "add", "origin", remoteDir)
	runGit(t, repoDir, "push", "-u", "origin", "main")
	runGit(t, repoDir, "push", "-u", "origin", "develop")

	runGit(t, repoDir, "checkout", "-b", "feature/parent")
	writeFileAndCommit(t, repoDir, "app.txt", "parent work\n", "parent work")

	return repoDir
}

func startTestConfig() *utils.Config {
	cfg := &utils.Config{}
	cfg.Branches.Main = "main"
	cfg.Branches.Develop = "develop"
	cfg.Branches.Uat = "uat"
	cfg.Branches.Features = "feature/"
	cfg.Branches.Releases = "release/"
	cfg.Branches.Hotfixes = "hotfix/"
	cfg.Branches.Bugfixes = "bugfix/"
	cfg.Flow.Feature.Base = "develop"
	cfg.Flow.Feature.FinishTargets = []string{"develop"}
	cfg.Workflow.DefaultMergeMode = "manual"
	cfg.Workflow.BranchRules = map[string]utils.WorkflowBranchRule{
		"develop": {MergeMode: "auto"},
	}
	return cfg
}

func TestStartFromCreatesChainedBranch(t *testing.T) {
	repoDir := setupStartRepo(t)

	withWorkingDir(t, repoDir, func() {
		if err := utils.SaveConfig(startTestConfig()); err != nil {
			t.Fatalf("failed to save config: %v", err)
		}

		if err := commands.StartCmd.Flags().Set("from", "feature/parent"); err != nil {
			t.Fatalf("failed to set --from: %v", err)
		}
		if err := commands.StartCmd.Flags().Set("no-push", "true"); err != nil {
			t.Fatalf("failed to set --no-push: %v", err)
		}
		defer func() {
			_ = commands.StartCmd.Flags().Set("from", "")
			_ = commands.StartCmd.Flags().Set("no-push", "false")
		}()

		if err := commands.StartCmd.RunE(commands.StartCmd, []string{"feat", "child"}); err != nil {
			t.Fatalf("StartCmd returned error: %v", err)
		}

		if !branchExists(t, repoDir, "feature/child") {
			t.Fatalf("expected feature/child to be created")
		}

		childTip := strings.TrimSpace(runGitOutput(t, repoDir, "rev-parse", "feature/child"))
		parentTip := strings.TrimSpace(runGitOutput(t, repoDir, "rev-parse", "feature/parent"))
		if childTip != parentTip {
			t.Fatalf("expected feature/child to be based on feature/parent (%s), got %s", parentTip, childTip)
		}

		if currentBranchName(t, repoDir) != "feature/child" {
			t.Fatalf("expected current branch to be feature/child, got %q", currentBranchName(t, repoDir))
		}
	})
}

func TestStartFromUnknownParentDoesNotCreateBranch(t *testing.T) {
	repoDir := setupStartRepo(t)

	withWorkingDir(t, repoDir, func() {
		if err := utils.SaveConfig(startTestConfig()); err != nil {
			t.Fatalf("failed to save config: %v", err)
		}

		if err := commands.StartCmd.Flags().Set("from", "feature/does-not-exist"); err != nil {
			t.Fatalf("failed to set --from: %v", err)
		}
		if err := commands.StartCmd.Flags().Set("no-push", "true"); err != nil {
			t.Fatalf("failed to set --no-push: %v", err)
		}
		defer func() {
			_ = commands.StartCmd.Flags().Set("from", "")
			_ = commands.StartCmd.Flags().Set("no-push", "false")
		}()

		if err := commands.StartCmd.RunE(commands.StartCmd, []string{"feat", "child"}); err != nil {
			t.Fatalf("StartCmd returned error: %v", err)
		}

		if branchExists(t, repoDir, "feature/child") {
			t.Fatalf("expected feature/child NOT to be created for an unknown parent")
		}
	})
}

func TestStartNoPushDoesNotPublishRemote(t *testing.T) {
	repoDir := setupStartRepo(t)

	withWorkingDir(t, repoDir, func() {
		if err := utils.SaveConfig(startTestConfig()); err != nil {
			t.Fatalf("failed to save config: %v", err)
		}

		if err := commands.StartCmd.Flags().Set("no-push", "true"); err != nil {
			t.Fatalf("failed to set --no-push: %v", err)
		}
		defer func() {
			_ = commands.StartCmd.Flags().Set("no-push", "false")
		}()

		if err := commands.StartCmd.RunE(commands.StartCmd, []string{"feat", "child"}); err != nil {
			t.Fatalf("StartCmd returned error: %v", err)
		}

		if !branchExists(t, repoDir, "feature/child") {
			t.Fatalf("expected feature/child to be created")
		}

		if remoteBranchExists(t, repoDir, "feature/child") {
			t.Fatalf("expected feature/child NOT to be pushed with --no-push")
		}
	})
}

func TestStartPushPublishesRemote(t *testing.T) {
	repoDir := setupStartRepo(t)

	withWorkingDir(t, repoDir, func() {
		if err := utils.SaveConfig(startTestConfig()); err != nil {
			t.Fatalf("failed to save config: %v", err)
		}

		if err := commands.StartCmd.Flags().Set("push", "true"); err != nil {
			t.Fatalf("failed to set --push: %v", err)
		}
		defer func() {
			_ = commands.StartCmd.Flags().Set("push", "false")
		}()

		if err := commands.StartCmd.RunE(commands.StartCmd, []string{"feat", "child"}); err != nil {
			t.Fatalf("StartCmd returned error: %v", err)
		}

		if !branchExists(t, repoDir, "feature/child") {
			t.Fatalf("expected feature/child to be created")
		}

		if !remoteBranchExists(t, repoDir, "feature/child") {
			t.Fatalf("expected feature/child to be pushed with --push")
		}
	})
}
