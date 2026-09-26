package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yepizrene-devoost/dflow/cmd/commands"
	"github.com/yepizrene-devoost/dflow/cmd/utils"
	"github.com/yepizrene-devoost/dflow/pkg/flow"
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

	cfg := &flow.Config{}
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
	cfg.Workflow.BranchRules = map[string]flow.WorkflowBranchRule{
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

	cfg := &flow.Config{}
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
	cfg.Workflow.BranchRules = map[string]flow.WorkflowBranchRule{
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

// TestFinishDeleteRemovesBranchWhenNoManualTargetsRemain also proves the
// in-process command path is fully non-interactive: --delete is explicit intent
// and must not wait for a confirmation prompt.
func TestFinishDeleteRemovesBranchWhenNoManualTargetsRemain(t *testing.T) {
	repoDir := initTempGitRepo(t)
	remoteDir := initBareGitRepo(t)

	runGit(t, repoDir, "checkout", "-b", "develop")
	writeFileAndCommit(t, repoDir, "app.txt", "develop base\n", "seed develop")

	runGit(t, repoDir, "remote", "add", "origin", remoteDir)
	runGit(t, repoDir, "push", "-u", "origin", "main")
	runGit(t, repoDir, "push", "-u", "origin", "develop")

	runGit(t, repoDir, "checkout", "-b", "feature/delete-me")
	writeFileAndCommit(t, repoDir, "app.txt", "feature work\n", "feature work")
	runGit(t, repoDir, "push", "-u", "origin", "feature/delete-me")

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

	withWorkingDir(t, repoDir, func() {
		if err := utils.SaveConfig(cfg); err != nil {
			t.Fatalf("failed to save config: %v", err)
		}
		runGit(t, repoDir, "add", ".dflow.yaml")
		runGit(t, repoDir, "commit", "-m", "add dflow config")

		if err := commands.FinishCmd.Flags().Set("delete", "true"); err != nil {
			t.Fatalf("failed to enable delete flag: %v", err)
		}
		defer func() {
			_ = commands.FinishCmd.Flags().Set("delete", "false")
		}()

		if err := commands.FinishCmd.RunE(commands.FinishCmd, []string{}); err != nil {
			t.Fatalf("FinishCmd returned error: %v", err)
		}

		currentBranch := currentBranchName(t, repoDir)
		if currentBranch != "develop" {
			t.Fatalf("expected to end on develop, got %q", currentBranch)
		}

		if branchExists(t, repoDir, "feature/delete-me") {
			t.Fatalf("expected local feature/delete-me branch to be deleted")
		}

		if remoteBranchExists(t, repoDir, "feature/delete-me") {
			t.Fatalf("expected remote feature/delete-me branch to be deleted")
		}
	})
}

func TestFinishDeleteSkipsBranchRemovalWhenManualTargetsRemain(t *testing.T) {
	repoDir := initTempGitRepo(t)
	remoteDir := initBareGitRepo(t)

	runGit(t, repoDir, "checkout", "-b", "develop")
	writeFileAndCommit(t, repoDir, "app.txt", "develop base\n", "seed develop")

	runGit(t, repoDir, "remote", "add", "origin", remoteDir)
	runGit(t, repoDir, "push", "-u", "origin", "main")
	runGit(t, repoDir, "push", "-u", "origin", "develop")

	runGit(t, repoDir, "checkout", "-b", "feature/keep-me")
	writeFileAndCommit(t, repoDir, "app.txt", "feature work\n", "feature work")
	runGit(t, repoDir, "push", "-u", "origin", "feature/keep-me")

	cfg := &flow.Config{}
	cfg.Branches.Main = "main"
	cfg.Branches.Develop = "develop"
	cfg.Branches.Uat = "uat"
	cfg.Branches.Features = "feature/"
	cfg.Branches.Releases = "release/"
	cfg.Branches.Hotfixes = "hotfix/"
	cfg.Branches.Bugfixes = "bugfix/"
	cfg.Flow.Feature.Base = "develop"
	cfg.Flow.Feature.FinishTargets = []string{"develop", "uat"}
	cfg.Workflow.DefaultMergeMode = "manual"
	cfg.Workflow.BranchRules = map[string]flow.WorkflowBranchRule{
		"develop": {MergeMode: "auto"},
		"uat":     {MergeMode: "manual"},
	}

	withWorkingDir(t, repoDir, func() {
		if err := utils.SaveConfig(cfg); err != nil {
			t.Fatalf("failed to save config: %v", err)
		}
		runGit(t, repoDir, "add", ".dflow.yaml")
		runGit(t, repoDir, "commit", "-m", "add dflow config")

		if err := commands.FinishCmd.Flags().Set("delete", "true"); err != nil {
			t.Fatalf("failed to enable delete flag: %v", err)
		}
		defer func() {
			_ = commands.FinishCmd.Flags().Set("delete", "false")
		}()

		if err := commands.FinishCmd.RunE(commands.FinishCmd, []string{}); err != nil {
			t.Fatalf("FinishCmd returned error: %v", err)
		}

		currentBranch := currentBranchName(t, repoDir)
		if currentBranch != "develop" {
			t.Fatalf("expected to end on develop, got %q", currentBranch)
		}

		if !branchExists(t, repoDir, "feature/keep-me") {
			t.Fatalf("expected local feature/keep-me branch to remain because manual targets are pending")
		}

		if !remoteBranchExists(t, repoDir, "feature/keep-me") {
			t.Fatalf("expected remote feature/keep-me branch to remain because manual targets are pending")
		}
	})
}

func TestFinishDeleteKeepsBranchAfterMergeConflict(t *testing.T) {
	repoDir, _ := finishPublishRepo(t)

	withWorkingDir(t, repoDir, func() {
		runGit(t, repoDir, "checkout", "develop")
		writeFileAndCommit(t, repoDir, "app.txt", "develop conflict\n", "conflict on develop")
		runGit(t, repoDir, "checkout", "feature/publish-me")
		finishPublishConfig(t, repoDir, []string{"develop"}, map[string]flow.WorkflowBranchRule{
			"develop": {MergeMode: "auto"},
		})

		if err := commands.FinishCmd.Flags().Set("delete", "true"); err != nil {
			t.Fatalf("failed to enable delete flag: %v", err)
		}
		defer func() {
			_ = commands.FinishCmd.Flags().Set("delete", "false")
		}()

		err := commands.FinishCmd.RunE(commands.FinishCmd, []string{})
		if err == nil {
			t.Fatalf("expected FinishCmd to fail on a merge conflict")
		}
		if !strings.Contains(err.Error(), "Merge conflict") {
			t.Fatalf("expected conflict error, got: %v", err)
		}
		if !branchExists(t, repoDir, "feature/publish-me") {
			t.Fatalf("expected the work branch to remain after a merge conflict")
		}
		if !remoteBranchExists(t, repoDir, "feature/publish-me") {
			t.Fatalf("expected the published work branch to remain after a merge conflict")
		}
	})
}

func TestFinishDeleteKeepsBranchAfterLaterAutoTargetConflict(t *testing.T) {
	repoDir, _ := finishPublishRepo(t)

	withWorkingDir(t, repoDir, func() {
		runGit(t, repoDir, "checkout", "develop")
		runGit(t, repoDir, "checkout", "-b", "uat")
		writeFileAndCommit(t, repoDir, "app.txt", "uat conflict\n", "conflict on uat")
		runGit(t, repoDir, "push", "-u", "origin", "uat")
		runGit(t, repoDir, "checkout", "feature/publish-me")
		finishPublishConfig(t, repoDir, []string{"develop", "uat"}, map[string]flow.WorkflowBranchRule{
			"develop": {MergeMode: "auto"},
			"uat":     {MergeMode: "auto"},
		})

		if err := commands.FinishCmd.Flags().Set("delete", "true"); err != nil {
			t.Fatalf("failed to enable delete flag: %v", err)
		}
		defer func() {
			_ = commands.FinishCmd.Flags().Set("delete", "false")
		}()

		err := commands.FinishCmd.RunE(commands.FinishCmd, []string{})
		if err == nil {
			t.Fatalf("expected FinishCmd to fail on the later auto-target conflict")
		}
		if !strings.Contains(err.Error(), "Merge conflict") {
			t.Fatalf("expected conflict error, got: %v", err)
		}
		if !branchExists(t, repoDir, "feature/publish-me") {
			t.Fatalf("expected the work branch to remain after a later target failure")
		}
		if !remoteBranchExists(t, repoDir, "feature/publish-me") {
			t.Fatalf("expected the published work branch to remain after a later target failure")
		}
		if current := currentBranchName(t, repoDir); current != "uat" {
			t.Fatalf("expected to remain on failed target uat, got %q", current)
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

func branchExists(t *testing.T, repoDir, branch string) bool {
	t.Helper()

	output := runGitOutput(t, repoDir, "branch", "--list", branch)
	return strings.TrimSpace(output) != ""
}

func remoteBranchExists(t *testing.T, repoDir, branch string) bool {
	t.Helper()

	return remoteRevision(t, repoDir, branch) != ""
}

// finishPublishRepo builds the fixture the publish tests share: a repository with
// an origin, a pushed develop, and a work branch with one commit that was never
// pushed.
func finishPublishRepo(t *testing.T) (repoDir string, remoteDir string) {
	t.Helper()

	repoDir = initTempGitRepo(t)
	remoteDir = initBareGitRepo(t)

	runGit(t, repoDir, "checkout", "-b", "develop")
	writeFileAndCommit(t, repoDir, "app.txt", "develop base\n", "seed develop")

	runGit(t, repoDir, "remote", "add", "origin", remoteDir)
	runGit(t, repoDir, "push", "-u", "origin", "main")
	runGit(t, repoDir, "push", "-u", "origin", "develop")

	runGit(t, repoDir, "checkout", "-b", "feature/publish-me")
	writeFileAndCommit(t, repoDir, "app.txt", "feature work\n", "feature work")

	return repoDir, remoteDir
}

// finishPublishConfig writes and commits a .dflow.yaml for a feature branch whose
// finish targets are `targets` with the given merge modes. It must be called
// inside withWorkingDir.
func finishPublishConfig(t *testing.T, repoDir string, targets []string, rules map[string]flow.WorkflowBranchRule) {
	t.Helper()

	cfg := &flow.Config{}
	cfg.Branches.Main = "main"
	cfg.Branches.Develop = "develop"
	cfg.Branches.Uat = "uat"
	cfg.Branches.Features = "feature/"
	cfg.Branches.Releases = "release/"
	cfg.Branches.Hotfixes = "hotfix/"
	cfg.Branches.Bugfixes = "bugfix/"
	cfg.Flow.Feature.Base = "develop"
	cfg.Flow.Feature.FinishTargets = targets
	cfg.Workflow.DefaultMergeMode = "manual"
	cfg.Workflow.BranchRules = rules

	if err := utils.SaveConfig(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}
	runGit(t, repoDir, "add", ".dflow.yaml")
	runGit(t, repoDir, "commit", "-m", "add dflow config")
}

// setUnreachablePushURL points remote.origin.pushurl at a path that cannot be a
// repository, so any `git push` fails while every lookup keeps working: ls-remote
// reads the fetch URL and only push uses the push URL. A publish that is attempted
// therefore fails deterministically, which is what tells "the publish was skipped"
// apart from "a push happened". A failing pre-push hook cannot do this, because
// Git skips the hook for a ref that is already up to date.
func setUnreachablePushURL(t *testing.T, repoDir string) {
	t.Helper()

	runGit(t, repoDir, "remote", "set-url", "--push", "origin", filepath.Join(t.TempDir(), "not-a-repo"))
}

// remoteRevision returns origin's commit for the branch, or "" when origin does
// not have it, using `ls-remote --heads origin <branch>`.
func remoteRevision(t *testing.T, repoDir, branch string) string {
	t.Helper()

	output := runGitOutput(t, repoDir, "ls-remote", "--heads", "origin", branch)
	fields := strings.Fields(output)
	if len(fields) == 0 {
		return ""
	}

	return fields[0]
}

func TestFinishPublishesTheWorkBranchBeforeMerging(t *testing.T) {
	repoDir, _ := finishPublishRepo(t)

	withWorkingDir(t, repoDir, func() {
		finishPublishConfig(t, repoDir, []string{"develop"}, map[string]flow.WorkflowBranchRule{
			"develop": {MergeMode: "auto"},
		})

		if err := commands.FinishCmd.RunE(commands.FinishCmd, []string{}); err != nil {
			t.Fatalf("FinishCmd returned error: %v", err)
		}

		if current := currentBranchName(t, repoDir); current != "develop" {
			t.Fatalf("expected to end on develop, got %q", current)
		}

		if !branchExists(t, repoDir, "feature/publish-me") {
			t.Fatalf("expected local feature/publish-me to remain without --delete")
		}

		localRevision := strings.TrimSpace(runGitOutput(t, repoDir, "rev-parse", "refs/heads/feature/publish-me"))
		if origin := remoteRevision(t, repoDir, "feature/publish-me"); origin != localRevision {
			t.Fatalf("expected origin to hold feature/publish-me at %s, got %q", localRevision, origin)
		}

		developContent := readFile(t, filepath.Join(repoDir, "app.txt"))
		if !strings.Contains(developContent, "feature work") {
			t.Fatalf("expected develop to contain merged feature changes, got %q", developContent)
		}
	})
}

func TestFinishPublishesTheWorkBranchWithOnlyManualTargets(t *testing.T) {
	repoDir, _ := finishPublishRepo(t)

	withWorkingDir(t, repoDir, func() {
		finishPublishConfig(t, repoDir, []string{"main"}, map[string]flow.WorkflowBranchRule{
			"main": {MergeMode: "manual"},
		})

		mainBefore := strings.TrimSpace(runGitOutput(t, repoDir, "rev-parse", "refs/heads/main"))

		if err := commands.FinishCmd.RunE(commands.FinishCmd, []string{}); err != nil {
			t.Fatalf("FinishCmd returned error: %v", err)
		}

		if current := currentBranchName(t, repoDir); current != "feature/publish-me" {
			t.Fatalf("expected to stay on feature/publish-me, got %q", current)
		}

		if !remoteBranchExists(t, repoDir, "feature/publish-me") {
			t.Fatalf("expected origin to hold feature/publish-me")
		}

		mainAfter := strings.TrimSpace(runGitOutput(t, repoDir, "rev-parse", "refs/heads/main"))
		if mainAfter != mainBefore {
			t.Fatalf("expected local main to stay at %s, got %s", mainBefore, mainAfter)
		}

		if originMain := remoteRevision(t, repoDir, "main"); originMain != mainAfter {
			t.Fatalf("expected origin main to stay at %s, got %q", mainAfter, originMain)
		}
	})
}

func TestFinishSkipsThePublishWhenOriginAlreadyHoldsTheCommit(t *testing.T) {
	repoDir, _ := finishPublishRepo(t)

	withWorkingDir(t, repoDir, func() {
		// Manual-only on purpose: there is no auto target whose own push the
		// unreachable push URL would break, and this is the path where publishing is
		// the whole job.
		finishPublishConfig(t, repoDir, []string{"main"}, map[string]flow.WorkflowBranchRule{
			"main": {MergeMode: "manual"},
		})

		// The config commit lands on the work branch, so the branch is published
		// only after it: pushing earlier would leave origin behind.
		runGit(t, repoDir, "push", "-u", "origin", "feature/publish-me")
		publishedRevision := strings.TrimSpace(runGitOutput(t, repoDir, "rev-parse", "refs/heads/feature/publish-me"))

		setUnreachablePushURL(t, repoDir)

		// Succeeding is the proof: with the identity check removed, PushWorkBranch
		// would attempt the push and the unreachable push URL would reject it, so the
		// whole finish would fail.
		if err := commands.FinishCmd.RunE(commands.FinishCmd, []string{}); err != nil {
			t.Fatalf("FinishCmd returned error: %v", err)
		}

		if origin := remoteRevision(t, repoDir, "feature/publish-me"); origin != publishedRevision {
			t.Fatalf("expected feature/publish-me to stay at %s on origin, got %q", publishedRevision, origin)
		}

		if current := currentBranchName(t, repoDir); current != "feature/publish-me" {
			t.Fatalf("expected the manual-only path to stay on feature/publish-me, got %q", current)
		}
	})
}

func TestFinishPublishesNewCommitsOnAnAlreadyPublishedWorkBranch(t *testing.T) {
	repoDir, _ := finishPublishRepo(t)

	runGit(t, repoDir, "push", "-u", "origin", "feature/publish-me")
	writeFileAndCommit(t, repoDir, "app.txt", "feature work again\n", "more feature work")

	withWorkingDir(t, repoDir, func() {
		finishPublishConfig(t, repoDir, []string{"develop"}, map[string]flow.WorkflowBranchRule{
			"develop": {MergeMode: "auto"},
		})

		// Captured after the config commit, which lands on the work branch: this is
		// the revision origin must end up holding once the push this test requires
		// has happened.
		newRevision := strings.TrimSpace(runGitOutput(t, repoDir, "rev-parse", "refs/heads/feature/publish-me"))

		if err := commands.FinishCmd.RunE(commands.FinishCmd, []string{}); err != nil {
			t.Fatalf("FinishCmd returned error: %v", err)
		}

		if origin := remoteRevision(t, repoDir, "feature/publish-me"); origin != newRevision {
			t.Fatalf("expected origin to hold the new revision %s, got %q", newRevision, origin)
		}
	})
}

func TestFinishFailsWhenTheWorkBranchCannotBePublished(t *testing.T) {
	repoDir, _ := finishPublishRepo(t)

	setUnreachablePushURL(t, repoDir)

	withWorkingDir(t, repoDir, func() {
		finishPublishConfig(t, repoDir, []string{"develop"}, map[string]flow.WorkflowBranchRule{
			"develop": {MergeMode: "auto"},
		})

		developBefore := remoteRevision(t, repoDir, "develop")

		err := commands.FinishCmd.RunE(commands.FinishCmd, []string{})
		if err == nil {
			t.Fatalf("expected FinishCmd to fail when the work branch cannot be published")
		}
		if !strings.Contains(err.Error(), "failed to push branch") {
			t.Fatalf("expected the error to name the failed push, got: %v", err)
		}

		if current := currentBranchName(t, repoDir); current != "feature/publish-me" {
			t.Fatalf("expected to stay on feature/publish-me, got %q", current)
		}

		if origin := remoteRevision(t, repoDir, "develop"); origin != developBefore {
			t.Fatalf("expected origin develop to stay at %s, got %q", developBefore, origin)
		}
	})
}

func TestFinishNoPushKeepsTheWorkBranchLocal(t *testing.T) {
	repoDir, _ := finishPublishRepo(t)

	withWorkingDir(t, repoDir, func() {
		finishPublishConfig(t, repoDir, []string{"develop"}, map[string]flow.WorkflowBranchRule{
			"develop": {MergeMode: "auto"},
		})

		if err := commands.FinishCmd.Flags().Set("no-push", "true"); err != nil {
			t.Fatalf("failed to enable no-push flag: %v", err)
		}
		defer func() {
			_ = commands.FinishCmd.Flags().Set("no-push", "false")
		}()

		if err := commands.FinishCmd.RunE(commands.FinishCmd, []string{}); err != nil {
			t.Fatalf("FinishCmd returned error: %v", err)
		}

		if origin := remoteRevision(t, repoDir, "feature/publish-me"); origin != "" {
			t.Fatalf("expected --no-push to keep the work branch local, got origin at %q", origin)
		}

		if current := currentBranchName(t, repoDir); current != "develop" {
			t.Fatalf("expected to end on develop, got %q", current)
		}

		developContent := readFile(t, filepath.Join(repoDir, "app.txt"))
		if !strings.Contains(developContent, "feature work") {
			t.Fatalf("expected develop to contain merged feature changes, got %q", developContent)
		}
	})
}
