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
// The two copies are independent, so the deletion is idempotent: each copy is
// deleted when it exists, and a copy that is already gone is not an error. A
// half-finished delete, a network failure in the middle of one, or a local
// branch removed by hand must not block the copy that still exists.
//
// It executes `git branch -D <branch>` for an existing local branch and
// `git push origin --delete <branch>` for an existing remote one, in that order.
// It returns an error when neither copy exists, when the remote lookup cannot
// determine existence, and when deleting a copy that does exist failed. A remote
// failure after a successful local deletion names the remote branch that remains,
// and a failed lookup is never reported as an absent remote: with a local copy it
// is deleted first and the lookup error names what remains unknown, and with no
// local copy nothing is touched.
func Delete(branch string) error {
	// Refuse before anything is announced: this is a condition dflow can check
	// without asking Git, and letting Git answer would surface its refusal as a
	// foreign message. A branch checked out in another worktree stays Git's call,
	// because only Git knows about it.
	//
	// A failed lookup deliberately does not fail the deletion: when the current
	// branch cannot be determined the guard stands aside and lets Git decide,
	// because propagating that error would turn a deletion Git would have allowed
	// into a refusal caused by an unrelated problem.
	if current, err := CurrentBranch(); err == nil && current == branch {
		return fmt.Errorf("cannot delete branch '%s' because it is the branch you are currently on", branch)
	}

	// Decide existence once, before either copy is touched: that is what keeps the
	// local half untouched when only the remote branch is left. A branch absent
	// from both places is the one case with nothing to delete, and reporting
	// success there would be a lie.
	localExisted := BranchExists(branch)
	remoteExisted, remoteErr := RemoteBranchExists(branch)

	// The two outcomes that touch nothing are settled before the spinner exists,
	// which is what leaves the operation below a single creation site: with no local
	// copy the lookup failure is the whole answer and neither copy is touched, and a
	// branch absent from both places is the one case with nothing to delete.
	if remoteErr != nil && !localExisted {
		return remoteErr
	}

	if !localExisted && !remoteExisted {
		return fmt.Errorf("branch '%s' does not exist locally or on origin; nothing to delete", branch)
	}

	// One creation site and one termination: Stop and Clear share the spinner's
	// stopOnce guard, so the deferred Clear is the only terminator on every failure
	// path below and a no-op on the success paths, which call Stop first.
	spinner := utils.NewSpinner(fmt.Sprintf("Deleting branch '%s' locally and remotely...", branch))
	spinner.Start()
	defer spinner.Clear()

	// The "remote could not be checked" outcome is settled here, as one explicit
	// branch, instead of a return dropped between the two deletions below: with a
	// local copy, the plan is to delete that copy and then report the failure naming
	// the half that is gone.
	if remoteErr != nil {
		// The remote half is unknown, so the local half is the only one this call can
		// finish. Deleting it keeps the two halves independent, and the error below
		// reports the operation as unfinished: the local half is gone and the remote
		// half could not be checked.
		if err := deleteLocalBranch(branch); err != nil {
			return err
		}

		return fmt.Errorf("deleted local branch '%s' but %w", branch, remoteErr)
	}

	// The local half goes first, and it is not rolled back: once `git branch -D` has
	// removed the branch it is gone, so a failure in a later half leaves the
	// operation finished locally and unfinished overall. Every failure after this
	// point therefore exits non-zero while naming the half this call already
	// deleted; a partial success is unfinished work, never a no-op and never a
	// success.
	if localExisted {
		if err := deleteLocalBranch(branch); err != nil {
			return err
		}
	}

	if remoteExisted {
		var stderr bytes.Buffer
		cmd := exec.Command("git", "push", "origin", "--delete", branch)
		cmd.Stdout = nil
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return remoteDeleteFailure(branch, localExisted, stderr.String())
		}
	}

	switch {
	case localExisted && remoteExisted:
		spinner.Stop(fmt.Sprintf("Branch '%s' deleted locally and remotely.", branch), "🗑️")
	case localExisted:
		spinner.Stop(fmt.Sprintf("Branch '%s' deleted locally.", branch), "🗑️")
		utils.Info("Remote branch '%s' does not exist. Skipping remote deletion.", branch)
	case remoteExisted:
		// Explicit instead of a default: only the remote copy existed, and saying so
		// must not depend on knowing which outcome arms came before this one.
		spinner.Stop(fmt.Sprintf("Branch '%s' deleted remotely.", branch), "🗑️")
		utils.Info("Local branch '%s' does not exist. Skipping local deletion.", branch)
	}

	return nil
}

// remoteDeleteFailure reports a `git push origin --delete <branch>` that failed.
//
// Both message variants are chosen here, once, so no caller can drift from the
// rule that decides between them. `localExisted` is the observation taken before
// either half was touched, so a true value means this same call already deleted
// the local branch and the report must name the half that remains: letting the
// caller read the failure as "nothing was deleted" would hide a real deletion.
//
// Each variant keeps its exact wording, Git's captured diagnostics included.
func remoteDeleteFailure(branch string, localExisted bool, diagnostics string) error {
	// The remote half's sentence is built once and the other variant composes over
	// it, so the wording of `failed to delete remote branch '<branch>'` exists in a
	// single place and the two variants cannot drift apart in what they say.
	remoteFailure := fmt.Errorf("failed to delete remote branch '%s': %s", branch, strings.TrimSpace(diagnostics))
	if !localExisted {
		return remoteFailure
	}

	// No `%w` here: this variant has never unwrapped to the remote sentence, and
	// sharing the wording must not add an error chain the callers never saw.
	return fmt.Errorf("deleted local branch '%s' but %s", branch, remoteFailure)
}

// deleteLocalBranch removes the local branch with `git branch -D` and reports
// Git's refusal in dflow's own words. It is a named step so the two call sites
// that may need the local half deleted (the ordinary path and the path where the
// remote half could not be checked) cannot drift apart in what they report.
func deleteLocalBranch(branch string) error {
	var stderr bytes.Buffer
	cmd := exec.Command("git", "branch", "-D", branch)
	cmd.Stdout = nil
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete local branch '%s': %s", branch, strings.TrimSpace(stderr.String()))
	}

	return nil
}

// remoteBranchRevision returns the commit `origin` holds for the branch, or an
// empty string when origin has no such branch.
//
// It runs `git ls-remote --heads origin <branch>` and keeps three states apart,
// which is what RemoteBranchExists is defined over:
//
//   - No `origin` remote is configured. No remote copy of any branch can exist,
//     which is a known absence determinable locally, so the answer is `"", nil`
//     and no network call is made.
//   - `origin` is configured but the lookup fails (unreachable, bad URL, auth).
//     That is the unknown case, and it returns an error carrying git's own
//     diagnostics instead of pretending the branch is absent.
//   - Otherwise the revision is origin's commit for the branch, empty when origin
//     does not have it.
//
// The lookup's stderr is captured into the error rather than wired to the CLI's
// streams, so it never reaches stdout.
func remoteBranchRevision(branch string) (string, error) {
	// A missing origin is a known absence, not a failed check: a remote copy can
	// only live in a remote, and with no remote configured there is none to find.
	if !HasOriginRemote() {
		return "", nil
	}

	cmd := exec.Command("git", "ls-remote", "--heads", "origin", branch)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		diagnostics := strings.TrimSpace(stderr.String())
		if diagnostics == "" {
			diagnostics = strings.TrimSpace(stdout.String())
		}
		// Git's own explanation is the reason, so it ends the sentence: the existing
		// delete and merge messages do the same and never append the exit status
		// after it. `err` only carries the sentence when Git said nothing at all,
		// which is the case where the status is the entire information available.
		if diagnostics == "" {
			return "", fmt.Errorf("failed to check remote branch '%s' on origin: %w", branch, err)
		}
		return "", fmt.Errorf("failed to check remote branch '%s' on origin: %s", branch, diagnostics)
	}

	// An exact ref name yields at most one line; no line means origin does not
	// have the branch.
	fields := strings.Fields(stdout.String())
	if len(fields) == 0 {
		return "", nil
	}

	return fields[0], nil
}

// RemoteBranchExists reports whether a branch exists on the remote `origin`.
//
// Existence is the remote revision being present, so this asks the one lookup
// that the publish check also uses: the bool answers existence and the error
// answers whether that could be determined, keeping the same two states apart as
// remoteBranchRevision does (a missing `origin` is a known absence, and a
// configured but unreachable `origin` is an error carrying git's diagnostics).
func RemoteBranchExists(branch string) (bool, error) {
	revision, err := remoteBranchRevision(branch)
	if err != nil {
		return false, err
	}

	return revision != "", nil
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
