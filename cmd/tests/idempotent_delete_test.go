package tests

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yepizrene-devoost/dflow/cmd/gitutils"
)

// The tests below cover the idempotent-delete work unit (issue #19): the local
// and the remote copy of a branch are independent, so `dflow delete` removes
// whichever copy still exists, names the copy that was already gone, and only
// fails when neither copy exists or when deleting a copy that exists failed.

// TestDeleteIsIdempotentAcrossBothCopies walks the four rows of the issue's
// contract at the gitutils boundary, where the outcome of each half is directly
// observable as branch state.
func TestDeleteIsIdempotentAcrossBothCopies(t *testing.T) {
	cases := []struct {
		name         string
		createLocal  bool
		createRemote bool
		wantErr      bool
	}{
		{name: "both copies exist", createLocal: true, createRemote: true},
		{name: "only the local copy exists", createLocal: true},
		{name: "only the remote copy exists", createRemote: true},
		{name: "neither copy exists", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := initTempGitRepo(t)
			remoteDir := initBareGitRepo(t)
			runGit(t, repo, "remote", "add", "origin", remoteDir)

			if tc.createLocal || tc.createRemote {
				// A remote copy is only reachable by pushing the local one, so the
				// branch is always created first and removed again for the
				// remote-only row.
				runGit(t, repo, "branch", "feature/example")
			}
			if tc.createRemote {
				runGit(t, repo, "push", "origin", "feature/example")
			}
			if tc.createRemote && !tc.createLocal {
				// The half-finished delete the issue reports: the local pointer is
				// gone by hand and only the remote copy is left.
				runGit(t, repo, "branch", "-D", "feature/example")
			}

			withWorkingDir(t, repo, func() {
				err := gitutils.Delete("feature/example")
				if tc.wantErr && err == nil {
					t.Fatalf("Delete must fail when neither copy exists")
				}
				if !tc.wantErr && err != nil {
					t.Fatalf("Delete returned an unexpected error: %v", err)
				}
			})

			if branchExists(t, repo, "feature/example") {
				t.Fatalf("local branch survived the delete")
			}
			if remoteBranchExists(t, repo, "feature/example") {
				t.Fatalf("remote branch survived the delete")
			}
		})
	}
}

// TestDeleteNamesTheCopyThatWasAlreadyGone covers the observable half of the
// same contract through the real binary: a missing copy is reported as absent
// rather than as a failure, and a branch absent from both places fails with
// dflow's own message instead of Git's "branch not found".
func TestDeleteNamesTheCopyThatWasAlreadyGone(t *testing.T) {
	setUpCLIEnv(t)
	binary := buildDflowCLI(t)

	t.Run("local already gone, remote deleted", func(t *testing.T) {
		repo := initTempGitRepo(t)
		remoteDir := initBareGitRepo(t)
		runGit(t, repo, "remote", "add", "origin", remoteDir)
		runGit(t, repo, "branch", "feature/example")
		runGit(t, repo, "push", "origin", "feature/example")
		// Rebuild the state the issue reports: the local pointer is gone by hand and
		// only the remote copy is left to delete.
		runGit(t, repo, "branch", "-D", "feature/example")

		output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "delete", "feature/example", "--yes")
		if exitCode != 0 {
			t.Fatalf("deleting the remote copy alone exited %d, want 0\n%s", exitCode, output)
		}
		if !strings.Contains(output, "does not exist") {
			t.Fatalf("output must say the local copy was already absent:\n%s", output)
		}
		if remoteBranchExists(t, repo, "feature/example") {
			t.Fatalf("remote branch survived the delete")
		}
	})

	t.Run("remote already gone, local deleted", func(t *testing.T) {
		repo := initTempGitRepo(t)
		remoteDir := initBareGitRepo(t)
		runGit(t, repo, "remote", "add", "origin", remoteDir)
		runGit(t, repo, "branch", "feature/example")

		output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "delete", "feature/example", "--yes")
		if exitCode != 0 {
			t.Fatalf("deleting the local copy alone exited %d, want 0\n%s", exitCode, output)
		}
		if !strings.Contains(output, "Remote branch 'feature/example' does not exist") {
			t.Fatalf("output must say the remote copy was already absent:\n%s", output)
		}
		if branchExists(t, repo, "feature/example") {
			t.Fatalf("local branch survived the delete")
		}
	})

	t.Run("neither copy exists", func(t *testing.T) {
		repo := initTempGitRepo(t)
		remoteDir := initBareGitRepo(t)
		runGit(t, repo, "remote", "add", "origin", remoteDir)

		output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "delete", "feature/example", "--yes")
		if exitCode == 0 {
			t.Fatalf("deleting a branch that exists nowhere exited 0, want non-zero\n%s", output)
		}
		if !strings.Contains(output, "nothing to delete") {
			t.Fatalf("failure must say there is nothing to delete:\n%s", output)
		}
	})
}

// TestDeleteDistinguishesAnAbsentRemoteFromAFailedRemoteLookup is the regression
// for the ambiguity issue #23 resolves: a failed `git ls-remote` (instrumented
// here by pointing origin at a path that does not exist, so the failure is
// deterministic and offline) must never be reported as "the remote branch does
// not exist".
//
// The four rows are the four rules of the deletion contract, and each row pins a
// different outcome: the local copy is deleted and an error returned when the
// lookup fails, nothing is touched when there is no local copy either, a
// successful lookup of an absent remote keeps the idle skip and exit 0, and the
// neither-exists case keeps its refusal.
func TestDeleteDistinguishesAnAbsentRemoteFromAFailedRemoteLookup(t *testing.T) {
	cases := []struct {
		name            string
		createLocal     bool
		reachableOrigin bool
		wantErr         bool
		wantContains    string
		wantExcludes    []string
	}{
		{
			name:         "local copy exists, remote lookup fails",
			createLocal:  true,
			wantErr:      true,
			wantContains: "failed to check remote branch 'feature/example'",
			// A lookup failure is not knowledge that the branch is absent.
			wantExcludes: []string{"does not exist"},
		},
		{
			name:         "no local copy, remote lookup fails",
			createLocal:  false,
			wantErr:      true,
			wantContains: "failed to check remote branch 'feature/example'",
			wantExcludes: []string{"nothing to delete"},
		},
		{
			name:            "remote lookup succeeds and the remote copy is absent",
			createLocal:     true,
			reachableOrigin: true,
		},
		{
			name:            "neither copy exists, remote lookup succeeds",
			createLocal:     false,
			reachableOrigin: true,
			wantErr:         true,
			wantContains:    "nothing to delete",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := initTempGitRepo(t)
			originURL := filepath.Join(t.TempDir(), "missing-remote")
			if tc.reachableOrigin {
				originURL = initBareGitRepo(t)
			}
			runGit(t, repo, "remote", "add", "origin", originURL)

			if tc.createLocal {
				runGit(t, repo, "branch", "feature/example")
			}

			// Assertions happen outside the chdir closure: a t.Fatalf inside it would
			// end the test goroutine before the post-call state checks ran.
			var deleteErr error
			withWorkingDir(t, repo, func() {
				deleteErr = gitutils.Delete("feature/example")
			})

			if tc.wantErr && deleteErr == nil {
				t.Fatalf("expected Delete to fail")
			}
			if !tc.wantErr && deleteErr != nil {
				t.Fatalf("unexpected error: %v", deleteErr)
			}
			if deleteErr != nil {
				if tc.wantContains != "" && !strings.Contains(deleteErr.Error(), tc.wantContains) {
					t.Fatalf("error %q does not contain %q", deleteErr.Error(), tc.wantContains)
				}
				for _, excluded := range tc.wantExcludes {
					if strings.Contains(deleteErr.Error(), excluded) {
						t.Fatalf("error %q must not contain %q", deleteErr.Error(), excluded)
					}
				}
			}

			// Every rule ends with the local pointer absent: the first row proves the
			// deletion happened (it existed before), the second proves nothing was
			// touched (it never existed and still does not).
			if branchExists(t, repo, "feature/example") {
				t.Fatalf("local branch survived the delete")
			}
		})
	}
}

// TestDeleteKeepsTheAbsentSkipWhenNoOriginIsConfigured is the end-to-end pair of
// the lookup distinction: a repository with no `origin` has no remote copy by
// definition, so deleting the local one keeps #19's exit 0 and skip message
// instead of failing the whole command. It reuses the same fixture the other
// real-binary delete tests use, with no origin remote added.
func TestDeleteKeepsTheAbsentSkipWhenNoOriginIsConfigured(t *testing.T) {
	setUpCLIEnv(t)
	binary := buildDflowCLI(t)

	repo := initTempGitRepo(t)
	runGit(t, repo, "branch", "feature/example")

	output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "delete", "feature/example", "--yes")
	if exitCode != 0 {
		t.Fatalf("deleting without an origin remote exited %d, want 0\n%s", exitCode, output)
	}
	if !strings.Contains(output, "Remote branch 'feature/example' does not exist. Skipping remote deletion.") {
		t.Fatalf("output must report the remote copy as absent, not as an unchecked failure:\n%s", output)
	}
	if branchExists(t, repo, "feature/example") {
		t.Fatalf("local branch survived the delete")
	}
}

// TestDeleteReportsWhatRemainsWhenTheRemoteStepFails guards the last part of the
// issue's expectation: a remote failure after a successful local deletion is
// reported, names the remote branch that remains, and does not hide that the
// local half is already gone.
//
// The remote refuses deletions through receive.denyDeletes, which makes the
// failure deterministic and local: no network and no timing involved.
func TestDeleteReportsWhatRemainsWhenTheRemoteStepFails(t *testing.T) {
	setUpCLIEnv(t)
	binary := buildDflowCLI(t)

	repo := initTempGitRepo(t)
	remoteDir := initBareGitRepo(t)
	runGit(t, repo, "remote", "add", "origin", remoteDir)
	runGit(t, repo, "branch", "feature/example")
	runGit(t, repo, "push", "origin", "feature/example")
	runGit(t, remoteDir, "config", "receive.denyDeletes", "true")

	output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "delete", "feature/example", "--yes")
	if exitCode == 0 {
		t.Fatalf("a refused remote deletion exited 0, want non-zero\n%s", output)
	}
	if !strings.Contains(output, "deleted local branch 'feature/example'") {
		t.Fatalf("failure must say the local half was deleted, got:\n%s", output)
	}
	if !strings.Contains(output, "failed to delete remote branch 'feature/example'") {
		t.Fatalf("failure must name the remote branch that remains, got:\n%s", output)
	}
	if branchExists(t, repo, "feature/example") {
		t.Fatalf("the local half should have been deleted before the remote failure")
	}
	if !remoteBranchExists(t, repo, "feature/example") {
		t.Fatalf("the remote branch must still exist after the refused deletion")
	}
}
