package selfupdate

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// stubEnv replaces the package's environment lookup for the duration of one
// test. Every cache-path and base-URL decision reads the environment through
// that seam, so a test can pin XDG_CACHE_HOME, HOME and DFLOW_UPDATE_API_URL
// without depending on — or mutating — the host's real values.
func stubEnv(t *testing.T, values map[string]string) {
	t.Helper()

	original := lookupEnv
	lookupEnv = func(key string) string { return values[key] }
	t.Cleanup(func() { lookupEnv = original })
}

// TestUpdateCheckTTLIsADay pins the product decision the freshness rule is
// built on. A later refactor that silently widened or narrowed the window would
// change how often dflow calls GitHub, which is exactly the trade-off the issue
// settled at 24 hours.
func TestUpdateCheckTTLIsADay(t *testing.T) {
	if updateCheckTTL != 24*time.Hour {
		t.Fatalf("updateCheckTTL = %v, want 24h", updateCheckTTL)
	}
}

// TestProbeHTTPTimeoutStaysWellUnderTheCommandBudget guards the reason the
// probe has its own client at all: it races the command the user asked for, so
// its timeout must stay far below the interactive 30s `dflow update` budget.
func TestProbeHTTPTimeoutStaysWellUnderTheCommandBudget(t *testing.T) {
	if probeHTTPTimeout <= 0 {
		t.Fatalf("probeHTTPTimeout = %v, want a positive duration", probeHTTPTimeout)
	}
	if probeHTTPTimeout >= defaultHTTPTimeout {
		t.Fatalf("probeHTTPTimeout = %v, want it below the %v update-client timeout", probeHTTPTimeout, defaultHTTPTimeout)
	}
	if probeHTTPTimeout > 5*time.Second {
		t.Fatalf("probeHTTPTimeout = %v, want at most a few seconds so startup never waits noticeably", probeHTTPTimeout)
	}
}

// TestUpdateCheckCachePath pins the cache location: XDG_CACHE_HOME when set,
// the classic ~/.cache fallback otherwise, and a loud error when the machine
// offers neither. The paths are asserted with filepath.Join so the expectation
// stays platform-shaped rather than hard-coding a separator.
func TestUpdateCheckCachePath(t *testing.T) {
	cases := []struct {
		name    string
		env     map[string]string
		want    string
		wantErr bool
	}{
		{
			name: "xdg cache home wins",
			env:  map[string]string{updateCacheDirEnv: "/xdg/cache", updateHomeEnv: "/home/tester"},
			want: filepath.Join("/xdg/cache", updateCacheDirName, updateCacheFileName),
		},
		{
			name: "falls back to home when xdg is absent",
			env:  map[string]string{updateHomeEnv: "/home/tester"},
			want: filepath.Join("/home/tester", ".cache", updateCacheDirName, updateCacheFileName),
		},
		{
			name: "blank xdg falls back to home",
			env:  map[string]string{updateCacheDirEnv: "   ", updateHomeEnv: "/home/tester"},
			want: filepath.Join("/home/tester", ".cache", updateCacheDirName, updateCacheFileName),
		},
		{
			name: "surrounding whitespace is trimmed",
			env:  map[string]string{updateCacheDirEnv: "  /xdg/cache  "},
			want: filepath.Join("/xdg/cache", updateCacheDirName, updateCacheFileName),
		},
		{
			name:    "no xdg and no home",
			env:     map[string]string{},
			wantErr: true,
		},
		{
			name:    "blank home is treated as absent",
			env:     map[string]string{updateHomeEnv: "  "},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stubEnv(t, tc.env)

			got, err := UpdateCheckCachePath()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("UpdateCheckCachePath() error = nil, want an error; got path %q", got)
				}
				for _, want := range []string{updateCacheDirEnv, updateHomeEnv} {
					if !strings.Contains(err.Error(), want) {
						t.Fatalf("UpdateCheckCachePath() error = %q, want it to name %q", err.Error(), want)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("UpdateCheckCachePath() error = %v, want nil", err)
			}
			if got != tc.want {
				t.Fatalf("UpdateCheckCachePath() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestReadUpdateCheckStateMissingFileIsFirstRun pins the one non-error absence:
// a clean machine has no cache yet, and that must read as "no information"
// rather than a failure the startup path would have to special-case.
func TestReadUpdateCheckStateMissingFileIsFirstRun(t *testing.T) {
	path := filepath.Join(t.TempDir(), updateCacheFileName)

	state, err := ReadUpdateCheckState(path)
	if err != nil {
		t.Fatalf("ReadUpdateCheckState(missing) error = %v, want nil", err)
	}
	if state != nil {
		t.Fatalf("ReadUpdateCheckState(missing) = %+v, want nil", state)
	}
}

// TestReadUpdateCheckStateParsesTheDocumentedKeys pins the wire format with
// hand-written JSON, so a renamed struct tag is caught here even though the
// round-trip test would still pass by encoding and decoding its own mistake.
func TestReadUpdateCheckStateParsesTheDocumentedKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), updateCacheFileName)
	body := `{"last_checked":"2026-01-02T03:04:05Z","latest_version":"v9.9.9","release_url":"https://example.test/releases/v9.9.9"}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("could not seed the cache: %v", err)
	}

	state, err := ReadUpdateCheckState(path)
	if err != nil {
		t.Fatalf("ReadUpdateCheckState() error = %v, want nil", err)
	}
	if state == nil {
		t.Fatal("ReadUpdateCheckState() = nil, want a decoded state")
	}
	if want := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC); !state.LastChecked.Equal(want) {
		t.Fatalf("LastChecked = %v, want %v", state.LastChecked, want)
	}
	if state.LatestVersion != "v9.9.9" {
		t.Fatalf("LatestVersion = %q, want %q", state.LatestVersion, "v9.9.9")
	}
	if state.ReleaseURL != "https://example.test/releases/v9.9.9" {
		t.Fatalf("ReleaseURL = %q", state.ReleaseURL)
	}
}

// TestReadUpdateCheckStateRejectsMalformedJSON is the "a cache you cannot trust
// is not an absent cache" case: a truncated or corrupted file must surface as
// an error so the caller can decide, not be silently treated as a first run.
func TestReadUpdateCheckStateRejectsMalformedJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), updateCacheFileName)
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatalf("could not seed the cache: %v", err)
	}

	_, err := ReadUpdateCheckState(path)
	if err == nil {
		t.Fatal("ReadUpdateCheckState(malformed) error = nil, want an error")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Fatalf("ReadUpdateCheckState(malformed) error = %q, want it to mention decoding", err.Error())
	}
}

// TestReadUpdateCheckStateRejectsAnUnreadablePath covers the other failure the
// reader promises to report. A directory is used as the unreadable path because
// it fails on every platform regardless of the user running the test, unlike a
// chmod-based permission fixture that root would defeat.
func TestReadUpdateCheckStateRejectsAnUnreadablePath(t *testing.T) {
	if _, err := ReadUpdateCheckState(t.TempDir()); err == nil {
		t.Fatal("ReadUpdateCheckState(directory) error = nil, want an error")
	}
}

// TestWriteAndReadUpdateCheckStateRoundTrip is the ordinary persistence path:
// the parent directory is created on demand and every field survives the trip.
func TestWriteAndReadUpdateCheckStateRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", updateCacheDirName, updateCacheFileName)
	want := UpdateCheckState{
		LastChecked:   time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		LatestVersion: "v0.3.0",
		ReleaseURL:    "https://example.test/releases/v0.3.0",
	}

	if err := WriteUpdateCheckState(path, want); err != nil {
		t.Fatalf("WriteUpdateCheckState() error = %v, want nil", err)
	}

	got, err := ReadUpdateCheckState(path)
	if err != nil {
		t.Fatalf("ReadUpdateCheckState() error = %v, want nil", err)
	}
	if got == nil {
		t.Fatal("ReadUpdateCheckState() = nil, want the written state")
	}
	if !got.LastChecked.Equal(want.LastChecked) {
		t.Fatalf("LastChecked = %v, want %v", got.LastChecked, want.LastChecked)
	}
	if got.LatestVersion != want.LatestVersion {
		t.Fatalf("LatestVersion = %q, want %q", got.LatestVersion, want.LatestVersion)
	}
	if got.ReleaseURL != want.ReleaseURL {
		t.Fatalf("ReleaseURL = %q, want %q", got.ReleaseURL, want.ReleaseURL)
	}
}

// TestWriteUpdateCheckStateLeavesNoTemporaryFile proves the staging file does
// not survive a successful write. A leftover dot-file would be invisible to a
// user but would accumulate one file per check, which is precisely the residue
// the rename-over pattern is chosen to avoid.
func TestWriteUpdateCheckStateLeavesNoTemporaryFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), updateCacheDirName)
	path := filepath.Join(dir, updateCacheFileName)

	state := UpdateCheckState{LastChecked: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), LatestVersion: "v0.3.0"}
	if err := WriteUpdateCheckState(path, state); err != nil {
		t.Fatalf("WriteUpdateCheckState() error = %v, want nil", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("could not list the cache directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != updateCacheFileName {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Fatalf("cache directory holds %v, want only %q", names, updateCacheFileName)
	}
}

// TestWriteUpdateCheckStateReportsAnUnencodableState covers the encode failure
// path. A year outside the RFC3339 range is the one way a well-typed state can
// fail to marshal, and the write must report it without leaving a cache file.
func TestWriteUpdateCheckStateReportsAnUnencodableState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, updateCacheFileName)

	state := UpdateCheckState{LastChecked: time.Date(10000, 1, 2, 3, 4, 5, 0, time.UTC), LatestVersion: "v0.3.0"}
	if err := WriteUpdateCheckState(path, state); err == nil {
		t.Fatal("WriteUpdateCheckState(year 10000) error = nil, want an encoding error")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("os.Stat(%s) error = %v, want the cache file to be absent", path, err)
	}
}

// TestIsUpdateCheckFresh pins the freshness rule at its boundaries. The strict
// comparison matters: a state recorded exactly TTL ago belongs to the next
// check, so off-by-one here would either delay or double every request.
func TestIsUpdateCheckFresh(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	cases := []struct {
		name  string
		state *UpdateCheckState
		want  bool
	}{
		{
			name:  "no state at all",
			state: nil,
			want:  false,
		},
		{
			name:  "zero last checked",
			state: &UpdateCheckState{LastChecked: time.Time{}},
			want:  false,
		},
		{
			name:  "checked just now",
			state: &UpdateCheckState{LastChecked: now},
			want:  true,
		},
		{
			name:  "one nanosecond inside the ttl",
			state: &UpdateCheckState{LastChecked: now.Add(-(updateCheckTTL - time.Nanosecond))},
			want:  true,
		},
		{
			name:  "exactly the ttl",
			state: &UpdateCheckState{LastChecked: now.Add(-updateCheckTTL)},
			want:  false,
		},
		{
			name:  "older than the ttl",
			state: &UpdateCheckState{LastChecked: now.Add(-updateCheckTTL - time.Nanosecond)},
			want:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsUpdateCheckFresh(tc.state, now); got != tc.want {
				t.Fatalf("IsUpdateCheckFresh() = %t, want %t", got, tc.want)
			}
		})
	}
}

// TestShouldProbeUpdate mirrors the freshness rule and nothing else: the
// suppression rules (dev builds, opt-out, machine-readable runs) are the wiring
// layer's business, so this function must stay a pure cache question.
func TestShouldProbeUpdate(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	cases := []struct {
		name  string
		state *UpdateCheckState
		want  bool
	}{
		{name: "first run probes", state: nil, want: true},
		{name: "stale cache probes", state: &UpdateCheckState{LastChecked: now.Add(-updateCheckTTL)}, want: true},
		{name: "fresh cache does not probe", state: &UpdateCheckState{LastChecked: now.Add(-time.Hour)}, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ShouldProbeUpdate(tc.state, now); got != tc.want {
				t.Fatalf("ShouldProbeUpdate() = %t, want %t", got, tc.want)
			}
		})
	}
}

// TestRecordUpdateCheckWritesNormalizedState pins what gets persisted: the
// release's normalized tag (leading `v`) and its page, stamped with the clock
// the caller passed rather than a hidden time.Now.
func TestRecordUpdateCheckWritesNormalizedState(t *testing.T) {
	path := filepath.Join(t.TempDir(), updateCacheFileName)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	release := &Release{Tag: "v0.4.0", Version: "0.4.0", HTMLURL: "https://example.test/releases/v0.4.0"}

	if err := RecordUpdateCheck(path, release, now); err != nil {
		t.Fatalf("RecordUpdateCheck() error = %v, want nil", err)
	}

	state, err := ReadUpdateCheckState(path)
	if err != nil {
		t.Fatalf("ReadUpdateCheckState() error = %v, want nil", err)
	}
	if state == nil {
		t.Fatal("ReadUpdateCheckState() = nil, want the recorded state")
	}
	if !state.LastChecked.Equal(now) {
		t.Fatalf("LastChecked = %v, want %v", state.LastChecked, now)
	}
	if state.LatestVersion != release.Tag {
		t.Fatalf("LatestVersion = %q, want %q", state.LatestVersion, release.Tag)
	}
	if state.ReleaseURL != release.HTMLURL {
		t.Fatalf("ReleaseURL = %q, want %q", state.ReleaseURL, release.HTMLURL)
	}
}

// TestRecordUpdateCheckIgnoresNilRelease pins the no-op: a probe that produced
// nothing must not record an empty version as if it were an answer, and must
// not create a cache file at all.
func TestRecordUpdateCheckIgnoresNilRelease(t *testing.T) {
	path := filepath.Join(t.TempDir(), updateCacheFileName)

	if err := RecordUpdateCheck(path, nil, time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)); err != nil {
		t.Fatalf("RecordUpdateCheck(nil) error = %v, want nil", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("os.Stat(%s) error = %v, want no cache file to be written", path, err)
	}
}

// TestResolveBaseURL pins the override contract the probe shares with
// `dflow update`: a real value wins, a blank one is ignored so it cannot turn
// every lookup into a request to the empty string.
func TestResolveBaseURL(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want string
	}{
		{
			name: "override wins",
			env:  map[string]string{updateBaseURLEnv: "https://mirror.test/api"},
			want: "https://mirror.test/api",
		},
		{
			name: "surrounding whitespace is trimmed",
			env:  map[string]string{updateBaseURLEnv: "  https://mirror.test/api  "},
			want: "https://mirror.test/api",
		},
		{
			name: "blank override falls back",
			env:  map[string]string{updateBaseURLEnv: "   "},
			want: DefaultBaseURL,
		},
		{
			name: "unset override falls back",
			env:  map[string]string{},
			want: DefaultBaseURL,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stubEnv(t, tc.env)

			if got := ResolveBaseURL(); got != tc.want {
				t.Fatalf("ResolveBaseURL() = %q, want %q", got, tc.want)
			}
		})
	}
}

// newTestProbeServer wires a throwaway httptest server for the probe tests.
func newTestProbeServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

// TestProbeLatestReleaseParsesTheGitHubResponse pins the probe's request shape
// and normalization: it hits the same `releases/latest` endpoint as the update
// client and returns a Release whose tag carries its leading `v`.
func TestProbeLatestReleaseParsesTheGitHubResponse(t *testing.T) {
	var requestedPath string

	server := newTestProbeServer(t, func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tag_name": "0.4.1",
			"html_url": "https://example.test/releases/v0.4.1",
		})
	})

	release, err := ProbeLatestRelease(server.URL)
	if err != nil {
		t.Fatalf("ProbeLatestRelease() error = %v, want nil", err)
	}
	if requestedPath != releasesPath {
		t.Fatalf("requested path = %q, want %q", requestedPath, releasesPath)
	}
	if release.Tag != "v0.4.1" {
		t.Fatalf("Tag = %q, want %q", release.Tag, "v0.4.1")
	}
	if release.HTMLURL != "https://example.test/releases/v0.4.1" {
		t.Fatalf("HTMLURL = %q", release.HTMLURL)
	}
}

// TestProbeLatestReleaseReportsHTTPFailures proves the probe reuses the update
// client's error translation rather than inventing its own: a non-200 must be
// an error, never a zero-value release that would look like version 0.0.0.
func TestProbeLatestReleaseReportsHTTPFailures(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := newTestProbeServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
			})

			release, err := ProbeLatestRelease(server.URL)
			if err == nil {
				t.Fatalf("ProbeLatestRelease() error = nil for HTTP %d, want an error", status)
			}
			if release != nil {
				t.Fatalf("ProbeLatestRelease() release = %+v, want nil on failure", release)
			}
		})
	}
}

// TestProbeLatestReleaseFallsBackToResolvedBaseURL covers the blank-argument
// fallback: a caller that passes no endpoint must still get the base URL the
// environment resolves to, not a request built against the empty string.
func TestProbeLatestReleaseFallsBackToResolvedBaseURL(t *testing.T) {
	server := newTestProbeServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"tag_name": "v0.5.0"})
	})
	stubEnv(t, map[string]string{updateBaseURLEnv: server.URL})

	release, err := ProbeLatestRelease("   ")
	if err != nil {
		t.Fatalf("ProbeLatestRelease(blank) error = %v, want nil", err)
	}
	if release.Tag != "v0.5.0" {
		t.Fatalf("Tag = %q, want %q", release.Tag, "v0.5.0")
	}
}
