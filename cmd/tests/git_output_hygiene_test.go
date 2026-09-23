package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yepizrene-devoost/dflow/cmd/gitutils"
	"github.com/yepizrene-devoost/dflow/cmd/utils"
)

// The tests below cover the git-output-hygiene work unit: Git's own output must
// not reach the caller's stdout through a raw passthrough, and a Git failure
// must explain itself instead of returning a bare exit status.
//
// They reuse the existing fixtures (initTempGitRepo, runGit, startCLIRawOutput,
// buildDflowCLI) and, where the contract is observable only at the Git boundary,
// assert at that boundary.

// TestFailingGitOperationMessageCarriesGitExplanation guards WU1: the passthrough
// sites used to wire the child's streams to the CLI's own, so a failure returned
// only "exit status 1" while Git's reason went straight to the terminal.
//
// The trigger is a checkout that Git refuses because it would overwrite a local
// modification. It is deterministic, it reaches `dflow start` through
// gitutils.Checkout, and Git's own explanation is the only honest source of the
// reason.
func TestFailingGitOperationMessageCarriesGitExplanation(t *testing.T) {
	setUpCLIEnv(t)
	binary := buildDflowCLI(t)

	repo := initTempGitRepo(t)
	writeFileAndCommit(t, repo, "f.txt", "base\n", "add f.txt")
	runGit(t, repo, "checkout", "-b", "side")
	writeFileAndCommit(t, repo, "f.txt", "side\n", "side change")
	runGit(t, repo, "checkout", "main")

	// A tracked local modification makes `git checkout side` refuse to switch.
	if err := os.WriteFile(filepath.Join(repo, "f.txt"), []byte("dirty\n"), 0644); err != nil {
		t.Fatalf("failed to dirty f.txt: %v", err)
	}

	withWorkingDir(t, repo, func() {
		if err := utils.SaveConfig(startTestConfig()); err != nil {
			t.Fatalf("failed to save config: %v", err)
		}
	})
	runGit(t, repo, "add", ".dflow.yaml")
	runGit(t, repo, "commit", "-m", "add dflow config")

	output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "start", "feat", "demo", "--from", "side", "--no-push")
	if exitCode == 0 {
		t.Fatalf("blocked checkout exited 0, want non-zero\n%s", output)
	}
	if !strings.Contains(output, "would be overwritten by checkout") {
		t.Fatalf("failure message must carry Git's own explanation, got:\n%s", output)
	}
	if strings.Contains(output, `failed to checkout branch "side": exit status 1`) {
		t.Fatalf("failure message still carries only the exit status:\n%s", output)
	}
}

// TestMergeConflictErrorCarriesGitExplanation covers the merge face of the same
// WU1 defect: a conflicted `git merge` returned a bare exit status while its
// explanation (the CONFLICT lines) went to the caller's stdout.
//
// The conflict is produced exactly as `dflow finish` reaches it, but asserted at
// the gitutils boundary, because cmd/commands/finish.go deliberately replaces
// this error with its own conflict message while a merge is in progress. The
// Git-level reason is therefore observable on the error that WU1 owns.
func TestMergeConflictErrorCarriesGitExplanation(t *testing.T) {
	repo := initTempGitRepo(t)

	withWorkingDir(t, repo, func() {
		writeFileAndCommit(t, repo, "conflict.txt", "base\n", "base")
		runGit(t, repo, "checkout", "-b", "feature/conflict")
		writeFileAndCommit(t, repo, "conflict.txt", "feature\n", "feature change")
		runGit(t, repo, "checkout", "main")
		writeFileAndCommit(t, repo, "conflict.txt", "main\n", "main change")

		err := gitutils.MergeBranchIntoCurrent("feature/conflict")
		if err == nil {
			t.Fatalf("expected the merge to fail with a conflict")
		}

		message := err.Error()
		if !strings.Contains(message, "CONFLICT") {
			t.Fatalf("merge error must carry Git's conflict explanation, got:\n%s", message)
		}
		if !strings.Contains(message, "conflict.txt") {
			t.Fatalf("merge error must name the conflicting path, got:\n%s", message)
		}

		runGit(t, repo, "merge", "--abort")
	})
}

// TestDeleteCurrentBranchIsRefusedWithDflowMessage guards WU2: deleting the
// branch you are on is a condition dflow can check without asking Git, so the
// refusal must be dflow's own message and the branch must survive.
func TestDeleteCurrentBranchIsRefusedWithDflowMessage(t *testing.T) {
	setUpCLIEnv(t)
	binary := buildDflowCLI(t)

	repo := initTempGitRepo(t)
	runGit(t, repo, "checkout", "-b", "feature/current")

	output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "delete", "feature/current", "--yes")
	if exitCode == 0 {
		t.Fatalf("deleting the current branch exited 0, want non-zero\n%s", output)
	}
	if !strings.Contains(output, "feature/current") {
		t.Fatalf("refusal must name the branch:\n%s", output)
	}
	if !strings.Contains(output, "currently on") {
		t.Fatalf("refusal must say the caller is on the branch, got:\n%s", output)
	}
	// Git's own wording for this case mentions a worktree; the refusal must be
	// dflow's, not that passthrough.
	if strings.Contains(output, "used by worktree") {
		t.Fatalf("refusal surfaced Git's verbatim message instead of dflow's:\n%s", output)
	}
	if !branchExists(t, repo, "feature/current") {
		t.Fatalf("the refused branch must still exist")
	}
}

// TestStartDoesNotLeakGitText guards the stdout side of WU1. The raw stream
// passthrough used to print Git's switch confirmations and branch-status advice
// into the caller's captured output; the issue calls out `Your branch is ahead
// of 'origin/develop'...` specifically, which Git writes to stdout. None of it
// may reach the caller, while dflow's own success line stays.
func TestStartDoesNotLeakGitText(t *testing.T) {
	setUpCLIEnv(t)
	binary := buildDflowCLI(t)

	repo := initTempGitRepo(t)
	remote := initBareGitRepo(t)
	runGit(t, repo, "checkout", "-b", "develop")
	writeFileAndCommit(t, repo, "develop.txt", "develop\n", "seed develop")
	runGit(t, repo, "remote", "add", "origin", remote)
	runGit(t, repo, "push", "-u", "origin", "main")
	runGit(t, repo, "push", "-u", "origin", "develop")
	// Leave develop ahead of origin so Git would print its branch-status advice
	// on checkout, which is exactly the leak the issue reports.
	writeFileAndCommit(t, repo, "ahead.txt", "ahead\n", "ahead of origin")
	runGit(t, repo, "checkout", "main")

	withWorkingDir(t, repo, func() {
		if err := utils.SaveConfig(startTestConfig()); err != nil {
			t.Fatalf("failed to save config: %v", err)
		}
	})
	runGit(t, repo, "add", ".dflow.yaml")
	runGit(t, repo, "commit", "-m", "add dflow config")

	output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "start", "feat", "demo", "--no-push")
	if exitCode != 0 {
		t.Fatalf("successful start exited %d, want 0\n%s", exitCode, output)
	}
	for _, leaked := range []string{"Switched to a new branch", "Switched to branch", "Already on ", "Your branch is"} {
		if strings.Contains(output, leaked) {
			t.Fatalf("Git text %q reached dflow's output:\n%s", leaked, output)
		}
	}
	if !strings.Contains(output, "Created and switched to branch 'feature/demo'") {
		t.Fatalf("dflow's own success line is missing:\n%s", output)
	}
}

// TestFinishSuccessPreservesGitMergeSummary guards the deliberate trade-off of
// WU1: Git's stderr is discarded as noise, but the merge summary and its diffstat
// (which Git writes to stdout) must still be reported, through the shared output
// helper rather than lost.
func TestFinishSuccessPreservesGitMergeSummary(t *testing.T) {
	setUpCLIEnv(t)
	binary := buildDflowCLI(t)

	repo := initTempGitRepo(t)
	remote := initBareGitRepo(t)
	runGit(t, repo, "checkout", "-b", "develop")
	runGit(t, repo, "remote", "add", "origin", remote)
	runGit(t, repo, "push", "-u", "origin", "main")
	runGit(t, repo, "push", "-u", "origin", "develop")

	withWorkingDir(t, repo, func() {
		if err := utils.SaveConfig(startTestConfig()); err != nil {
			t.Fatalf("failed to save config: %v", err)
		}
	})
	runGit(t, repo, "add", ".dflow.yaml")
	runGit(t, repo, "commit", "-m", "add dflow config")

	runGit(t, repo, "checkout", "-b", "feature/merge")
	writeFileAndCommit(t, repo, "added.txt", "new file\n", "feature add")

	output, exitCode := startCLIRawOutput(t, 20*time.Second, repo, binary, "finish")
	if exitCode != 0 {
		t.Fatalf("successful finish exited %d, want 0\n%s", exitCode, output)
	}
	if !strings.Contains(output, "added.txt") || !strings.Contains(output, "1 file changed") {
		t.Fatalf("merge diffstat must be preserved through the output helper:\n%s", output)
	}
}

// TestFinishMergeFailureCarriesGitExplanation proves the WU1 diagnostic
// attachment end to end through the real binary: a merge that fails inside
// `dflow finish` exits non-zero and the reported message carries Git's own
// explanation instead of a bare exit status.
//
// The content-conflict variant is deliberately not used here:
// cmd/commands/finish.go replaces a merge error with its own conflict message
// whenever a merge is in progress, and that file is outside this work unit. An
// unrelated-history source branch fails the merge without leaving a merge in
// progress, so the wrapped error reaches the caller, which is exactly the path
// WU1 owns. The conflict wording itself is asserted in
// TestMergeConflictErrorCarriesGitExplanation.
func TestFinishMergeFailureCarriesGitExplanation(t *testing.T) {
	setUpCLIEnv(t)
	binary := buildDflowCLI(t)

	repo := initTempGitRepo(t)
	remote := initBareGitRepo(t)

	withWorkingDir(t, repo, func() {
		if err := utils.SaveConfig(startTestConfig()); err != nil {
			t.Fatalf("failed to save config: %v", err)
		}
	})
	runGit(t, repo, "add", ".dflow.yaml")
	runGit(t, repo, "commit", "-m", "add dflow config")
	runGit(t, repo, "checkout", "-b", "develop")
	runGit(t, repo, "remote", "add", "origin", remote)
	runGit(t, repo, "push", "-u", "origin", "main")
	runGit(t, repo, "push", "-u", "origin", "develop")

	// An orphan branch shares no history with develop, so the merge cannot be a
	// fast-forward and Git refuses it for a stated reason. The working tree is
	// rebuilt from scratch so the finish clean-tree guard still passes.
	runGit(t, repo, "checkout", "--orphan", "feature/orphan")
	runGit(t, repo, "rm", "-rf", ".")
	withWorkingDir(t, repo, func() {
		if err := utils.SaveConfig(startTestConfig()); err != nil {
			t.Fatalf("failed to save orphan config: %v", err)
		}
	})
	runGit(t, repo, "add", ".dflow.yaml")
	runGit(t, repo, "commit", "-m", "orphan feature")

	output, exitCode := startCLIRawOutput(t, 20*time.Second, repo, binary, "finish")
	if exitCode == 0 {
		t.Fatalf("failing merge exited 0, want non-zero\n%s", output)
	}
	if !strings.Contains(output, "refusing to merge unrelated histories") {
		t.Fatalf("failure message must carry Git's own explanation, got:\n%s", output)
	}
}

// setUpCLIEnv pins the environment every real-binary test needs: no user or
// system Git config, no interactive prompt, and cwd-based config loading.
func setUpCLIEnv(t *testing.T) {
	t.Helper()

	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_TERMINAL_PROMPT", "0")
	t.Setenv("DFLOW_CWD", "")
}
