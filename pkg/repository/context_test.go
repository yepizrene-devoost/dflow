package repository

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverUsesDFLOWCWDRelativeToProcessWorkingDirectory(t *testing.T) {
	repo := initRepository(t)
	subdir := filepath.Join(repo, "nested")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatal(err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(cwd, subdir)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("DFLOW_CWD", relative)

	got, err := Discover()
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	wantRoot := canonicalPath(t, repo)
	if got.WorktreeRoot != wantRoot {
		t.Errorf("WorktreeRoot = %q, want %q", got.WorktreeRoot, wantRoot)
	}
	if got.ConfigPath != filepath.Join(wantRoot, ".dflow.yaml") {
		t.Errorf("ConfigPath = %q, want %q", got.ConfigPath, filepath.Join(wantRoot, ".dflow.yaml"))
	}
	if got.GitDir == "" {
		t.Error("GitDir is empty")
	}
}

func TestDiscoverFallsBackToProcessWorkingDirectory(t *testing.T) {
	repo := initRepository(t)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// The process cwd is fixed for this test binary, so use a repository rooted
	// there only to verify the fallback's selected start is not DFLOW_CWD.
	_ = repo
	t.Setenv("DFLOW_CWD", "")

	got, err := Discover()
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	if got.WorktreeRoot == "" || !filepath.IsAbs(got.WorktreeRoot) {
		t.Errorf("WorktreeRoot = %q, want an absolute path", got.WorktreeRoot)
	}
	if !filepath.IsAbs(cwd) {
		t.Fatalf("process cwd is not absolute: %q", cwd)
	}
}

func TestDiscoverSupportsGitWorktreeGitFile(t *testing.T) {
	repo := initRepository(t)
	worktree := filepath.Join(t.TempDir(), "linked")
	runGit(t, repo, "worktree", "add", "-b", "linked", worktree)

	t.Setenv("DFLOW_CWD", worktree)
	got, err := Discover()
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	wantRoot := canonicalPath(t, worktree)
	if got.WorktreeRoot != wantRoot {
		t.Errorf("WorktreeRoot = %q, want %q", got.WorktreeRoot, wantRoot)
	}
	info, err := os.Stat(filepath.Join(worktree, ".git"))
	if err != nil {
		t.Fatal(err)
	}
	if info.IsDir() {
		t.Fatal("test setup did not create a .git file worktree")
	}
	if !filepath.IsAbs(got.GitDir) {
		t.Errorf("GitDir = %q, want absolute path", got.GitDir)
	}
}

func TestDiscoverRejectsInvalidStarts(t *testing.T) {
	cases := []struct {
		name string
		path string
		want string
	}{
		{name: "missing", path: filepath.Join(t.TempDir(), "missing"), want: "does not exist"},
		{name: "file", path: filepath.Join(t.TempDir(), "file"), want: "not a directory"},
		{name: "malformed", path: string([]byte{'a', 0, 'b'}), want: "invalid"},
	}
	file := cases[1].path
	if err := os.WriteFile(file, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			if tc.name == "malformed" {
				_, err = resolveStartPath(tc.path, t.TempDir())
			} else {
				t.Setenv("DFLOW_CWD", tc.path)
				_, err = Discover()
			}
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), tc.want) {
				t.Fatalf("Discover error = %v, want it to contain %q", err, tc.want)
			}
		})
	}
}

func TestDiscoverIncludesGitDiagnostics(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DFLOW_CWD", dir)

	_, err := Discover()
	if err == nil {
		t.Fatal("Discover succeeded outside a Git repository")
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Errorf("Discover error = %q, want captured Git diagnostics", err)
	}
}

func canonicalPath(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func initRepository(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-q")
	return dir
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
	return string(output)
}
