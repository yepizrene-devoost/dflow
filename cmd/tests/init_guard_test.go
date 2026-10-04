package tests

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yepizrene-devoost/dflow/pkg/validators"
)

// initGuardConfig is a hand-written .dflow.yaml with deliberately recognizable
// branch names, flow rules, merge modes and an extra key. It is written directly
// to disk (never through SaveConfig) so the bytes are fully controlled and a
// regeneration is impossible to miss.
const initGuardConfig = `# hand-edited team contract; must survive a blocked dflow init
branches:
  main: trunk
  develop: integration
  uat: staging
  features: "feat/"
  releases: "rel/"
  hotfixes: "hot/"
  bugfixes: "bug/"
flow:
  feature:
    base: integration
    finish_targets:
      - integration
      - staging
workflow:
  default_merge_mode: manual
  branch_rules:
    trunk:
      merge_mode: manual
    integration:
      merge_mode: auto
custom_team_key: do-not-touch
`

// initRefusalMessage is the exact refusal sentence `dflow init` must emit when
// the project already has a `.dflow.yaml`. Both the CLI and validator assertions
// reference it so they cannot drift apart.
const initRefusalMessage = "this project is already initialized with .dflow.yaml; use --force to regenerate it"

// newInitializedRepo creates a repository whose .dflow.yaml was written by hand
// and returns the repo path together with those exact bytes.
func newInitializedRepo(t *testing.T) (string, []byte) {
	t.Helper()

	repo := initTempGitRepo(t)
	path := filepath.Join(repo, ".dflow.yaml")
	if err := os.WriteFile(path, []byte(initGuardConfig), 0644); err != nil {
		t.Fatalf("failed to write hand-written .dflow.yaml: %v", err)
	}

	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read back hand-written .dflow.yaml: %v", err)
	}
	return repo, original
}

// TestInitRefusesToReinitialize guards the core refusal: in a repo that already
// has a .dflow.yaml, `dflow init` must fail with the specific --force message and
// must not touch the file, even though the command is non-interactive.
func TestInitRefusesToReinitialize(t *testing.T) {
	setUpCLIEnv(t)
	binary := buildDflowCLI(t)

	repo, original := newInitializedRepo(t)

	output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "init")
	if exitCode == 0 {
		t.Fatalf("init in an initialized repo exited 0, want non-zero\n%s", output)
	}
	for _, want := range []string{"already initialized", ".dflow.yaml", "--force"} {
		if !strings.Contains(output, want) {
			t.Fatalf("refusal output is missing %q:\n%s", want, output)
		}
	}
	if !strings.Contains(output, initRefusalMessage) {
		t.Fatalf("refusal output is missing the exact sentence %q:\n%s", initRefusalMessage, output)
	}
	if got := strings.Count(output, "\u274c"); got != 1 {
		t.Fatalf("refusal must render exactly one cross icon, got %d:\n%s", got, output)
	}

	assertConfigUnchanged(t, repo, original)
}

// TestInitForceStillRequiresTerminal proves --force only bypasses the guard. The
// command then reaches the terminal check, so a non-TTY caller fails without
// ever writing the file.
func TestInitForceStillRequiresTerminal(t *testing.T) {
	setUpCLIEnv(t)
	binary := buildDflowCLI(t)

	repo, original := newInitializedRepo(t)

	output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "init", "--force")
	if exitCode == 0 {
		t.Fatalf("init --force without a terminal exited 0, want non-zero\n%s", output)
	}
	if !strings.Contains(output, "interactive") {
		t.Fatalf("--force must reach the terminal check, got:\n%s", output)
	}

	assertConfigUnchanged(t, repo, original)
}

// TestInitFirstRunStillReachesPrompt keeps the legitimate first run working: with
// no .dflow.yaml the guard must not block, and the command still stops at the
// terminal check instead of failing for some other reason.
func TestInitFirstRunStillReachesPrompt(t *testing.T) {
	setUpCLIEnv(t)
	binary := buildDflowCLI(t)

	repo := initTempGitRepo(t)

	output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, "init")
	if exitCode == 0 {
		t.Fatalf("init without a terminal exited 0, want non-zero\n%s", output)
	}
	if !strings.Contains(output, "interactive") {
		t.Fatalf("first run must reach the terminal check, got:\n%s", output)
	}
}

// initNoCommitsCLIWording and initNoOriginSkipWording are the exact sentences the
// real binary must render for the two issue #60 defects. They repeat the CLI's
// own wording on purpose, in the same style as initRefusalMessage above: a test
// that reaches into the command package could not pin what the user reads.
const (
	initNoCommitsCLIWording = "this repository has no commits yet; create the first commit, then rerun `dflow init`"
	initNoOriginSkipWording = "Remote 'origin' not found. Skipping base branch publication."
)

// TestInitFailsFastOnRepositoryWithoutCommits pins the WU1 preflight on the real
// binary. A repository with no commits cannot have base branches created at all,
// so init must refuse before the terminal check — which is what lets a plain pipe
// reach the refusal — and before any question or mutation. --force authorizes
// regenerating .dflow.yaml only, so it must reach the same refusal.
func TestInitFailsFastOnRepositoryWithoutCommits(t *testing.T) {
	setUpCLIEnv(t)
	binary := buildDflowCLI(t)

	repo := initTempEmptyGitRepo(t)

	for _, args := range [][]string{{"init"}, {"init", "--force"}} {
		output, exitCode := startCLIRawOutput(t, 15*time.Second, repo, binary, args...)
		if exitCode == 0 {
			t.Fatalf("%v in a commitless repository exited 0, want non-zero\n%s", args, output)
		}
		if !strings.Contains(output, initNoCommitsCLIWording) {
			t.Fatalf("%v is missing the exact refusal %q:\n%s", args, initNoCommitsCLIWording, output)
		}
		if strings.Contains(output, "interactive") {
			t.Fatalf("%v must fail at the preflight before the terminal check:\n%s", args, output)
		}
		requireFileAbsent(t, filepath.Join(repo, ".dflow.yaml"))
	}
}

// TestInitCompletesOnboardingWithoutOriginRemote pins WU2 end to end: the
// repository has a commit but no remote, which is exactly the local-only project
// whose onboarding used to abort on `git ls-remote --heads origin`. The run must
// complete, publish nothing, and report the skip exactly once.
func TestInitCompletesOnboardingWithoutOriginRemote(t *testing.T) {
	setUpCLIEnv(t)
	binary := buildDflowCLI(t)

	repo := initTempGitRepo(t)

	output, exitCode := runInteractiveCLI(t, 60*time.Second, repo, binary, initTerminalAnswers, "init")
	if exitCode != 0 {
		t.Fatalf("init in a remote-less repository exited %d, want 0\n%s", exitCode, output)
	}
	if !strings.Contains(output, "Created .dflow.yaml") {
		t.Fatalf("onboarding did not complete:\n%s", output)
	}
	if got := strings.Count(output, initNoOriginSkipWording); got != 1 {
		t.Fatalf("skip report rendered %d times, want exactly one line %q:\n%s", got, initNoOriginSkipWording, output)
	}
	if strings.Contains(output, "does not appear to be a git repository") {
		t.Fatalf("init reached an unguarded remote operation:\n%s", output)
	}

	if _, err := os.Stat(filepath.Join(repo, ".dflow.yaml")); err != nil {
		t.Fatalf(".dflow.yaml was not written: %v", err)
	}
	if got := runGitOutput(t, repo, "remote", "-v"); got != "" {
		t.Fatalf("repository gained a remote during onboarding: %q", got)
	}
}

// initTerminalAnswersWithPublication is initTerminalAnswers plus the answer to
// the publication question: the driver has to know about that question only when
// the command renders it, which is exactly what the test below pins.
var initTerminalAnswersWithPublication = []terminalAnswer{
	{trigger: "Main branch name:", reply: "\n"},
	{trigger: "Development branch name:", reply: "\n"},
	{trigger: "UAT branch name:", reply: "\n"},
	{trigger: "How do you manage merges by default in this project?", reply: "\n"},
	{trigger: "Which branches should behave differently", reply: "\n"},
	{trigger: "Do you want to push the base branches to 'origin'?", reply: "n\n"},
	{trigger: "Generate an agent workflow file", reply: "\n"},
}

// TestInitOffersPublicationPromptWhenOriginExists guards the other half of WU2: a
// repository that can be published still gets asked. Answering "n" keeps the test
// off the remote, and the bare origin is asserted empty so the refusal is real.
func TestInitOffersPublicationPromptWhenOriginExists(t *testing.T) {
	setUpCLIEnv(t)
	binary := buildDflowCLI(t)

	repo := initTempGitRepo(t)
	origin := initBareGitRepo(t)
	runGit(t, repo, "remote", "add", "origin", origin)

	output, exitCode := runInteractiveCLI(t, 60*time.Second, repo, binary, initTerminalAnswersWithPublication, "init")
	if exitCode != 0 {
		t.Fatalf("init with an origin remote exited %d, want 0\n%s", exitCode, output)
	}
	if !strings.Contains(output, "Do you want to push the base branches to 'origin'?") {
		t.Fatalf("the publication question must still be offered when origin exists:\n%s", output)
	}
	if strings.Contains(output, initNoOriginSkipWording) {
		t.Fatalf("a repository with origin must not report the skip:\n%s", output)
	}

	if refs := strings.TrimSpace(runGitOutput(t, origin, "for-each-ref", "--format=%(refname)")); refs != "" {
		t.Fatalf("init published to origin without asking for it: %q", refs)
	}
}

// TestEnsureDflowNotInitializedUnit covers the validator in isolation with the
// repo's manual os.Chdir pattern (withWorkingDir), because Go 1.21 has no
// t.Chdir.
func TestEnsureDflowNotInitializedUnit(t *testing.T) {
	empty := initTempGitRepo(t)
	withWorkingDir(t, empty, func() {
		if err := validators.EnsureDflowNotInitialized(); err != nil {
			t.Fatalf("EnsureDflowNotInitialized() in an empty dir = %v, want nil", err)
		}
	})

	initialized := initTempGitRepo(t)
	if err := os.WriteFile(filepath.Join(initialized, ".dflow.yaml"), []byte("branches: {}\n"), 0644); err != nil {
		t.Fatalf("failed to write .dflow.yaml: %v", err)
	}
	withWorkingDir(t, initialized, func() {
		err := validators.EnsureDflowNotInitialized()
		if err == nil {
			t.Fatalf("EnsureDflowNotInitialized() with a .dflow.yaml = nil, want an error")
		}
		if err.Error() != initRefusalMessage {
			t.Fatalf("error = %q, want the exact sentence %q", err.Error(), initRefusalMessage)
		}
	})
}

// assertConfigUnchanged fails unless .dflow.yaml is byte-for-byte identical to
// the original hand-written contract.
func assertConfigUnchanged(t *testing.T, repo string, original []byte) {
	t.Helper()

	after, err := os.ReadFile(filepath.Join(repo, ".dflow.yaml"))
	if err != nil {
		t.Fatalf("failed to read .dflow.yaml after init: %v", err)
	}
	if !bytes.Equal(after, original) {
		t.Fatalf(".dflow.yaml changed:\n--- before ---\n%s\n--- after ---\n%s", original, after)
	}
}
