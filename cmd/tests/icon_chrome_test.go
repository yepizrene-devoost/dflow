package tests

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	errorIcon   = "\u274c"       // ❌
	successIcon = "\u2705"       // ✅
	deleteIcon  = "\U0001F5D1"   // 🗑️
	infoIcon    = "\u2139\ufe0f" // ℹ️
)

// failureWords are the words a status line uses to describe an operation that
// did not happen. A line that carries one of them must not also carry a success
// icon: the icon is chrome, so a failure path either clears silently and lets
// the single ❌ render speak, or it renders with its own failure icon.
var failureWords = []string{"failed", "failure", "fail", "error", "not found"}

// TestFailurePathRendersOneErrorIconAndNoSuccessChrome guards the WU5 contract
// against the real binary: a failing Git operation renders exactly one ❌ line,
// no line pairs the success icon with a failure word, and the process exits
// non-zero. The companion success case proves the fix cleared failure chrome
// without silencing the success icons.
func TestFailurePathRendersOneErrorIconAndNoSuccessChrome(t *testing.T) {
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_TERMINAL_PROMPT", "0")

	binary := buildCLIBinaryForIconChromeTest(t)

	t.Run("failing delete", func(t *testing.T) {
		repo := initTempGitRepo(t)
		runGit(t, repo, "checkout", "-b", "feature/untouched")

		output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "delete", "no-such-branch-xyz", "--yes")
		if exitCode == 0 {
			t.Fatalf("deleting a missing branch exited 0, want a non-zero exit code\n%s", output)
		}

		if got := strings.Count(output, errorIcon); got != 1 {
			t.Fatalf("failure rendered the error icon %d times, want exactly 1:\n%s", got, output)
		}

		for _, line := range strings.Split(output, "\n") {
			if !strings.Contains(line, errorIcon) {
				continue
			}
			if got := strings.Count(line, errorIcon); got != 1 {
				t.Fatalf("the failure line carries %d error icons, want 1: %q", got, line)
			}
		}

		for _, line := range strings.Split(output, "\n") {
			if strings.Contains(line, successIcon) && hasFailureWord(line) {
				t.Fatalf("line pairs the success icon with a failure word: %q\n%s", line, output)
			}
		}

		if !branchExists(t, repo, "feature/untouched") {
			t.Fatalf("unrelated local branch must survive a refused delete")
		}
	})

	t.Run("successful delete keeps its chrome", func(t *testing.T) {
		repo := initTempGitRepo(t)
		runGit(t, repo, "checkout", "-b", "feature/doomed")
		runGit(t, repo, "checkout", "main")

		output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "delete", "feature/doomed", "--yes")
		if exitCode != 0 {
			t.Fatalf("delete --yes exited %d, want 0\n%s", exitCode, output)
		}
		if strings.Contains(output, errorIcon) {
			t.Fatalf("successful delete rendered an error icon:\n%s", output)
		}
		if !strings.Contains(output, deleteIcon) {
			t.Fatalf("successful delete lost its success chrome:\n%s", output)
		}
		if !strings.Contains(output, "deleted locally") {
			t.Fatalf("successful delete lost its status text:\n%s", output)
		}
	})
}

// TestShortValueArgumentRendersAsValue guards the icon-argument contract against
// the real binary: a one-rune value in the last argument position is a format
// value, not an icon, so it must render instead of being swallowed.
//
// Before the shared chrome stopped sniffing the last argument, a short string
// was consumed as a custom icon and removed from the format arguments, so this
// exact path rendered `%!s(MISSING)`. The defect is only observable end to end,
// through a message whose value the command layer passes unchanged.
func TestShortValueArgumentRendersAsValue(t *testing.T) {
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_TERMINAL_PROMPT", "0")

	binary := buildCLIBinaryForIconChromeTest(t)

	repo := initTempGitRepo(t)
	// `x` is the whole point: one rune is what the removed heuristic mistook
	// for an icon, and it is deleted from a branch, so it is not the current
	// branch.
	runGit(t, repo, "branch", "x")

	output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "delete", "x", "--yes")
	if exitCode != 0 {
		t.Fatalf("delete x --yes exited %d, want 0\n%s", exitCode, output)
	}

	const line = "Remote branch 'x' does not exist. Skipping remote deletion."
	if !strings.Contains(output, line) {
		t.Fatalf("the one-rune value did not render in its message; want %q in:\n%s", line, output)
	}
	if !strings.Contains(output, infoIcon) {
		t.Fatalf("the level default icon is missing; want %s in:\n%s", infoIcon, output)
	}
	if strings.Contains(output, "%!") {
		t.Fatalf("output carries an unrendered format verb:\n%s", output)
	}
}

// buildCLIBinaryForIconChromeTest builds the real entry point outside the
// repository so the assertions observe the shipped binary and its exit code.
func buildCLIBinaryForIconChromeTest(t *testing.T) string {
	t.Helper()

	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	name := "dflow"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binary := filepath.Join(t.TempDir(), name)
	startCLICommand(t, 2*time.Minute, root, "go", "build", "-o", binary, ".")

	return binary
}

// hasFailureWord reports whether a rendered line describes a failed operation.
func hasFailureWord(line string) bool {
	lower := strings.ToLower(line)
	for _, word := range failureWords {
		if strings.Contains(lower, word) {
			return true
		}
	}
	return false
}
