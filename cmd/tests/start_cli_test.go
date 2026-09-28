package tests

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
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

	binary := buildDflowCLI(t)

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

	t.Run("invalid normalized names do not mutate repository state", func(t *testing.T) {
		tests := []struct {
			name       string
			branchName string
			fullName   string
		}{
			{name: "empty after whitespace normalization", branchName: "   ", fullName: "feature/"},
			{name: "double dot", branchName: "bad..name", fullName: "feature/bad..name"},
			{name: "lock suffix", branchName: "topic.lock", fullName: "feature/topic.lock"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				repo := setupStartRepo(t)
				saveStartCLIConfig(t, repo)
				before := startCLICommand(t, 10*time.Second, repo, "git", "rev-parse", "HEAD")

				output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "start", "feat", tt.branchName, "--no-push")
				if exitCode == 0 {
					t.Fatalf("start with invalid normalized name exited 0, want non-zero\n%s", output)
				}
				if !strings.Contains(output, "invalid branch name") {
					t.Fatalf("invalid-name failure missing validation reason:\n%s", output)
				}
				if branchExists(t, repo, tt.fullName) {
					t.Fatalf("invalid normalized name created %q", tt.fullName)
				}
				if got := startCLICommand(t, 10*time.Second, repo, "git", "branch", "--show-current"); got != "feature/parent" {
					t.Fatalf("current branch = %q, want untouched feature/parent", got)
				}
				if got := startCLICommand(t, 10*time.Second, repo, "git", "rev-parse", "HEAD"); got != before {
					t.Fatalf("HEAD = %s, want untouched %s", got, before)
				}
			})
		}
	})

	// A non-zero exit either leaves the repository as it was, or states
	// explicitly what it created. The base checkout and the pull happen before
	// the new branch exists, so a failure there must restore the caller's branch.
	t.Run("base checkout failure preserves diagnostics and state", func(t *testing.T) {
		repo := setupStartRepo(t)
		saveStartCLIConfig(t, repo)
		before := startCLICommand(t, 10*time.Second, repo, "git", "rev-parse", "HEAD")
		if err := os.WriteFile(filepath.Join(repo, "app.txt"), []byte("dirty local work\n"), 0644); err != nil {
			t.Fatalf("dirty checkout fixture: %v", err)
		}

		output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "start", "feat", "checkout-failure", "--no-push")
		if exitCode == 0 {
			t.Fatalf("start with a blocked base checkout exited 0, want non-zero\n%s", output)
		}
		if !strings.Contains(output, "would be overwritten by checkout") {
			t.Fatalf("checkout failure lost Git's diagnostic:\n%s", output)
		}
		if got := startCLICommand(t, 10*time.Second, repo, "git", "branch", "--show-current"); got != "feature/parent" {
			t.Fatalf("current branch = %q, want original feature/parent", got)
		}
		if branchExists(t, repo, "feature/checkout-failure") {
			t.Fatal("blocked base checkout created the requested branch")
		}
		if got := startCLICommand(t, 10*time.Second, repo, "git", "rev-parse", "HEAD"); got != before {
			t.Fatalf("HEAD = %s, want original %s", got, before)
		}
	})

	t.Run("pull failure restores the original branch", func(t *testing.T) {
		repo := setupStartRepoLocalDevelop(t)
		saveStartCLIConfig(t, repo)
		before := startCLICommand(t, 10*time.Second, repo, "git", "rev-parse", "HEAD")

		output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "start", "feat", "pre-create", "--no-push")
		if exitCode == 0 {
			t.Fatalf("start with a failing pull exited 0, want non-zero\n%s", output)
		}
		if !strings.Contains(output, "failed to pull latest changes from 'develop'") {
			t.Fatalf("failure message must keep the base-pull wording:\n%s", output)
		}
		if !strings.Contains(output, "no tracking information") {
			t.Fatalf("pull failure lost Git's diagnostic:\n%s", output)
		}
		if got := startCLICommand(t, 10*time.Second, repo, "git", "branch", "--show-current"); got != "feature/parent" {
			t.Fatalf("current branch = %q, want the original 'feature/parent'", got)
		}
		if branchExists(t, repo, "feature/pre-create") {
			t.Fatalf("a failed pull must not leave the new branch created")
		}
		if got := startCLICommand(t, 10*time.Second, repo, "git", "rev-parse", "HEAD"); got != before {
			t.Fatalf("HEAD = %s, want the untouched %s", got, before)
		}
	})

	t.Run("branch creation failure restores the original branch", func(t *testing.T) {
		repo := setupStartRepo(t)
		saveStartCLIConfig(t, repo)
		runGit(t, repo, "branch", "feature/existing", "develop")
		before := startCLICommand(t, 10*time.Second, repo, "git", "rev-parse", "HEAD")

		output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "start", "feat", "existing", "--no-push")
		if exitCode == 0 {
			t.Fatalf("start with an existing target exited 0, want non-zero\n%s", output)
		}
		if !strings.Contains(output, "already exists") {
			t.Fatalf("branch creation failure lost Git's diagnostic:\n%s", output)
		}
		if got := startCLICommand(t, 10*time.Second, repo, "git", "branch", "--show-current"); got != "feature/parent" {
			t.Fatalf("current branch = %q, want restored feature/parent", got)
		}
		if got := startCLICommand(t, 10*time.Second, repo, "git", "rev-parse", "HEAD"); got != before {
			t.Fatalf("HEAD = %s, want restored %s", got, before)
		}
	})

	t.Run("restoration failure reports the current branch", func(t *testing.T) {
		probeExecutablePostCheckoutHook(t)

		repo := setupStartRepo(t)
		saveStartCLIConfig(t, repo)
		runGit(t, repo, "branch", "feature/existing", "develop")

		hook := filepath.Join(repo, ".git", "hooks", "post-checkout")
		hookScript := "#!/bin/sh\nif [ \"$(git branch --show-current)\" = develop ]; then\n  printf 'dirty after checkout\\n' > app.txt\nfi\n"
		if err := os.WriteFile(hook, []byte(hookScript), 0755); err != nil {
			t.Fatalf("write post-checkout hook: %v", err)
		}

		output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "start", "feat", "existing", "--no-push")
		if exitCode == 0 {
			t.Fatalf("start with failed creation and restoration exited 0, want non-zero\n%s", output)
		}
		if !strings.Contains(output, "already exists") {
			t.Fatalf("failure lost the original branch-creation diagnostic:\n%s", output)
		}
		if !strings.Contains(output, "Could not restore the original branch 'feature/parent'") ||
			!strings.Contains(output, "caller is now on 'develop'") {
			t.Fatalf("restoration failure did not identify the original and current branches:\n%s", output)
		}
		if got := startCLICommand(t, 10*time.Second, repo, "git", "branch", "--show-current"); got != "develop" {
			t.Fatalf("current branch = %q, want reported develop", got)
		}
	})

	// After the branch is created the failure must not delete it, and it must
	// say so, so a caller that sees a non-zero exit does not retry into
	// "already exists".
	t.Run("push failure names the created branch", func(t *testing.T) {
		repo := setupStartRepo(t)
		saveStartCLIConfig(t, repo)
		// Break only the push: the chained parent exists locally, so --from skips
		// the base checkout and pull and reaches the push directly.
		startCLICommand(t, 10*time.Second, repo, "git", "remote", "set-url", "origin", filepath.Join(t.TempDir(), "missing-remote.git"))

		output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "start", "feat", "post-create", "--from", "feature/parent", "--push")
		if exitCode == 0 {
			t.Fatalf("start with a failing push exited 0, want non-zero\n%s", output)
		}
		if !strings.Contains(output, "was created and remains") {
			t.Fatalf("failure message must state the branch was created and remains:\n%s", output)
		}
		if !strings.Contains(output, "does not appear to be a git repository") {
			t.Fatalf("push failure lost Git's diagnostic:\n%s", output)
		}
		if !strings.Contains(output, "git push -u origin feature/post-create") {
			t.Fatalf("push failure must explain how to publish the retained branch:\n%s", output)
		}
		if !branchExists(t, repo, "feature/post-create") {
			t.Fatalf("the created branch must remain after a failed push")
		}
		if got := startCLICommand(t, 10*time.Second, repo, "git", "branch", "--show-current"); got != "feature/post-create" {
			t.Fatalf("current branch = %q, want 'feature/post-create'", got)
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

type executableHookProbeResult int

const (
	executableHookSupported executableHookProbeResult = iota
	executableHookUnsupported
	executableHookBroken
)

func classifyExecutableHookProbe(hookErr, controlErr error, markerPresent bool, hookOutput string) executableHookProbeResult {
	if hookErr == nil {
		if markerPresent {
			return executableHookSupported
		}
		return executableHookUnsupported
	}
	if controlErr == nil && isUnavailablePostCheckoutHookDiagnostic(hookOutput) {
		return executableHookUnsupported
	}
	return executableHookBroken
}

func isUnavailablePostCheckoutHookDiagnostic(output string) bool {
	diagnostic := strings.ToLower(output)
	if !strings.Contains(diagnostic, "post-checkout") {
		return false
	}
	return strings.Contains(diagnostic, "cannot run") ||
		strings.Contains(diagnostic, "cannot spawn") ||
		strings.Contains(diagnostic, "no such file or directory") ||
		strings.Contains(diagnostic, "not found")
}

func TestClassifyExecutableHookProbe(t *testing.T) {
	hookErr := errors.New("checkout with hook failed")
	controlErr := errors.New("checkout without hook failed")

	tests := []struct {
		name          string
		hookErr       error
		controlErr    error
		markerPresent bool
		diagnostic    string
		want          executableHookProbeResult
	}{
		{name: "supported", markerPresent: true, want: executableHookSupported},
		{name: "hook ignored", want: executableHookUnsupported},
		{name: "hook execution unavailable", hookErr: hookErr, diagnostic: "fatal: cannot run .git/hooks/post-checkout: No such file or directory", want: executableHookUnsupported},
		{name: "unrelated failure despite successful control", hookErr: hookErr, diagnostic: "fatal: cannot lock ref", want: executableHookBroken},
		{name: "unrelated checkout failure", hookErr: hookErr, controlErr: controlErr, diagnostic: "fatal: cannot run .git/hooks/post-checkout: No such file or directory", want: executableHookBroken},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyExecutableHookProbe(tt.hookErr, tt.controlErr, tt.markerPresent, tt.diagnostic)
			if got != tt.want {
				t.Fatalf("classifyExecutableHookProbe() = %v, want %v", got, tt.want)
			}
		})
	}
}

func probeExecutablePostCheckoutHook(t *testing.T) {
	t.Helper()

	repo := initTempGitRepo(t)
	runGit(t, repo, "branch", "hook-probe")

	marker := filepath.Join(t.TempDir(), "post-checkout-ran")
	hook := filepath.Join(repo, ".git", "hooks", "post-checkout")
	hookScript := "#!/bin/sh\nprintf 'executed\\n' > \"$DFLOW_HOOK_PROBE_MARKER\"\n"
	if err := os.WriteFile(hook, []byte(hookScript), 0755); err != nil {
		t.Fatalf("write executable post-checkout capability probe: %v", err)
	}

	checkout := exec.Command("git", "checkout", "hook-probe")
	checkout.Dir = repo
	checkout.Env = append(os.Environ(), "DFLOW_HOOK_PROBE_MARKER="+marker)
	checkoutOutput, checkoutErr := checkout.CombinedOutput()

	markerPresent := false
	if _, err := os.Stat(marker); err == nil {
		markerPresent = true
	} else if !os.IsNotExist(err) {
		t.Fatalf("inspect executable post-checkout capability marker: %v", err)
	}

	var controlErr error
	var controlOutput []byte
	if checkoutErr != nil {
		resetHead := exec.Command("git", "symbolic-ref", "HEAD", "refs/heads/main")
		resetHead.Dir = repo
		resetOutput, resetErr := resetHead.CombinedOutput()
		if resetErr != nil {
			t.Fatalf("post-checkout hook probe failed and its same-branch control could not reset HEAD: %v\n%s\nprobe failure: %v\n%s", resetErr, resetOutput, checkoutErr, checkoutOutput)
		}

		emptyHooksDir := t.TempDir()
		control := exec.Command("git", "-c", "core.hooksPath="+emptyHooksDir, "checkout", "hook-probe")
		control.Dir = repo
		controlOutput, controlErr = control.CombinedOutput()
	}

	switch classifyExecutableHookProbe(checkoutErr, controlErr, markerPresent, string(checkoutOutput)) {
	case executableHookSupported:
		return
	case executableHookUnsupported:
		if checkoutErr != nil {
			t.Skipf("host cannot execute the required POSIX post-checkout hook: %v\n%s", checkoutErr, checkoutOutput)
		}
		t.Skip("host did not execute the required POSIX post-checkout hook")
	case executableHookBroken:
		t.Fatalf("post-checkout hook probe failed for an unrelated Git or host error: %v\n%s\nhook-disabled checkout also failed: %v\n%s", checkoutErr, checkoutOutput, controlErr, controlOutput)
	default:
		t.Fatal("executable post-checkout hook probe returned an inconsistent result")
	}
}

func saveStartCLIConfig(t *testing.T, repo string) {
	t.Helper()
	withWorkingDir(t, repo, func() {
		if err := utils.SaveConfig(startTestConfig()); err != nil {
			t.Fatalf("save CLI fixture config: %v", err)
		}
	})
}

// setupStartRepoLocalDevelop is setupStartRepo without the develop upstream: a
// base branch that cannot be pulled is the failure window the start command must
// repair by restoring the caller's branch.
func setupStartRepoLocalDevelop(t *testing.T) string {
	t.Helper()

	repoDir := initTempGitRepo(t)
	remoteDir := initBareGitRepo(t)

	runGit(t, repoDir, "checkout", "-b", "develop")
	writeFileAndCommit(t, repoDir, "app.txt", "develop base\n", "seed develop")

	runGit(t, repoDir, "remote", "add", "origin", remoteDir)
	// Deliberately do NOT push develop: without an upstream, `git pull` fails.
	runGit(t, repoDir, "checkout", "-b", "feature/parent")

	return repoDir
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
