package gitutils

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/yepizrene-devoost/dflow/cmd/utils"
)

// CurrentBranch returns the currently checked out Git branch name.
func CurrentBranch() (string, error) {
	session, err := newGitSession()
	if err != nil {
		return "", err
	}
	return session.CurrentBranch()
}

func (s *GitSession) CurrentBranch() (string, error) {
	output, err := s.command("rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("failed to determine current branch: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// BranchExists reports whether the given local branch exists.
func BranchExists(branch string) bool {
	session, err := newGitSession()
	return err == nil && session.BranchExists(branch)
}

func (s *GitSession) BranchExists(branch string) bool {
	return gitSucceeds(s.command("rev-parse", "--verify", "--quiet", "refs/heads/"+branch))
}

// CheckoutExistingBranch switches to an existing local branch.
func CheckoutExistingBranch(branch string) error {
	session, err := newGitSession()
	if err != nil {
		return fmt.Errorf("branch %q does not exist locally", branch)
	}
	return session.CheckoutExistingBranch(branch)
}

func (s *GitSession) CheckoutExistingBranch(branch string) error {
	if !s.BranchExists(branch) {
		return fmt.Errorf("branch %q does not exist locally", branch)
	}

	if err := runCapturingGit(s.command("checkout", "--quiet", branch)); err != nil {
		return fmt.Errorf("failed to checkout branch %q: %w", branch, err)
	}

	return nil
}

// CheckoutTrackingBranch creates a local branch that tracks origin/<branch>.
//
// Git's own output is captured rather than wired to the CLI's streams, and
// `--quiet` keeps a successful switch silent at the source so its status advice
// never reaches the caller's stdout.
func CheckoutTrackingBranch(branch string) error {
	session, err := newGitSession()
	if err != nil {
		return fmt.Errorf("failed to create tracking branch %q from origin/%s: %w", branch, branch, err)
	}
	return session.checkoutTrackingBranch(branch)
}

func (s *GitSession) checkoutTrackingBranch(branch string) error {
	if err := runCapturingGit(s.command("checkout", "--quiet", "--track", "-b", branch, "origin/"+branch)); err != nil {
		return fmt.Errorf("failed to create tracking branch %q from origin/%s: %w", branch, branch, err)
	}
	return nil
}

// IsWorkingTreeClean reports whether the repository has no staged, unstaged,
// or untracked changes.
func IsWorkingTreeClean() (bool, error) {
	session, err := newGitSession()
	if err != nil {
		return false, err
	}
	return session.IsWorkingTreeClean()
}

func (s *GitSession) IsWorkingTreeClean() (bool, error) {
	output, err := s.command("status", "--porcelain").Output()
	if err != nil {
		return false, fmt.Errorf("failed to inspect working tree: %w", err)
	}

	return len(bytes.TrimSpace(output)) == 0, nil
}

// EnsureWorkingTreeClean returns an error when the working tree is dirty.
func EnsureWorkingTreeClean() error {
	session, err := newGitSession()
	if err != nil {
		return err
	}
	return session.EnsureWorkingTreeClean()
}

func (s *GitSession) EnsureWorkingTreeClean() error {
	clean, err := s.IsWorkingTreeClean()
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
	session, err := newGitSession()
	return err == nil && session.MergeInProgress()
}

func (s *GitSession) MergeInProgress() bool {
	return gitSucceeds(s.command("rev-parse", "-q", "--verify", "MERGE_HEAD"))
}

// MergeBranchIntoCurrent merges the source branch into the current branch.
//
// Git's own output is captured rather than wired to the CLI's streams, so a
// conflict reports what Git found instead of only its exit status.
func MergeBranchIntoCurrent(sourceBranch string) error {
	session, err := newGitSession()
	if err != nil {
		return fmt.Errorf("failed to merge branch %q into current branch: %w", sourceBranch, err)
	}
	return session.MergeBranchIntoCurrent(sourceBranch)
}

func (s *GitSession) MergeBranchIntoCurrent(sourceBranch string) error {
	if err := runCapturingGit(s.command("merge", "--no-ff", "--no-edit", sourceBranch)); err != nil {
		return fmt.Errorf("failed to merge branch %q into current branch: %w", sourceBranch, err)
	}
	return nil
}

// AbortMerge aborts an in-progress merge.
//
// Git's own output is captured rather than wired to the CLI's streams.
func AbortMerge() error {
	session, err := newGitSession()
	if err != nil {
		return fmt.Errorf("failed to abort merge: %w", err)
	}
	return session.AbortMerge()
}

func (s *GitSession) AbortMerge() error {
	if err := runCapturingGit(s.command("merge", "--abort")); err != nil {
		return fmt.Errorf("failed to abort merge: %w", err)
	}
	return nil
}

// FetchOrigin fetches updates from the remote 'origin' when available.
func FetchOrigin() error {
	session, err := newGitSession()
	if err != nil {
		return err
	}
	return session.FetchOrigin()
}

func (s *GitSession) FetchOrigin() error {
	return fetchOrigin(s)
}

func fetchOrigin(session *GitSession) error {
	if !session.hasOriginRemote() {
		utils.Icon("📁", "Remote 'origin' not found. Skipping fetch.")
		return nil
	}

	spinner := utils.NewSpinner("Fetching updates from origin...")
	spinner.Start()

	cmd := session.command("fetch", "origin", "--prune")
	if _, err := runGit(cmd, false); err != nil {
		spinner.Clear()
		return fmt.Errorf("failed to fetch origin: %w", err)
	}

	spinner.Stop("Fetched updates from origin.")
	return nil
}

// HasUpstream reports whether the given local branch has an upstream configured.
func HasUpstream(branch string) bool {
	session, err := newGitSession()
	return err == nil && session.hasUpstream(branch)
}

func (s *GitSession) hasUpstream(branch string) bool {
	return gitSucceeds(s.command("rev-parse", "--abbrev-ref", "--symbolic-full-name", branch+"@{upstream}"))
}

// CheckoutBranch switches to an existing branch without pulling or merging.
//
// When the branch exists locally, it is checked out directly with no network
// access. When it exists only on origin, it is fetched and checked out as a
// tracking branch. This is the checkout-only counterpart of PullBranch, used
// when branching off a source branch that must not be updated.
//
// The remote is resolved before the fetch, so a lookup that cannot reach origin
// returns its own error instead of reporting the branch as absent.
func CheckoutBranch(branch string) error {
	session, err := newGitSession()
	if err != nil {
		return err
	}
	return checkoutBranch(session, branch)
}

func checkoutBranch(session *GitSession, branch string) error {
	if session.BranchExists(branch) {
		return session.CheckoutExistingBranch(branch)
	}

	if !session.hasOriginRemote() {
		return fmt.Errorf("branch %q does not exist locally and no 'origin' remote is configured", branch)
	}

	// The fetch exists only to materialize origin/<branch> for the tracking
	// checkout, so existence is resolved first: a lookup that cannot check origin
	// must surface as its own error, not be reduced to "absent".
	remoteExisted, err := remoteBranchExists(session, branch)
	if err != nil {
		return err
	}

	if err := fetchOrigin(session); err != nil {
		return err
	}

	if !remoteExisted {
		return fmt.Errorf("branch %q does not exist locally nor on 'origin'", branch)
	}

	return session.checkoutTrackingBranch(branch)
}

// PullBranch checks out the given branch and updates it from origin when possible.
func PullBranch(branch string) error {
	session, err := newGitSession()
	if err != nil {
		return err
	}
	return session.PullBranch(branch)
}

func (s *GitSession) PullBranch(branch string) error {
	if err := checkoutBranch(s, branch); err != nil {
		return err
	}

	if !s.hasOriginRemote() {
		return nil
	}

	if s.hasUpstream(branch) {
		if err := pullWithSession(s, branch, "pull"); err != nil {
			return fmt.Errorf("failed to update branch %q: %w", branch, err)
		}
		return nil
	}

	remoteExisted, err := remoteBranchExists(s, branch)
	if err != nil {
		return err
	}
	if !remoteExisted {
		utils.Info("Remote branch '%s' does not exist. Skipping pull for this target.", branch)
		return nil
	}

	if err := pullWithSession(s, branch, "pull", "origin", branch); err != nil {
		return fmt.Errorf("failed to update branch %q from origin: %w", branch, err)
	}
	return nil
}

func pullWithSession(session *GitSession, branch string, args ...string) error {
	spinner := utils.NewSpinner(fmt.Sprintf("Pulling '%s' from origin...", branch))
	spinner.Start()
	if _, err := runGit(session.command(args...), false); err != nil {
		spinner.Clear()
		return err
	}
	spinner.Stop(fmt.Sprintf("Updated '%s' from origin.", branch))
	return nil
}

// PushBranchUpdate pushes the given branch to origin without changing upstream tracking.
func PushBranchUpdate(branch string) error {
	session, err := newGitSession()
	if err != nil {
		return err
	}
	return session.PushBranchUpdate(branch)
}

func (s *GitSession) PushBranchUpdate(branch string) error {
	if !s.hasOriginRemote() {
		utils.Icon("📁", "Remote 'origin' not found. Skipping push for '%s'.", branch)
		return nil
	}

	spinner := utils.NewSpinner(fmt.Sprintf("Pushing updates for '%s' to origin...", branch))
	spinner.Start()

	cmd := s.command("push", "origin", branch)
	if _, err := runGit(cmd, false); err != nil {
		spinner.Clear()
		return fmt.Errorf("failed to push branch %q to origin: %w", branch, err)
	}

	spinner.Stop(fmt.Sprintf("Pushed updates for '%s'.", branch), "🚀")
	return nil
}

// PushWorkBranch publishes the work branch to origin and sets its upstream.
//
// `dflow finish` publishes the branch before merging because a manual target is
// completed through a pull request and a pull request cannot be opened until the
// branch exists on origin; for an auto target the published branch is the backup
// of the work the merge depends on.
//
// Publishing is idempotent on identity, not on existence: the remote commit for
// the branch is compared with the local one, so a branch origin already holds at
// this exact commit is left untouched and reported as up to date instead of
// claiming a push that did not happen, while a branch origin does not have or
// holds at an older commit is pushed. A repository without an 'origin' remote is
// reported and skipped instead of failing a finish that can still merge its
// local targets.
//
// A comparison that fails because origin is configured but unreachable is only a
// warning: the publish is still attempted, so only a failed push fails the
// finish.
func PushWorkBranch(branch string) error {
	session, err := newGitSession()
	if err != nil {
		return err
	}
	return session.PushWorkBranch(branch)
}

func (s *GitSession) PushWorkBranch(branch string) error {
	if !s.hasOriginRemote() {
		utils.Icon("📁", "Remote 'origin' not found. Skipping push for '%s'.", branch)
		return nil
	}

	// The comparison only decides between an honest no-op and a push, so its own
	// failure is a warning, never a failed finish: the push below is the operation
	// of record and reports its own failure. A lookup that says nothing must not
	// turn the cheaper path into a load-bearing one.
	remoteRevision, lookupErr := remoteBranchRevisionIn(s, branch)
	if lookupErr != nil {
		utils.Warn("Could not compare '%s' with origin (%v); attempting the publish anyway.", branch, lookupErr)
	} else {
		localResult, err := runGit(s.command("rev-parse", "--verify", "refs/heads/"+branch), false)
		if err != nil {
			return fmt.Errorf("failed to resolve local branch '%s': %w", branch, err)
		}

		if local := strings.TrimSpace(localResult.stdout); local == remoteRevision {
			utils.Icon("✔", "Branch '%s' is already published and up to date on origin.", branch)
			return nil
		}
	}

	spinner := utils.NewSpinner(fmt.Sprintf("Publishing '%s' to origin...", branch))
	spinner.Start()

	cmd := s.command("push", "-u", "origin", branch)
	if _, err := runGit(cmd, false); err != nil {
		spinner.Clear()
		return fmt.Errorf("failed to push branch '%s': %w", branch, err)
	}

	spinner.Stop(fmt.Sprintf("Published '%s' to origin.", branch), "🚀")
	return nil
}
