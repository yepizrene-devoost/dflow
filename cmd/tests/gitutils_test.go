package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/yepizrene-devoost/dflow/cmd/gitutils"
)

func TestCurrentBranchAndWorkingTreeState(t *testing.T) {
	repoDir := initTempGitRepo(t)

	withWorkingDir(t, repoDir, func() {
		branch, err := gitutils.CurrentBranch()
		if err != nil {
			t.Fatalf("CurrentBranch returned error: %v", err)
		}
		if branch != "main" {
			t.Fatalf("expected current branch 'main', got %q", branch)
		}

		clean, err := gitutils.IsWorkingTreeClean()
		if err != nil {
			t.Fatalf("IsWorkingTreeClean returned error: %v", err)
		}
		if !clean {
			t.Fatalf("expected clean working tree")
		}

		filePath := filepath.Join(repoDir, "README.md")
		if err := os.WriteFile(filePath, []byte("modified\n"), 0644); err != nil {
			t.Fatalf("failed to modify file: %v", err)
		}

		clean, err = gitutils.IsWorkingTreeClean()
		if err != nil {
			t.Fatalf("IsWorkingTreeClean returned error after modification: %v", err)
		}
		if clean {
			t.Fatalf("expected dirty working tree")
		}

		if err := gitutils.EnsureWorkingTreeClean(); err == nil {
			t.Fatalf("expected EnsureWorkingTreeClean to fail on dirty tree")
		}
	})
}

func TestMergeInProgressAndAbortMerge(t *testing.T) {
	repoDir := initTempGitRepo(t)

	withWorkingDir(t, repoDir, func() {
		writeFileAndCommit(t, repoDir, "conflict.txt", "base\n", "base commit")

		runGit(t, repoDir, "checkout", "-b", "feature/conflict")
		writeFileAndCommit(t, repoDir, "conflict.txt", "feature\n", "feature change")

		runGit(t, repoDir, "checkout", "main")
		writeFileAndCommit(t, repoDir, "conflict.txt", "main\n", "main change")

		if err := gitutils.MergeBranchIntoCurrent("feature/conflict"); err == nil {
			t.Fatalf("expected merge conflict when merging feature/conflict into main")
		}

		if !gitutils.MergeInProgress() {
			t.Fatalf("expected merge to be in progress after conflict")
		}

		if err := gitutils.AbortMerge(); err != nil {
			t.Fatalf("AbortMerge returned error: %v", err)
		}

		if gitutils.MergeInProgress() {
			t.Fatalf("expected merge state to be cleared after abort")
		}
	})
}

func initTempGitRepo(t *testing.T) string {
	t.Helper()

	repoDir := t.TempDir()
	runGit(t, repoDir, "init", "-b", "main")
	runGit(t, repoDir, "config", "user.name", "Dflow Test")
	runGit(t, repoDir, "config", "user.email", "test@example.com")
	runGit(t, repoDir, "config", "commit.gpgsign", "false")
	writeFileAndCommit(t, repoDir, ".gitkeep", "seed\n", "seed repository")
	return repoDir
}

func writeFileAndCommit(t *testing.T, repoDir, fileName, content, message string) {
	t.Helper()

	filePath := filepath.Join(repoDir, fileName)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file %q: %v", filePath, err)
	}

	runGit(t, repoDir, "add", fileName)
	runGit(t, repoDir, "commit", "-m", message)
}

func runGit(t *testing.T, repoDir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = repoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
	}
}

func withWorkingDir(t *testing.T, dir string, fn func()) {
	t.Helper()

	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change working directory to %q: %v", dir, err)
	}
	defer func() {
		if err := os.Chdir(previousDir); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	}()

	fn()
}
