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
