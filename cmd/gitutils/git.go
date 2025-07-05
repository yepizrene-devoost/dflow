package gitutils

import (
	"fmt"
	"os"
	"os/exec"
)

// CheckOrCreateBranch verifica si una rama existe localmente y la crea si no
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

// PushBranch publish branch to 'origin'
func PushBranch(branch string) {
	cmd := exec.Command("git", "push", "-u", "origin", branch)
	if err := cmd.Run(); err != nil {
		fmt.Printf("❌ Failed to push branch '%s': %v\n", branch, err)
	} else {
		fmt.Printf("🚀 Pushed branch '%s' to remote\n", branch)
	}
}

// function to checkout to origin branch
func Checkout(branch string) error {
	cmd := exec.Command("git", "checkout", branch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// function to create new branch
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
