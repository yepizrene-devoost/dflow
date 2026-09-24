package tests

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/yepizrene-devoost/dflow/cmd/gitutils"
)

// TestRemoteBranchExistsSeparatesAbsentFromFailedLookup pins the contract the
// idempotent-delete residuals resolve: the bool answers "does the remote branch
// exist" and the error answers "could that be determined". A failed lookup is
// never `false, nil`.
//
// origin points at a path that does not exist, so git ls-remote fails
// deterministically and offline: no network, no mock, no timing.
func TestRemoteBranchExistsSeparatesAbsentFromFailedLookup(t *testing.T) {
	repoDir := initTempGitRepo(t)
	runGit(t, repoDir, "remote", "add", "origin", filepath.Join(t.TempDir(), "missing-remote"))

	withWorkingDir(t, repoDir, func() {
		exists, err := gitutils.RemoteBranchExists("feature/example")
		if err == nil {
			t.Fatalf("a failed lookup must return an error, got exists=%v, err=nil", exists)
		}
		if exists {
			t.Fatalf("a failed lookup must not report the branch as existing")
		}
		if !strings.Contains(err.Error(), "failed to check remote branch") {
			t.Fatalf("the error must identify the lookup, got: %v", err)
		}
	})
}

// TestRemoteBranchExistsKeepsAMissingOriginAndAFailedOriginApart pins the
// distinction the deletion contract rests on, and is the case that separates the
// two states the brief originally conflated.
//
// No `origin` remote is a KNOWN absence: no remote copy of any branch can exist,
// it is determinable locally without a network call, and it is `false, nil`. An
// `origin` that is configured but cannot be reached is the UNKNOWN case, and only
// that one may return an error. A test that lumps them together cannot tell which
// behaviour it is pinning.
func TestRemoteBranchExistsKeepsAMissingOriginAndAFailedOriginApart(t *testing.T) {
	withoutOrigin := initTempGitRepo(t)
	withWorkingDir(t, withoutOrigin, func() {
		exists, err := gitutils.RemoteBranchExists("feature/example")
		if err != nil {
			t.Fatalf("a missing origin is a known absence, not a lookup failure: %v", err)
		}
		if exists {
			t.Fatalf("a repository without origin cannot have a remote branch")
		}
	})

	unreachable := initTempGitRepo(t)
	runGit(t, unreachable, "remote", "add", "origin", filepath.Join(t.TempDir(), "missing-remote"))
	withWorkingDir(t, unreachable, func() {
		exists, err := gitutils.RemoteBranchExists("feature/example")
		if err == nil {
			t.Fatalf("a configured but unreachable origin is a lookup failure, got exists=%v, err=nil", exists)
		}
		if exists {
			t.Fatalf("a failed lookup must not report the branch as existing")
		}

		// Git already explains the failure, so its text must carry the end of the
		// sentence. Appending the exit status after it ("...exists.: exit status
		// 128") restates nothing and reads as a truncation, which is why the existing
		// delete and merge messages never do it.
		if !strings.Contains(err.Error(), "does not appear to be a git repository") {
			t.Fatalf("the error must carry git's own explanation, got: %v", err)
		}
		if strings.Contains(err.Error(), "exit status") {
			t.Fatalf("the error must not append the exit status after git's explanation, got: %v", err)
		}
	})
}

// TestRemoteBranchExistsReportsAbsentAndPresentOnAReachableOrigin is the control
// that keeps the failure test honest: on a reachable origin the two states are
// told apart, so a lookup that always errored (or never did) cannot pass both.
func TestRemoteBranchExistsReportsAbsentAndPresentOnAReachableOrigin(t *testing.T) {
	repoDir := initTempGitRepo(t)
	remoteDir := initBareGitRepo(t)
	runGit(t, repoDir, "remote", "add", "origin", remoteDir)
	runGit(t, repoDir, "branch", "feature/present")
	runGit(t, repoDir, "push", "origin", "feature/present")

	withWorkingDir(t, repoDir, func() {
		exists, err := gitutils.RemoteBranchExists("feature/present")
		if err != nil {
			t.Fatalf("looking up a present branch must not fail: %v", err)
		}
		if !exists {
			t.Fatalf("expected the present branch to be reported as existing")
		}

		exists, err = gitutils.RemoteBranchExists("feature/absent")
		if err != nil {
			t.Fatalf("an absent branch is a successful lookup, not a failure: %v", err)
		}
		if exists {
			t.Fatalf("an absent branch must not be reported as existing")
		}
	})
}

// TestRemoteBranchExistsDoesNotMatchALongerBranchName pins the exact-ref-name
// premise the publish identity comparison in PushWorkBranch rests on: a lookup
// that prefix-matched would report `feature/demo` as existing when origin only
// holds `feature/demo/sub`, letting a finish claim a branch it never published.
func TestRemoteBranchExistsDoesNotMatchALongerBranchName(t *testing.T) {
	repoDir := initTempGitRepo(t)
	remoteDir := initBareGitRepo(t)
	runGit(t, repoDir, "remote", "add", "origin", remoteDir)
	runGit(t, repoDir, "branch", "feature/demo/sub")
	runGit(t, repoDir, "push", "origin", "feature/demo/sub")

	withWorkingDir(t, repoDir, func() {
		exists, err := gitutils.RemoteBranchExists("feature/demo")
		if err != nil {
			t.Fatalf("looking up an absent shorter name must not fail: %v", err)
		}
		if exists {
			t.Fatalf("expected feature/demo to be absent while origin only holds feature/demo/sub")
		}

		exists, err = gitutils.RemoteBranchExists("feature/demo/sub")
		if err != nil {
			t.Fatalf("looking up the pushed branch must not fail: %v", err)
		}
		if !exists {
			t.Fatalf("expected feature/demo/sub to be reported as existing")
		}
	})
}
