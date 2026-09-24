package tests

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// This file owns the CLI binary the process tests run: one `go build` per
// distinct linker configuration, shared by every test in the package.
//
// Two costs used to be paid once per test. The first was a cold compile:
// setUpCLIEnv points XDG_CACHE_HOME at a fresh t.TempDir() so the update-check
// cache stays isolated, and the spawned `go build` inherited it. On Linux
// os.UserCacheDir() honors $XDG_CACHE_HOME, so the child's default GOCACHE was
// an empty per-test directory. The second was the build itself, repeated by
// every test even against a warm cache.
//
// Anchoring the child's GOCACHE restores the shared, warm compiler cache;
// memoizing per linker configuration removes the repeated re-link. Both leave
// the XDG_CACHE_HOME isolation exactly as it was.

// realGOCACHE is the host's Go build cache, captured at package initialization.
// Package-level variables are initialized before TestMain and before any test
// can call t.Setenv, so this is the host's real cache rather than a test-local
// temp directory.
//
// When the query fails or returns nothing the value stays empty and the spawned
// build inherits the environment unchanged: correct, only slower.
var realGOCACHE = func() string {
	out, err := exec.Command("go", "env", "GOCACHE").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}()

// cliBuilds holds the shared build directory and one memoized result per
// distinct linker configuration.
var cliBuilds = struct {
	dirOnce sync.Once
	dir     string
	dirErr  error

	mu    sync.Mutex
	byKey map[string]*cliBuild
}{
	byKey: make(map[string]*cliBuild),
}

// cliBuild is a single memoized `go build`. The failure and its captured output
// are cached alongside the binary, so every test asking for a configuration
// that cannot build observes the same error and the same diagnostics.
type cliBuild struct {
	once   sync.Once
	binary string
	output string
	err    error
}

// dflowBinaryName is the platform-correct file name for a built dflow binary.
func dflowBinaryName() string {
	name := "dflow"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

// sharedBuildDir creates the one directory every memoized build lives in, and
// returns the same path to every caller.
func sharedBuildDir() (string, error) {
	cliBuilds.dirOnce.Do(func() {
		cliBuilds.dir, cliBuilds.dirErr = os.MkdirTemp("", "dflow-cli-test-builds-")
	})
	return cliBuilds.dir, cliBuilds.dirErr
}

// buildEntry returns the slot for one linker configuration, creating it on
// first use so that concurrent callers contend for the same build instead of
// each starting one.
func buildEntry(ldflags string) *cliBuild {
	cliBuilds.mu.Lock()
	defer cliBuilds.mu.Unlock()

	entry, ok := cliBuilds.byKey[ldflags]
	if !ok {
		entry = &cliBuild{}
		cliBuilds.byKey[ldflags] = entry
	}
	return entry
}

// runSharedBuild compiles the real entry point into the shared build directory
// with the given linker flags, where "" means no -ldflags argument at all. It
// reports the captured output even on failure so the caller can surface it.
//
// Each configuration builds into its own subdirectory of the shared directory.
// They must not share one output path: a later build would overwrite the
// artifact a previously memoized configuration still points at.
func runSharedBuild(ldflags string) (binary, output string, err error) {
	dir, err := sharedBuildDir()
	if err != nil {
		return "", "", fmt.Errorf("create the shared CLI build directory: %w", err)
	}
	configDir, err := os.MkdirTemp(dir, "config-")
	if err != nil {
		return "", "", fmt.Errorf("create the build directory for ldflags %q: %w", ldflags, err)
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		return "", "", fmt.Errorf("resolve the repository root: %w", err)
	}
	binary = filepath.Join(configDir, dflowBinaryName())

	args := []string{"build"}
	if ldflags != "" {
		args = append(args, "-ldflags", ldflags)
	}
	args = append(args, "-o", binary, ".")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = root
	// Anchor the child's compiler cache when the host's could be captured; the
	// test-local XDG_CACHE_HOME stays in place either way.
	if realGOCACHE != "" {
		cmd.Env = append(os.Environ(), "GOCACHE="+realGOCACHE)
	}

	raw, buildErr := cmd.CombinedOutput()
	output = string(raw)
	if ctx.Err() != nil {
		return "", output, fmt.Errorf("go %s timed out after 2m: %w", strings.Join(args, " "), ctx.Err())
	}
	if buildErr != nil {
		return "", output, fmt.Errorf("go %s failed: %w", strings.Join(args, " "), buildErr)
	}
	return binary, output, nil
}

// sharedDflowCLI returns a fresh copy of the memoized binary for one linker
// configuration, building it the first time that configuration is requested.
//
// The copy is what makes sharing safe: TestUpdateCLIReplacesBinary replaces the
// binary it runs, so handing out the shared file itself would let one test
// corrupt every later test. A copy failure fails the calling test.
func sharedDflowCLI(t *testing.T, ldflags string) string {
	t.Helper()

	entry := buildEntry(ldflags)
	entry.once.Do(func() {
		entry.binary, entry.output, entry.err = runSharedBuild(ldflags)
	})
	if entry.err != nil {
		t.Fatalf("build the shared dflow binary (ldflags %q): %v\n%s", ldflags, entry.err, entry.output)
	}
	target := filepath.Join(t.TempDir(), dflowBinaryName())

	data, err := os.ReadFile(entry.binary)
	if err != nil {
		t.Fatalf("read the shared dflow binary %s: %v", entry.binary, err)
	}
	// WriteFile applies 0o755 at creation, which keeps the copy executable;
	// the file is fresh in a t.TempDir(), so no follow-up chmod is needed.
	if err := os.WriteFile(target, data, 0o755); err != nil {
		t.Fatalf("copy the shared dflow binary to %s: %v", target, err)
	}
	return target
}

// TestMain removes the shared build directory once the package has run. The
// directory is created lazily by the first build, so a run that builds nothing
// removes nothing.
func TestMain(m *testing.M) {
	code := m.Run()
	_ = os.RemoveAll(cliBuilds.dir)
	os.Exit(code)
}
