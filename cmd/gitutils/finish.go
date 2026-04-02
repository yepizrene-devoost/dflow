package gitutils

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/yepizrene-devoost/dflow/cmd/utils"
)

// CurrentBranch returns the currently checked out Git branch name.
func CurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to determine current branch: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// IsWorkingTreeClean reports whether the repository has no staged, unstaged,
// or untracked changes.
func IsWorkingTreeClean() (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to inspect working tree: %w", err)
	}

	return len(bytes.TrimSpace(output)) == 0, nil
}

// EnsureWorkingTreeClean returns an error when the working tree is dirty.
func EnsureWorkingTreeClean() error {
	clean, err := IsWorkingTreeClean()
	if err != nil {
		return err
	}
	if !clean {
		return fmt.Errorf("working tree is not clean; commit or stash your changes before continuing")
	}
	return nil
}

// MergeInProgress reports whether Git currently has an unfinished merge.
func MergeInProgress() bool {
	cmd := exec.Command("git", "rev-parse", "-q", "--verify", "MERGE_HEAD")
	return cmd.Run() == nil
}

// MergeBranchIntoCurrent merges the source branch into the current branch.
func MergeBranchIntoCurrent(sourceBranch string) error {
	cmd := exec.Command("git", "merge", "--no-ff", "--no-edit", sourceBranch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to merge branch %q into current branch: %w", sourceBranch, err)
	}
	return nil
}

// AbortMerge aborts an in-progress merge.
func AbortMerge() error {
	cmd := exec.Command("git", "merge", "--abort")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to abort merge: %w", err)
	}
	return nil
}

// PushBranchUpdate pushes the given branch to origin without changing upstream tracking.
func PushBranchUpdate(branch string) error {
	if !HasOriginRemote() {
		utils.Info("📁   Remote 'origin' not found. Skipping push for '%s'.", branch)
		return nil
	}

	spinner := utils.NewSpinner(fmt.Sprintf("Pushing updates for '%s' to origin...", branch))
	spinner.Start()

	cmd := exec.Command("git", "push", "origin", branch)
	if err := cmd.Run(); err != nil {
		spinner.Stop("Failed to push branch updates.")
		return fmt.Errorf("failed to push branch %q: %w", branch, err)
	}

	spinner.Stop(fmt.Sprintf("Pushed updates for '%s'.", branch), "🚀")
	return nil
}
