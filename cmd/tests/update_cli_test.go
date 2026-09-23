package tests

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/yepizrene-devoost/dflow/cmd/selfupdate"
)

// TestUpdateCLIReplacesBinary covers the one behaviour a self-updater cannot get
// wrong: `dflow update` replaces the running binary with the verified release
// payload. It runs a copy of the built CLI, never the binary in the source tree,
// so the test can observe the swap without touching the checkout.
func TestUpdateCLIReplacesBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the Windows release archive is a zip; the zip path is covered by the cmd/selfupdate tests")
	}

	setUpCLIEnv(t)

	payload := []byte("#!/bin/sh\necho the-updated-dflow\n")
	server := startFakeReleaseServer(t, "v9.9.9", payload)
	t.Setenv("DFLOW_UPDATE_API_URL", server.URL)

	// A stale marker (older than the served tag) makes the version comparison
	// decide on its own that an update is available, exactly as it would for a
	// release-built binary. The plain `go build` marker is "dev", which has no
	// release provenance and is deliberately not comparable.
	source := buildDflowCLIWithMarker(t, "v0.1.0")

	installDir := t.TempDir()
	installed := filepath.Join(installDir, "dflow")
	copyExecutable(t, source, installed)

	before := readFileBytes(t, installed)
	if bytes.Equal(before, payload) {
		t.Fatalf("the installed copy already holds the release payload; the test cannot prove a swap happened")
	}
	resolved, err := filepath.EvalSymlinks(installed)
	if err != nil {
		t.Fatalf("could not resolve the install path %s: %v", installed, err)
	}

	output, exitCode := startCLIRawOutput(t, time.Minute, installDir, installed, "update", "--json")
	if exitCode != 0 {
		t.Fatalf("update --json exited %d, want 0\n%s", exitCode, output)
	}
	assertNoHumanChrome(t, output)

	doc := decodeSingleJSONDocument(t, output)
	if len(doc) != 6 {
		t.Fatalf("update document has %d keys, want exactly 6:\n%v", len(doc), doc)
	}
	requireJSONString(t, doc, "current_version", "v0.1.0")
	requireJSONString(t, doc, "latest_version", "v9.9.9")
	requireJSONString(t, doc, "path", resolved)
	if doc["update_available"] != true {
		t.Fatalf("update_available = %#v, want true", doc["update_available"])
	}
	if doc["updated"] != true {
		t.Fatalf("updated = %#v, want true: the binary was replaced, so the report must say so", doc["updated"])
	}
	if url, ok := doc["release_url"].(string); !ok || url == "" {
		t.Fatalf("release_url = %#v, want a non-empty string", doc["release_url"])
	}

	after := readFileBytes(t, installed)
	if !bytes.Equal(after, payload) {
		t.Fatalf("the installed binary was not replaced with the release payload:\ngot  %q\nwant %q", after, payload)
	}
}

// TestUpdateCLICheckReportsAnAvailableUpdate pins the read-only contract: a
// check asks the release API and reports, and it must not download or replace
// anything.
func TestUpdateCLICheckReportsAnAvailableUpdate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the Windows release archive is a zip; the zip path is covered by the cmd/selfupdate tests")
	}

	setUpCLIEnv(t)

	server := startFakeReleaseServer(t, "v9.9.9", []byte("payload"))
	t.Setenv("DFLOW_UPDATE_API_URL", server.URL)

	binary := buildDflowCLIWithMarker(t, "v0.1.0")

	output, exitCode := startCLIRawOutput(t, time.Minute, t.TempDir(), binary, "update", "--check", "--json")
	if exitCode != 0 {
		t.Fatalf("update --check --json exited %d, want 0\n%s", exitCode, output)
	}
	assertNoHumanChrome(t, output)

	doc := decodeSingleJSONDocument(t, output)
	if len(doc) != 6 {
		t.Fatalf("check document has %d keys, want exactly 6:\n%v", len(doc), doc)
	}
	requireJSONString(t, doc, "current_version", "v0.1.0")
	requireJSONString(t, doc, "latest_version", "v9.9.9")
	requireJSONString(t, doc, "path", "")
	if doc["update_available"] != true {
		t.Fatalf("update_available = %#v, want true for a stale binary", doc["update_available"])
	}
	if doc["updated"] != false {
		t.Fatalf("updated = %#v, want false: --check never replaces the binary", doc["updated"])
	}
}

// TestUpdateCLIAlreadyUpToDate pins the no-op answer. A binary whose version is
// not older than the release must not be replaced, and the report must say so
// through update_available and updated.
func TestUpdateCLIAlreadyUpToDate(t *testing.T) {
	setUpCLIEnv(t)

	// A binary stamped at the same version the server advertises has provenance
	// and is not older, so the comparison itself must stand down. (The marker
	// "dev" cannot test this path: without provenance the command always
	// reports an update, because there is always a release to move to.)
	server := startFakeReleaseServer(t, "v9.9.9", []byte("payload"))
	t.Setenv("DFLOW_UPDATE_API_URL", server.URL)

	binary := buildDflowCLIWithMarker(t, "v9.9.9")

	output, exitCode := startCLIRawOutput(t, time.Minute, t.TempDir(), binary, "update", "--check", "--json")
	if exitCode != 0 {
		t.Fatalf("update --check --json exited %d, want 0\n%s", exitCode, output)
	}
	assertNoHumanChrome(t, output)

	doc := decodeSingleJSONDocument(t, output)
	requireJSONString(t, doc, "current_version", "v9.9.9")
	requireJSONString(t, doc, "latest_version", "v9.9.9")
	requireJSONString(t, doc, "path", "")
	if doc["update_available"] != false {
		t.Fatalf("update_available = %#v, want false when the latest release is not newer", doc["update_available"])
	}
	if doc["updated"] != false {
		t.Fatalf("updated = %#v, want false", doc["updated"])
	}
}

// TestUpdateCLIRejectsCheckWithForce pins the input guard. The two flags
// contradict each other, so the command must fail before it reaches the
// network, and the failure must stay machine-readable under --json.
func TestUpdateCLIRejectsCheckWithForce(t *testing.T) {
	setUpCLIEnv(t)

	binary := buildDflowCLI(t)

	output, exitCode := startCLIRawOutput(t, 30*time.Second, t.TempDir(), binary, "update", "--check", "--force", "--json")
	if exitCode == 0 {
		t.Fatalf("update --check --force exited 0, want a non-zero exit\n%s", output)
	}
	assertNoHumanChrome(t, output)

	doc := decodeSingleJSONDocument(t, output)
	message, ok := doc["error"].(string)
	if !ok || message == "" {
		t.Fatalf("failure document has no non-empty string \"error\" key:\n%v", doc)
	}
	if !strings.Contains(message, "--check") || !strings.Contains(message, "--force") {
		t.Fatalf("error message does not name the conflicting flags: %q", message)
	}
}

// TestUpdateCLIWarnsWithoutReleaseProvenance covers the user the warning exists
// for: a binary built with a plain `go build` (the marker is "dev") cannot have
// its version compared, so the human renderer must explain why instead of
// pretending the comparison is meaningful.
func TestUpdateCLIWarnsWithoutReleaseProvenance(t *testing.T) {
	setUpCLIEnv(t)

	server := startFakeReleaseServer(t, "v9.9.9", []byte("payload"))
	t.Setenv("DFLOW_UPDATE_API_URL", server.URL)

	// buildDflowCLI injects no linker flag, so the binary reports the literal
	// "dev" marker this test is about.
	binary := buildDflowCLI(t)

	t.Run("human mode warns and reports the update", func(t *testing.T) {
		output, exitCode := startCLIRawOutput(t, time.Minute, t.TempDir(), binary, "update", "--check")
		if exitCode != 0 {
			t.Fatalf("update --check exited %d, want 0\n%s", exitCode, output)
		}
		if !strings.Contains(output, "no release provenance") {
			t.Fatalf("human output must explain the missing release provenance, got:\n%s", output)
		}
		// The confirmed semantics: without provenance there is always a release
		// to move to, so the verdict is "update available" — never "already up
		// to date", which would strand the user behind a --force they have no
		// reason to know about.
		if !strings.Contains(output, "an update is available") {
			t.Fatalf("a dev binary must report an available update, got:\n%s", output)
		}
	})

	t.Run("json mode reports the update without chrome", func(t *testing.T) {
		output, exitCode := startCLIRawOutput(t, time.Minute, t.TempDir(), binary, "update", "--check", "--json")
		if exitCode != 0 {
			t.Fatalf("update --check --json exited %d, want 0\n%s", exitCode, output)
		}
		assertNoHumanChrome(t, output)

		doc := decodeSingleJSONDocument(t, output)
		requireJSONString(t, doc, "current_version", "dev")
		requireJSONString(t, doc, "latest_version", "v9.9.9")
		if doc["update_available"] != true {
			t.Fatalf("update_available = %#v, want true for a binary without provenance", doc["update_available"])
		}
		if doc["updated"] != false {
			t.Fatalf("updated = %#v, want false: --check never replaces the binary", doc["updated"])
		}
	})
}

// buildDflowCLIWithMarker compiles the real entry point with a release version
// marker injected through the linker, the way .goreleaser.yaml and the Makefile
// stamp a release build. The version comparison is only meaningful for a binary
// that carries one, so the tests that exercise "is the release newer?" need a
// stamped binary; the provenance-warning test uses buildDflowCLI's plain "dev"
// build instead.
func buildDflowCLIWithMarker(t *testing.T, marker string) string {
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
	startCLICommand(t, 2*time.Minute, root, "go", "build", "-ldflags", "-X main.version="+marker, "-o", binary, ".")
	return binary
}

// startFakeReleaseServer serves the GitHub release endpoint and both release
// assets of a fabricated release, so the CLI's whole network surface is
// exercised without a single packet leaving the test host.
//
// The release payload is a real archive for the host platform (a tar.gz on
// Linux and macOS) whose checksum matches the served checksums file, so the
// success path runs the same verification a user's machine would.
func startFakeReleaseServer(t *testing.T, tag string, binaryPayload []byte) *httptest.Server {
	t.Helper()

	archiveName, err := selfupdate.ArchiveName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skipf("the test host %s/%s has no dflow release asset naming: %v", runtime.GOOS, runtime.GOARCH, err)
	}
	archive := buildReleaseArchive(t, archiveName, binaryPayload)
	checksumsName := selfupdate.ChecksumsName(tag)
	checksums := []byte(fmt.Sprintf("%s  %s\n", sha256Hex(archive), archiveName))

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	archiveURL := server.URL + "/assets/" + archiveName
	checksumsURL := server.URL + "/assets/" + checksumsName

	mux.HandleFunc("/repos/yepizrene-devoost/dflow/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tag_name": tag,
			"html_url": server.URL + "/repos/yepizrene-devoost/dflow/releases/tag/" + tag,
			"assets": []map[string]string{
				{"name": archiveName, "browser_download_url": archiveURL},
				{"name": checksumsName, "browser_download_url": checksumsURL},
			},
		})
	})
	mux.HandleFunc("/assets/"+archiveName, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(archive)
	})
	mux.HandleFunc("/assets/"+checksumsName, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(checksums)
	})

	return server
}

// buildReleaseArchive packs binaryPayload as the "dflow" entry of a gzipped tar
// archive, the layout .goreleaser.yaml publishes for Linux and macOS. The entry
// name, not the archive name, is what ExtractBinary looks for.
func buildReleaseArchive(t *testing.T, archiveName string, binaryPayload []byte) []byte {
	t.Helper()

	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)

	header := &tar.Header{
		Name:     "dflow",
		Mode:     0o755,
		Size:     int64(len(binaryPayload)),
		Typeflag: tar.TypeReg,
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatalf("write %s header: %v", archiveName, err)
	}
	if _, err := tarWriter.Write(binaryPayload); err != nil {
		t.Fatalf("write %s payload: %v", archiveName, err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close %s tar stream: %v", archiveName, err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close %s gzip stream: %v", archiveName, err)
	}
	return buffer.Bytes()
}

// sha256Hex renders the lower-case hex digest the release checksums file
// carries.
func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// copyExecutable copies source to dest with the executable mode, so the copy
// can be run and later replaced in place.
func copyExecutable(t *testing.T, source, dest string) {
	t.Helper()

	data := readFileBytes(t, source)
	if err := os.WriteFile(dest, data, 0o755); err != nil {
		t.Fatalf("copy %s to %s: %v", source, dest, err)
	}
	if err := os.Chmod(dest, 0o755); err != nil {
		t.Fatalf("mark %s executable: %v", dest, err)
	}
}

// readFileBytes reads a file into memory or fails the test, keeping the swap
// assertions free of repeated error handling.
func readFileBytes(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}
