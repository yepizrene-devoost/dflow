package tests

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/yepizrene-devoost/dflow/cmd/utils"
)

// Run the real entry point so Cobra parsing and non-TTY behavior are covered.
func TestStartCLI(t *testing.T) {
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_TERMINAL_PROMPT", "0")

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

	for _, flag := range []string{"--push", "--no-push"} {
		t.Run(flag, func(t *testing.T) {
			repo := setupStartRepo(t)
			saveStartCLIConfig(t, repo)
			base := startCLICommand(t, 10*time.Second, repo, "git", "rev-parse", "develop")
			startCLICommand(t, 15*time.Second, repo, binary, "start", "feat", "cli-child", flag)
			assertStartCLIHead(t, repo, "feature/cli-child", base)

			// Query the actual origin, not the local remote-tracking refs.
			remote := startCLICommand(t, 10*time.Second, repo, "git", "ls-remote", "--heads", "origin", "refs/heads/feature/cli-child")
			if flag == "--push" {
				want := base + "\trefs/heads/feature/cli-child"
				if remote != want {
					t.Fatalf("remote branch = %q, want %q", remote, want)
				}
			} else if remote != "" {
				t.Fatalf("--no-push published a branch: %s", remote)
			}
		})
	}

	// The exit code is the contract a non-TTY caller observes through $?.
	// Assert it against the real binary, and assert the styled failure line is
	// rendered exactly once.
	t.Run("exit codes", func(t *testing.T) {
		repo := setupStartRepo(t)
		saveStartCLIConfig(t, repo)

		output, exitCode := startCLIExitCode(t, 15*time.Second, repo, binary, "start", "feat", "conflict", "--push", "--no-push")
		if exitCode == 0 {
			t.Fatalf("conflicting push flags exited %d, want non-zero\n%s", exitCode, output)
		}
		if !strings.Contains(output, "cannot be used together") {
			t.Fatalf("failure message missing from output:\n%s", output)
		}
		if got := strings.Count(output, "\u274c"); got != 1 {
			t.Fatalf("error line rendered %d times, want exactly 1:\n%s", got, output)
		}

		output, exitCode = startCLIExitCode(t, 15*time.Second, repo, binary, "start", "feat", "exit-ok", "--no-push")
		if exitCode != 0 {
			t.Fatalf("successful start exited %d, want 0\n%s", exitCode, output)
		}
	})

	// A missing .dflow.yaml must fail through the validator wrapper, not only
	// through a command handler returning its own error.
	t.Run("uninitialized repo", func(t *testing.T) {
		repo := initTempGitRepo(t)
		output, exitCode := startCLIExitCode(t, 15*time.Second, repo, binary, "start", "feat", "no-config", "--no-push")
		if exitCode == 0 {
			t.Fatalf("start without .dflow.yaml exited 0, want non-zero\n%s", output)
		}
		if !strings.Contains(output, "not initialized") {
			t.Fatalf("missing initialization error:\n%s", output)
		}
		if got := strings.Count(output, "\u274c"); got != 1 {
			t.Fatalf("error line rendered %d times, want exactly 1:\n%s", got, output)
		}
	})

	t.Run("remote-only parent", func(t *testing.T) {
		publisher := setupStartRepo(t)
		origin := startCLICommand(t, 10*time.Second, publisher, "git", "remote", "get-url", "origin")
		// Clone BEFORE publishing the distinct parent commit.
		// The CLI must discover and fetch the parent itself.
		repo := filepath.Join(t.TempDir(), "consumer")
		startCLICommand(t, 10*time.Second, publisher, "git", "clone", "--branch", "develop", origin, repo)
		saveStartCLIConfig(t, repo)
		parent := startCLICommand(t, 10*time.Second, publisher, "git", "rev-parse", "feature/parent")
		base := startCLICommand(t, 10*time.Second, repo, "git", "rev-parse", "HEAD")
		if parent == base {
			t.Fatal("fixture parent must differ from develop")
		}
		startCLICommand(t, 10*time.Second, publisher, "git", "push", "origin", "feature/parent")
		refs := startCLICommand(t, 10*time.Second, repo, "git", "for-each-ref", "--format=%(refname)", "refs/heads/feature/parent", "refs/remotes/origin/feature/parent")
		if refs != "" {
			t.Fatalf("parent already exists locally: %s", refs)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		probe := exec.CommandContext(ctx, "git", "cat-file", "-e", parent+"^{commit}")
		probe.Dir = repo
		if output, err := probe.CombinedOutput(); err == nil {
			t.Fatalf("parent commit already available locally: %s", output)
		} else if ctx.Err() != nil {
			t.Fatalf("parent object probe timed out: %v", ctx.Err())
		} else if _, ok := err.(*exec.ExitError); !ok {
			t.Fatalf("could not probe parent object: %v", err)
		}

		startCLICommand(t, 15*time.Second, repo, binary, "start", "feat", "remote-child", "--from", "feature/parent", "--no-push")
		assertStartCLIHead(t, repo, "feature/remote-child", parent)
		startCLICommand(t, 10*time.Second, repo, "git", "merge-base", "--is-ancestor", parent, "HEAD")
		remote := startCLICommand(t, 10*time.Second, repo, "git", "ls-remote", "--heads", "origin", "refs/heads/feature/remote-child")
		if remote != "" {
			t.Fatalf("remote child unexpectedly published: %s", remote)
		}
	})

	// Non-interactive contract: with captured stdout and no stdin, output must
	// stay free of carriage returns, ANSI escapes, banner art and repeated
	// spinner frames, and prompts must fail fast naming the concrete remedy.
	t.Run("non-interactive output", func(t *testing.T) {
		repo := setupStartRepo(t)
		saveStartCLIConfig(t, repo)

		output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "start", "feat", "clean-output", "--no-push")
		if exitCode != 0 {
			t.Fatalf("start --no-push exited %d, want 0\n%s", exitCode, output)
		}
		if strings.ContainsRune(output, '\r') {
			t.Fatalf("captured output contains a carriage return:\n%q", output)
		}
		if strings.Contains(output, "\x1b[") {
			t.Fatalf("captured output contains an ANSI escape:\n%q", output)
		}
		if strings.Contains(output, "Git branching made simple") {
			t.Fatalf("captured output contains the banner:\n%s", output)
		}
		if got := strings.Count(output, "Pulling latest changes from origin..."); got > 1 {
			t.Fatalf("pull status rendered %d times, want at most 1:\n%s", got, output)
		}
	})

	t.Run("start without push flags", func(t *testing.T) {
		repo := setupStartRepo(t)
		saveStartCLIConfig(t, repo)
		before := startCLICommand(t, 10*time.Second, repo, "git", "rev-parse", "HEAD")

		output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "start", "feat", "no-flags")
		if exitCode == 0 {
			t.Fatalf("start without --push/--no-push exited 0, want non-zero\n%s", output)
		}
		if !strings.Contains(output, "--push") || !strings.Contains(output, "--no-push") {
			t.Fatalf("failure message must name --push and --no-push:\n%s", output)
		}

		// The precondition must run before the base checkout, the pull and
		// CheckoutNew: a refused invocation leaves the repository untouched, so
		// a retry cannot fail with "branch already exists".
		if branchExists(t, repo, "feature/no-flags") {
			t.Fatalf("start without --push/--no-push created the branch before failing")
		}
		if _, verify := startCLIExitCode(t, 10*time.Second, repo, "git", "rev-parse", "--verify", "refs/heads/feature/no-flags"); verify == 0 {
			t.Fatalf("git rev-parse --verify resolved feature/no-flags after a refused start")
		}
		if got := startCLICommand(t, 10*time.Second, repo, "git", "branch", "--show-current"); got != "feature/parent" {
			t.Fatalf("current branch = %q, want the untouched fixture branch 'feature/parent'", got)
		}
		if got := startCLICommand(t, 10*time.Second, repo, "git", "rev-parse", "HEAD"); got != before {
			t.Fatalf("HEAD = %s, want the untouched %s", got, before)
		}
	})

	t.Run("delete without --yes", func(t *testing.T) {
		repo := initTempGitRepo(t)
		runGit(t, repo, "checkout", "-b", "feature/to-delete")

		output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "delete", "feature/to-delete")
		if exitCode == 0 {
			t.Fatalf("delete without --yes exited 0, want non-zero\n%s", output)
		}
		if !strings.Contains(output, "--yes") {
			t.Fatalf("failure message must name --yes:\n%s", output)
		}
		if !branchExists(t, repo, "feature/to-delete") {
			t.Fatalf("branch must still exist after a refused delete")
		}
	})

	t.Run("delete --yes", func(t *testing.T) {
		repo := initTempGitRepo(t)
		runGit(t, repo, "checkout", "-b", "feature/doomed")
		runGit(t, repo, "checkout", "main")

		output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "delete", "feature/doomed", "--yes")
		if exitCode != 0 {
			t.Fatalf("delete --yes exited %d, want 0\n%s", exitCode, output)
		}
		if branchExists(t, repo, "feature/doomed") {
			t.Fatalf("branch must be gone after delete --yes")
		}
	})

	t.Run("init requires a terminal", func(t *testing.T) {
		repo := initTempGitRepo(t)

		output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "init")
		if exitCode == 0 {
			t.Fatalf("init without a terminal exited 0, want non-zero\n%s", output)
		}
		if !strings.Contains(output, "interactive") {
			t.Fatalf("init failure message must say it needs a terminal:\n%s", output)
		}
	})
}

func saveStartCLIConfig(t *testing.T, repo string) {
	t.Helper()
	withWorkingDir(t, repo, func() {
		if err := utils.SaveConfig(startTestConfig()); err != nil {
			t.Fatalf("save CLI fixture config: %v", err)
		}
	})
}

func assertStartCLIHead(t *testing.T, repo, branch, tip string) {
	t.Helper()
	if got := startCLICommand(t, 10*time.Second, repo, "git", "branch", "--show-current"); got != branch {
		t.Fatalf("current branch = %q, want %q", got, branch)
	}
	if got := startCLICommand(t, 10*time.Second, repo, "git", "rev-parse", "HEAD"); got != tip {
		t.Fatalf("HEAD = %s, want %s", got, tip)
	}
}

func startCLICommand(t *testing.T, timeout time.Duration, dir, program string, args ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, program, args...)
	cmd.Dir = dir
	// A nil Stdin reads from the null device: EOF, never an interactive TTY.
	cmd.Stdin = nil
	cmd.WaitDelay = time.Second
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("%s %v timed out: %v\n%s", program, args, ctx.Err(), output)
	}
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", program, args, err, output)
	}
	return strings.TrimSpace(string(output))
}

// startCLIRawOutput mirrors startCLIExitCode but returns the captured bytes
// unchanged so tests can assert on carriage returns and ANSI escapes that
// strings.TrimSpace would otherwise strip from the edges.
func startCLIRawOutput(t *testing.T, timeout time.Duration, dir, program string, args ...string) (string, int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, program, args...)
	cmd.Dir = dir
	// A nil Stdin reads from the null device: EOF, never an interactive TTY.
	cmd.Stdin = nil
	cmd.WaitDelay = time.Second
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("%s %v timed out: %v\n%s", program, args, ctx.Err(), output)
	}
	if err == nil {
		return string(output), 0
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("%s %v did not run: %v\n%s", program, args, err, output)
	}
	return string(output), exitErr.ExitCode()
}

// startCLIExitCode mirrors startCLICommand but returns the process exit code
// instead of failing the test on a non-zero result.
func startCLIExitCode(t *testing.T, timeout time.Duration, dir, program string, args ...string) (string, int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, program, args...)
	cmd.Dir = dir
	// A nil Stdin reads from the null device: EOF, never an interactive TTY.
	cmd.Stdin = nil
	cmd.WaitDelay = time.Second
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("%s %v timed out: %v\n%s", program, args, ctx.Err(), output)
	}
	if err == nil {
		return strings.TrimSpace(string(output)), 0
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("%s %v did not run: %v\n%s", program, args, err, output)
	}
	return strings.TrimSpace(string(output)), exitErr.ExitCode()
}
