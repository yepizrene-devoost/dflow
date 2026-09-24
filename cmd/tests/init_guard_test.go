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

// TestEnsureDflowNotInitializedUnit covers the validator in isolation with the
// repo's manual os.Chdir pattern (withWorkingDir), because Go 1.21 has no
// t.Chdir.
func TestEnsureDflowNotInitializedUnit(t *testing.T) {
	empty := t.TempDir()
	withWorkingDir(t, empty, func() {
		if err := validators.EnsureDflowNotInitialized(); err != nil {
			t.Fatalf("EnsureDflowNotInitialized() in an empty dir = %v, want nil", err)
		}
	})

	initialized := t.TempDir()
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
