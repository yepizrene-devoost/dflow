// Package gitutils provides low-level Git utility functions used by dflow commands.
//
// These helpers wrap common Git operations such as checking out branches,
// creating new ones, pushing to origin, and pulling updates. This package is
// used internally by dflow to manage Git workflows programmatically.
package gitutils

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/yepizrene-devoost/dflow/cmd/utils"
)

// CheckOrCreateBranch verifies whether the given branch exists locally.
//
// If the branch does not exist, it creates it using `git branch <branch>`.
// This operation does not switch to the branch; it only ensures its presence.
func CheckOrCreateBranch(branch string) error {
	cmd := exec.Command("git", "rev-parse", "--verify", branch)
	if err := cmd.Run(); err != nil {
		utils.Info("Branch '%s' does not exist. Creating...", branch)
		create := exec.Command("git", "branch", branch)
		if err := create.Run(); err != nil {
			return fmt.Errorf("failed to create branch '%s': %w", branch, err)
		}
		utils.Success("Created branch '%s'", branch)
	} else {
		utils.Icon("✔", "Branch '%s' exists", branch)
	}

	return nil
}

// PushBranch pushes the specified branch to the remote 'origin' and sets upstream tracking.
//
// This wraps the command `git push -u origin <branch>` and logs the result
// to the console. It returns an error if the push operation fails.
func PushBranch(branch string) error {
	spinner := utils.NewSpinner(fmt.Sprintf("Pushing branch '%s' to origin...", branch))
	spinner.Start()

	cmd := exec.Command("git", "push", "-u", "origin", branch)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to push branch '%s': %w", branch, err)
	}
	spinner.Stop(fmt.Sprintf("Pushed branch '%s' to remote", branch), "🚀")

	return nil
}

// runCapturingGit runs cmd with both streams captured, so Git's own text never
// reaches the caller's stdout directly.
//
// On failure the captured diagnostics are attached to the returned error, because
// the reason a Git command failed is exactly what the caller needs and the exit
// status alone never carries it. On success the captured stderr is discarded as
// progress and advice noise, and any captured stdout is reported through the shared
// CLI output helper, so it stays inside the single output choke point and a
// machine-readable mode can silence it.
func runCapturingGit(cmd *exec.Cmd) error {
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		diagnostics := strings.TrimSpace(stderr.String())
		if diagnostics == "" {
			// Some failures explain themselves on stdout instead (a merge conflict,
			// for example); prefer stderr so a failure is never bare.
			diagnostics = strings.TrimSpace(stdout.String())
		}
		if diagnostics == "" {
			return err
		}
		return fmt.Errorf("%s: %w", diagnostics, err)
	}

	// A successful command's stderr is progress and advice noise and is dropped.
	// Its stdout is the operation's result, so it goes through the shared output
	// helper one line at a time instead of straight to the caller's stdout.
	captured := strings.TrimSpace(stdout.String())
	if captured == "" {
		return nil
	}
	for _, line := range strings.Split(captured, "\n") {
		utils.Plain("%s", strings.TrimRight(line, "\r"))
	}

	return nil
}

// Checkout switches the working directory to the given branch using `git checkout <branch>`.
//
// Git's own output is captured rather than wired to the CLI's streams, so a
// failed checkout reports why instead of only its exit status. `--quiet` keeps
// the successful checkout silent at the source: Git writes its branch-status
// advice ("Your branch is ahead of 'origin/develop'...") to stdout, which is not
// a result dflow asked for and must not reach the caller's stdout.
//
// Returns an error if the checkout operation fails.
func Checkout(branch string) error {
	return runCapturingGit(exec.Command("git", "checkout", "--quiet", branch))
}

// CheckoutNew creates and checks out a new branch from the current HEAD.
//
// It wraps `git checkout -b <branch>`. Git's own output is captured rather than
// wired to the CLI's streams, and `--quiet` keeps a successful switch silent at
// the source so its status advice never reaches the caller's stdout.
//
// It returns an error if the operation fails.
func CheckoutNew(branch string) error {
	return runCapturingGit(exec.Command("git", "checkout", "--quiet", "-b", branch))
}

// Pull updates the current branch with the latest changes from the remote 'origin'.
//
// If the remote 'origin' is not configured, the function skips the pull and
// assumes the local branch is up-to-date. This is useful for local-only Git
// repositories where no remote is defined.
//
// It returns an error only if the pull fails when attempted.
func Pull() error {
	if !HasOriginRemote() {
		utils.Icon("📁", "Remote 'origin' not found. Skipping pull. Using local branch as latest.")
		return nil
	}

	spinner := utils.NewSpinner("Pulling latest changes from origin...")
	spinner.Start()

	cmd := exec.Command("git", "pull")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		spinner.Clear()
		return err
	}

	spinner.Stop("Repository updated.")
	return nil
}

// Delete removes the given Git branch both locally and remotely.
//
// It executes `git branch -D <branch>` to delete the local branch,
// and `git push origin --delete <branch>` to remove the branch from the remote repository.
//
// If both operations succeed, it logs a success message via utils.Success.
// Returns an error if either operation fails.
func Delete(branch string) error {
	// Refuse before anything is announced: this is a condition dflow can check
	// without asking Git, and letting Git answer would surface its refusal as a
	// foreign message. A branch checked out in another worktree stays Git's call,
	// because only Git knows about it.
	if current, err := CurrentBranch(); err == nil && current == branch {
		return fmt.Errorf("cannot delete branch '%s' because it is the branch you are currently on", branch)
	}

	spinner := utils.NewSpinner(fmt.Sprintf("Deleting branch '%s' locally and remotely...", branch))
	spinner.Start()

	var stderr bytes.Buffer
	cmd := exec.Command("git", "branch", "-D", branch)
	cmd.Stderr = &stderr
	cmd.Stdout = nil
	if err := cmd.Run(); err != nil {
		spinner.Clear()
		return fmt.Errorf("failed to delete local branch '%s': %s", branch, stderr.String())
	}

	if RemoteBranchExists(branch) {
		cmd = exec.Command("git", "push", "origin", "--delete", branch)
		cmd.Stdout = nil
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			spinner.Clear()
			return fmt.Errorf("failed to delete remote branch: %s", stderr.String())
		}
	} else {
		spinner.Stop(fmt.Sprintf("Branch '%s' deleted locally.", branch), "🗑️")
		utils.Info("Remote branch '%s' does not exist. Skipping remote deletion.", branch)
		return nil
	}

	spinner.Stop(fmt.Sprintf("Branch '%s' deleted locally and remotely.", branch), "🗑️")
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

// HasOriginRemote checks whether the Git remote named 'origin' is configured.
//
// It executes `git remote get-url origin`, returning true if the command
// succeeds, meaning the 'origin' remote exists and has a valid URL.
//
// This is useful to avoid pull/push errors in local-only Git repositories.
func HasOriginRemote() bool {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	err := cmd.Run()
	return err == nil
}
