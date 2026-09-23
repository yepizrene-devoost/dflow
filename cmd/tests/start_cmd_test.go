package tests

import (
	"strings"
	"testing"

	"github.com/yepizrene-devoost/dflow/cmd/commands"
	"github.com/yepizrene-devoost/dflow/cmd/utils"
	"github.com/yepizrene-devoost/dflow/pkg/flow"
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

func startTestConfig() *flow.Config {
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
	cfg.Workflow.DefaultMergeMode = "manual"
	cfg.Workflow.BranchRules = map[string]flow.WorkflowBranchRule{
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

		if err := commands.StartCmd.RunE(commands.StartCmd, []string{"feat", "child"}); err == nil {
			t.Fatalf("expected StartCmd to return an error for an unknown parent")
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

func TestStartFromLocalParentWithUnreachableOrigin(t *testing.T) {
	repoDir := initTempGitRepo(t)

	runGit(t, repoDir, "checkout", "-b", "develop")
	writeFileAndCommit(t, repoDir, "app.txt", "develop base\n", "seed develop")

	runGit(t, repoDir, "checkout", "-b", "feature/parent")
	writeFileAndCommit(t, repoDir, "app.txt", "parent work\n", "parent work")

	// Dead origin: any fetch/ls-remote/pull must not be reached for a local parent.
	runGit(t, repoDir, "remote", "add", "origin", "/nonexistent/dflow-dead.git")

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
			t.Fatalf("expected feature/child to be created from a local parent without touching origin")
		}

		childTip := strings.TrimSpace(runGitOutput(t, repoDir, "rev-parse", "feature/child"))
		parentTip := strings.TrimSpace(runGitOutput(t, repoDir, "rev-parse", "feature/parent"))
		if childTip != parentTip {
			t.Fatalf("expected feature/child based on feature/parent (%s), got %s", parentTip, childTip)
		}
	})
}

func TestStartRejectsConflictingPushFlagsBeforeMutation(t *testing.T) {
	repoDir := setupStartRepo(t)

	withWorkingDir(t, repoDir, func() {
		if err := utils.SaveConfig(startTestConfig()); err != nil {
			t.Fatalf("failed to save config: %v", err)
		}

		if err := commands.StartCmd.Flags().Set("from", "feature/parent"); err != nil {
			t.Fatalf("failed to set --from: %v", err)
		}
		if err := commands.StartCmd.Flags().Set("push", "true"); err != nil {
			t.Fatalf("failed to set --push: %v", err)
		}
		if err := commands.StartCmd.Flags().Set("no-push", "true"); err != nil {
			t.Fatalf("failed to set --no-push: %v", err)
		}
		defer func() {
			_ = commands.StartCmd.Flags().Set("from", "")
			_ = commands.StartCmd.Flags().Set("push", "false")
			_ = commands.StartCmd.Flags().Set("no-push", "false")
		}()

		if err := commands.StartCmd.RunE(commands.StartCmd, []string{"feat", "child"}); err == nil {
			t.Fatalf("expected StartCmd to return an error when --push and --no-push conflict")
		}

		if branchExists(t, repoDir, "feature/child") {
			t.Fatalf("expected feature/child NOT to be created when --push and --no-push conflict")
		}

		if currentBranchName(t, repoDir) != "feature/parent" {
			t.Fatalf("expected to remain on feature/parent with no mutation, got %q", currentBranchName(t, repoDir))
		}
	})
}
