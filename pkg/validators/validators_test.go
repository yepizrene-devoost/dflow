package validators

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureGitRepoDiscoversNestedDirectory(t *testing.T) {
	repo := initValidatorRepository(t)
	nested := filepath.Join(repo, "nested")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DFLOW_CWD", nested)

	if err := EnsureGitRepo(); err != nil {
		t.Fatalf("EnsureGitRepo() = %v, want nil", err)
	}
}

func TestEnsureGitRepoRejectsNonRepository(t *testing.T) {
	t.Setenv("DFLOW_CWD", t.TempDir())

	if err := EnsureGitRepo(); err == nil || err.Error() != "this is not a Git repository" {
		t.Fatalf("EnsureGitRepo() = %v, want repository error", err)
	}
}

func TestDflowValidatorsUseRepositoryRoot(t *testing.T) {
	repo := initValidatorRepository(t)
	nested := filepath.Join(repo, "nested")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".dflow.yaml"), []byte("branches: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DFLOW_CWD", nested)

	if err := EnsureDflowInitialized(); err != nil {
		t.Fatalf("EnsureDflowInitialized() = %v, want nil", err)
	}
	if err := EnsureDflowNotInitialized(); err == nil {
		t.Fatal("EnsureDflowNotInitialized() = nil, want initialized error")
	}
}

func TestDflowValidatorsRejectMissingConfig(t *testing.T) {
	repo := initValidatorRepository(t)
	t.Setenv("DFLOW_CWD", repo)

	if err := EnsureDflowInitialized(); err == nil || !strings.Contains(err.Error(), "dflow is not initialized") {
		t.Fatalf("EnsureDflowInitialized() = %v, want missing config error", err)
	}
	if err := EnsureDflowNotInitialized(); err != nil {
		t.Fatalf("EnsureDflowNotInitialized() = %v, want nil", err)
	}
}

func initValidatorRepository(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmd := exec.Command("git", "-C", dir, "init", "-q")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, output)
	}
	return dir
}
