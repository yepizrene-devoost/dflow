// Package gitutils provides low-level Git utility functions used by dflow commands.
//
// These helpers wrap common Git operations such as checking out branches,
// creating new ones, pushing to origin, and pulling updates.
package gitutils

import (
	"fmt"
	"os"
	"os/exec"
)

// CheckOrCreateBranch verifies whether the given branch exists locally.
// If it doesn't, it creates the branch using `git branch <name>`.
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

// PushBranch pushes the specified branch to the remote `origin` with tracking enabled.
func PushBranch(branch string) {
	cmd := exec.Command("git", "push", "-u", "origin", branch)
	if err := cmd.Run(); err != nil {
		fmt.Printf("❌ Failed to push branch '%s': %v\n", branch, err)
	} else {
		fmt.Printf("🚀 Pushed branch '%s' to remote\n", branch)
	}
}

// Checkout switches the working tree to the specified branch using `git checkout`.
func Checkout(branch string) error {
	cmd := exec.Command("git", "checkout", branch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// CheckoutNew creates and switches to a new branch from the current HEAD.
func CheckoutNew(branch string) error {
	cmd := exec.Command("git", "checkout", "-b", branch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// function to pull branch
func Pull() error {
	cmd := exec.Command("git", "pull")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
