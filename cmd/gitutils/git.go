// Package gitutils provides low-level Git utility functions used by dflow commands.
//
// These helpers wrap common Git operations such as checking out branches,
// creating new ones, pushing to origin, and pulling updates.
package gitutils

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/yepizrene-devoost/dflow/cmd/utils"
)

// CheckOrCreateBranch verifies whether the given branch exists locally.
//
// If the branch does not exist, it creates it using `git branch <branch>`.
// This operation does not switch to the branch; it only ensures its presence.
func CheckOrCreateBranch(branch string) error {
	cmd := exec.Command("git", "rev-parse", "--verify", branch)
	if err := cmd.Run(); err != nil {
		fmt.Printf("ℹ️  Branch '%s' does not exist. Creating...\n", branch)
		create := exec.Command("git", "branch", branch)
		if err := create.Run(); err != nil {
			return fmt.Errorf("❌ failed to create branch '%s': %w", branch, err)
		}
		fmt.Printf("✅ Created branch '%s'\n", branch)
	} else {
		fmt.Printf("✔ Branch '%s' exists\n", branch)
	}

	return nil
}

// PushBranch pushes the specified branch to the remote `origin` and sets upstream tracking.
//
// This wraps the command `git push -u origin <branch>`.
// It logs success or failure to the console but does not return an error.
func PushBranch(branch string) error {
	cmd := exec.Command("git", "push", "-u", "origin", branch)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("❌ failed to push branch '%s': %w", branch, err)
	} else {
		fmt.Printf("🚀 Pushed branch '%s' to remote\n", branch)
	}

	return nil
}

// Checkout switches the working directory to the given branch using `git checkout <branch>`.
//
// Returns an error if the checkout operation fails.
func Checkout(branch string) error {
	cmd := exec.Command("git", "checkout", branch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// CheckoutNew creates and checks out a new branch from the current HEAD.
//
// It wraps `git checkout -b <branch>` and returns an error if the operation fails.
func CheckoutNew(branch string) error {
	cmd := exec.Command("git", "checkout", "-b", branch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Pull pulls the latest changes from the remote for the current branch.
//
// It executes `git pull` and returns an error if the command fails.
func Pull() error {
	cmd := exec.Command("git", "pull")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Delete removes the given Git branch both locally and remotely.
//
// It executes `git branch -D <branch>` to delete the local branch,
// and `git push origin --delete <branch>` to remove the branch from the remote repository.
//
// If both operations succeed, it logs a success message via utils.Success.
// Returns an error if either operation fails.
func Delete(branch string) error {
	// delete local branch
	var stderr bytes.Buffer
	cmd := exec.Command("git", "branch", "-D", branch)
	cmd.Stderr = &stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("❌ failed to delete local branch '%s': %s", branch, stderr.String())
	}

	// Check if remote branch exists before attempting to delete
	if RemoteBranchExists(branch) {
		cmd = exec.Command("git", "push", "origin", "--delete", branch)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("❌ failed to delete remote branch: %w", err)
		}
		fmt.Printf("🌐 Remote branch '%s' deleted.\n", branch)
	} else {
		utils.Info("Remote branch '%s' does not exist. Skipping remote deletion.", branch)
	}

	utils.Success("Branch '%s' deleted locally and remotely (if existed).", branch)

	return nil
}

// RemoteBranchExists checks if a branch exists on the remote `origin`.
//
// It runs `git ls-remote --heads origin <branch>` and returns true if the branch exists.
func RemoteBranchExists(branch string) bool {
	cmd := exec.Command("git", "ls-remote", "--heads", "origin", branch)
	output, err := cmd.Output()
	return err == nil && len(output) > 0
}

// GetLocalBranches returns a list of local Git branch names.
//
// It runs `git branch --format=%(refname:short)` and parses the output line by line.
func GetLocalBranches() []string {
	cmd := exec.Command("git", "branch", "--format=%(refname:short)")
	out, err := cmd.Output()
	if err != nil {
		return []string{}
	}

	lines := bytes.Split(out, []byte("\n"))
	var branches []string
	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) > 0 {
			branches = append(branches, string(trimmed))
		}
	}

	return branches
}
