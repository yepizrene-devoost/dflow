// Package gitutils provides low-level Git utility functions used by dflow commands.
//
// These helpers wrap common Git operations such as checking out branches,
// creating new ones, pushing to origin, and pulling updates.
package gitutils

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/yepizrene-devoost/dflow/cmd/utils"
)

// CheckOrCreateBranch verifies whether the given branch exists locally.
//
// If the branch does not exist, it creates it using `git branch <branch>`.
// This operation does not switch to the branch; it only ensures its presence.
func CheckOrCreateBranch(branch string) {
	cmd := exec.Command("git", "rev-parse", "--verify", branch)
	if err := cmd.Run(); err != nil {
		fmt.Printf("ℹ️  Branch '%s' does not exist. Creating...\n", branch)
		create := exec.Command("git", "branch", branch)
		if err := create.Run(); err != nil {
			fmt.Printf("❌ Failed to create branch '%s': %v\n", branch, err)
			return
		}
		fmt.Printf("✅ Created branch '%s'\n", branch)
	} else {
		fmt.Printf("✔ Branch '%s' exists\n", branch)
	}
}

// PushBranch pushes the specified branch to the remote `origin` and sets upstream tracking.
//
// This wraps the command `git push -u origin <branch>`.
// It logs success or failure to the console but does not return an error.
func PushBranch(branch string) {
	cmd := exec.Command("git", "push", "-u", "origin", branch)
	if err := cmd.Run(); err != nil {
		fmt.Printf("❌ Failed to push branch '%s': %v\n", branch, err)
	} else {
		fmt.Printf("🚀 Pushed branch '%s' to remote\n", branch)
	}
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
	cmd := exec.Command("git", "branch", "-D", branch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("❌ failed to delete local branch: %w", err)
	}

	//delete remote branch
	cmd = exec.Command("git", "push", "origin", "--delete", branch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("❌ failed to delete remote branch: %w", err)
	}

	utils.Success("Branch '%s' deleted locally and remotely. ", branch)

	return nil
}
