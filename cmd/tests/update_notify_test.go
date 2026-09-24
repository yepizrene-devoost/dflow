package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// updateNoticeExactLine is the one-line stderr contract, spelled out here rather
// than rebuilt from the implementation constant so the test fails if the
// wording drifts, not just if the plumbing breaks.
func updateNoticeExactLine(tag string) string {
	return "A new dflow release is available: " + tag + " — run dflow update"
}

// updateNoticeRepo prepares the working directory the notice tests run in: a Git
// repository (the config commands require `.git`) with a `.dflow.yaml`, which
// the same commands require to exist. Its content is never parsed.
func updateNoticeRepo(t *testing.T) string {
	t.Helper()

	repo := initTempGitRepo(t)
	if err := os.WriteFile(filepath.Join(repo, ".dflow.yaml"), []byte(""), 0o644); err != nil {
		t.Fatalf("write .dflow.yaml: %v", err)
	}
	return repo
}

// updateNoticeRunCLI runs the built binary with stdout and stderr captured
// separately, so a test can prove the notice reached stderr and left stdout
// alone. The existing helpers capture the combined stream, which cannot show
// which stream a line came from.
func updateNoticeRunCLI(t *testing.T, timeout time.Duration, dir, binary string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = dir
	// A nil Stdin reads from the null device: EOF, never an interactive TTY.
	cmd.Stdin = nil
	cmd.WaitDelay = time.Second

	var stdoutBuffer, stderrBuffer bytes.Buffer
	cmd.Stdout = &stdoutBuffer
	cmd.Stderr = &stderrBuffer

	err := cmd.Run()
	stdout, stderr = stdoutBuffer.String(), stderrBuffer.String()
	if ctx.Err() != nil {
		t.Fatalf("%s %v timed out: %v\nstdout:\n%s\nstderr:\n%s", binary, args, ctx.Err(), stdout, stderr)
	}
	if err == nil {
		return stdout, stderr, 0
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("%s %v did not run: %v", binary, args, err)
	}
	return stdout, stderr, exitErr.ExitCode()
}

// updateNoticeStartReleaseServer serves the one endpoint the startup probe
// queries — the latest release — and counts every request, so a test can prove a
// cached run asked GitHub zero times. It serves nothing else on purpose: the
// probe never downloads an asset.
func updateNoticeStartReleaseServer(t *testing.T, tag string, requests *int32) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/repos/yepizrene-devoost/dflow/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(requests, 1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tag_name": tag,
			"html_url": server.URL + "/repos/yepizrene-devoost/dflow/releases/tag/" + tag,
		})
	})

	return server
}

// TestUpdateNoticePrintsOneLineOnStderrAndCaches is the end-to-end pin for the
// notification: a release-stamped binary talking to a fake release server must
// print exactly one plain line on stderr, after the command's own output, leave
// stdout untouched, exit 0, and then answer the next run from the cache without
// asking the server again.
func TestUpdateNoticePrintsOneLineOnStderrAndCaches(t *testing.T) {
	setUpCLIEnv(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	var requests int32
	server := updateNoticeStartReleaseServer(t, "v9.9.9", &requests)
	t.Setenv("DFLOW_UPDATE_API_URL", server.URL)

	binary := buildDflowCLIWithMarker(t, "v0.1.0")
	repo := updateNoticeRepo(t)
	want := updateNoticeExactLine("v9.9.9")

	t.Run("first run probes and notifies", func(t *testing.T) {
		stdout, stderr, exitCode := updateNoticeRunCLI(t, time.Minute, repo, binary, "config", "list")
		if exitCode != 0 {
			t.Fatalf("config list exited %d, want 0\nstdout:\n%s\nstderr:\n%s", exitCode, stdout, stderr)
		}
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr does not carry the notice; want %q in:\n%s", want, stderr)
		}
		if strings.Contains(stdout, "A new dflow release") {
			t.Fatalf("the notice leaked onto stdout:\n%s", stdout)
		}
		if strings.TrimSpace(stdout) == "" {
			t.Fatalf("the command produced no stdout output; the notice cannot be proven to follow it")
		}
		if got := atomic.LoadInt32(&requests); got != 1 {
			t.Fatalf("the first run made %d release requests, want exactly 1", got)
		}
	})

	t.Run("second run is answered by the cache", func(t *testing.T) {
		// Combined output here, so the notice can be shown to follow the
		// command's own line: PersistentPostRun is what puts it last.
		output, exitCode := startCLIRawOutput(t, time.Minute, repo, binary, "config", "list")
		if exitCode != 0 {
			t.Fatalf("config list exited %d, want 0\n%s", exitCode, output)
		}
		if !strings.Contains(output, want) {
			t.Fatalf("the cached run did not render the notice; want %q in:\n%s", want, output)
		}
		if !strings.HasSuffix(strings.TrimSpace(output), want) {
			t.Fatalf("the notice must be the last line, after the command's own output, got:\n%s", output)
		}
		if got := atomic.LoadInt32(&requests); got != 1 {
			t.Fatalf("the second run made %d release requests, want 1: a fresh cache must not re-probe", got)
		}
	})
}

// TestUpdateNoticeHonoursNoUpdateCheckEnv pins the documented opt-out: a
// non-empty DFLOW_NO_UPDATE_CHECK must silence both the notice and the network
// request.
func TestUpdateNoticeHonoursNoUpdateCheckEnv(t *testing.T) {
	setUpCLIEnv(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("DFLOW_NO_UPDATE_CHECK", "1")

	var requests int32
	server := updateNoticeStartReleaseServer(t, "v9.9.9", &requests)
	t.Setenv("DFLOW_UPDATE_API_URL", server.URL)

	binary := buildDflowCLIWithMarker(t, "v0.1.0")
	repo := updateNoticeRepo(t)

	stdout, stderr, exitCode := updateNoticeRunCLI(t, time.Minute, repo, binary, "config", "list")
	if exitCode != 0 {
		t.Fatalf("config list exited %d, want 0\nstdout:\n%s\nstderr:\n%s", exitCode, stdout, stderr)
	}
	if strings.Contains(stderr, "A new dflow release") {
		t.Fatalf("DFLOW_NO_UPDATE_CHECK must suppress the notice, got:\n%s", stderr)
	}
	if got := atomic.LoadInt32(&requests); got != 0 {
		t.Fatalf("DFLOW_NO_UPDATE_CHECK must skip the network too, but %d requests were made", got)
	}
}

// TestUpdateNoticeStaysSilentWithoutReleaseProvenance pins the provenance gate:
// a plain `go build` ("dev") has no version to compare, so it must neither probe
// nor notify.
func TestUpdateNoticeStaysSilentWithoutReleaseProvenance(t *testing.T) {
	setUpCLIEnv(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	var requests int32
	server := updateNoticeStartReleaseServer(t, "v9.9.9", &requests)
	t.Setenv("DFLOW_UPDATE_API_URL", server.URL)

	// buildDflowCLI injects no linker flag, so the binary reports the literal
	// "dev" marker this test is about.
	binary := buildDflowCLI(t)
	repo := updateNoticeRepo(t)

	stdout, stderr, exitCode := updateNoticeRunCLI(t, time.Minute, repo, binary, "config", "list")
	if exitCode != 0 {
		t.Fatalf("config list exited %d, want 0\nstdout:\n%s\nstderr:\n%s", exitCode, stdout, stderr)
	}
	if strings.Contains(stderr, "A new dflow release") {
		t.Fatalf("a dev build must not notify, got:\n%s", stderr)
	}
	if got := atomic.LoadInt32(&requests); got != 0 {
		t.Fatalf("a dev build must not probe, but %d requests were made", got)
	}
}
