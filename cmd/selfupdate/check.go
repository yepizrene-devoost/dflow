package selfupdate

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// updateCheckTTL is how long a recorded update check stays authoritative. A
// result younger than this is reused without asking GitHub again, so a user who
// runs several dflow commands in a row pays for at most one probe per day. The
// value is the issue's product decision, not a tuning knob: long enough to stay
// far inside GitHub's unauthenticated rate limit, short enough that a release
// published in the morning is noticed the same day.
const updateCheckTTL = 24 * time.Hour

// probeHTTPTimeout bounds the background startup probe. It is deliberately far
// shorter than the 30s the interactive `dflow update` client allows, because
// the probe competes with the command the user actually asked for: a slow or
// unreachable GitHub must cost the startup path at most a couple of seconds
// before the check is abandoned for this run and retried on the next one.
const probeHTTPTimeout = 2 * time.Second

// The environment variables this file reads. They are constants so the names
// documented to users live in one place rather than in scattered `os.Getenv`
// string literals.
const (
	// updateCacheDirEnv overrides the base cache directory, per the XDG Base
	// Directory Specification.
	updateCacheDirEnv = "XDG_CACHE_HOME"
	// updateHomeEnv is the fallback for a machine with no XDG_CACHE_HOME set.
	updateHomeEnv = "HOME"
	// updateBaseURLEnv overrides the GitHub API base URL, the same variable
	// `dflow update` documents, so both paths honour one override.
	updateBaseURLEnv = "DFLOW_UPDATE_API_URL"
)

// The cache layout: `<base>/dflow/update-check.json`.
const (
	updateCacheDirName  = "dflow"
	updateCacheFileName = "update-check.json"
)

// lookupEnv is the environment lookup every override in this file goes through.
//
// It is a variable rather than a direct os.Getenv call so tests can drive the
// cache path and the API base URL without mutating the process environment: the
// suite must not depend on the host's real HOME, and t.Setenv would make the
// test process's environment the test's own mutable global state.
var lookupEnv = os.Getenv

// UpdateCheckState is the cached outcome of one update check: when it ran and
// what it found.
//
// It is the on-disk contract between the process that probed GitHub and every
// later process that reads the answer, so the JSON keys are part of the public
// surface: renaming a field breaks the cache of an already-installed binary
// until it expires.
type UpdateCheckState struct {
	// LastChecked is when the probe recorded this result, in RFC3339. It is the
	// only field freshness is judged on; a zero value means "never checked".
	LastChecked time.Time `json:"last_checked"`
	// LatestVersion is the normalized release tag, with its leading `v` (e.g.
	// "v0.2.0"). Storing the shape the rest of the package compares keeps every
	// reader from re-normalizing it.
	LatestVersion string `json:"latest_version"`
	// ReleaseURL is the GitHub release page, so a later process can offer the
	// changelog without a second network call.
	ReleaseURL string `json:"release_url"`
}

// UpdateCheckCachePath returns the file the update-check state lives in:
// `$XDG_CACHE_HOME/dflow/update-check.json` when XDG_CACHE_HOME is set and
// non-blank, otherwise `$HOME/.cache/dflow/update-check.json`.
//
// It resolves the variables itself instead of calling os.UserCacheDir because
// that helper picks a different root per platform (notably `~/Library/Caches`
// on macOS), and the issue's promise is one predictable, documentation-matching
// location that is trivially reproducible in tests.
//
// An environment with neither variable set has no writable home to fall back
// on, so the returned error names both variables rather than returning a
// relative path that would scatter cache files into the working directory.
func UpdateCheckCachePath() (string, error) {
	if cacheDir := strings.TrimSpace(lookupEnv(updateCacheDirEnv)); cacheDir != "" {
		return filepath.Join(cacheDir, updateCacheDirName, updateCacheFileName), nil
	}
	if home := strings.TrimSpace(lookupEnv(updateHomeEnv)); home != "" {
		return filepath.Join(home, ".cache", updateCacheDirName, updateCacheFileName), nil
	}
	return "", fmt.Errorf("could not locate the update-check cache: neither %s nor %s is set", updateCacheDirEnv, updateHomeEnv)
}

// ReadUpdateCheckState loads a previously recorded check.
//
// A missing file is `(nil, nil)` rather than an error: that is the first run on
// a clean machine, the expected state, and callers must treat it as "no
// information yet" instead of a failure. Every other problem — an unreadable
// file, or a body that is not valid JSON — is returned as an error, because a
// cached answer that cannot be trusted must not be silently treated as absent
// and overwritten without the caller knowing.
func ReadUpdateCheckState(path string) (*UpdateCheckState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("could not read the update-check cache at %s: %w", path, err)
	}

	var state UpdateCheckState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("could not decode the update-check cache at %s: %w", path, err)
	}
	return &state, nil
}

// WriteUpdateCheckState stores a check result, creating the parent directory if
// it does not exist yet.
//
// The write is atomic by construction: the payload is staged in a temporary
// file inside the destination directory and then renamed over the target. A
// crash, a full disk or a kill signal therefore leaves either the previous
// cache or the new one, never a half-written file that a later process would
// try to decode. Renaming within one directory is what keeps it atomic — a temp
// file in the system temp dir would cross filesystems and degrade to a copy.
//
// The caller owns the policy on failures: a startup notification is a courtesy,
// so a probe that cannot persist its result should log nothing and carry on.
func WriteUpdateCheckState(path string, state UpdateCheckState) error {
	// Marshal before touching the filesystem: a state that cannot be encoded
	// must not leave a freshly created cache directory behind.
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("could not encode the update-check state: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("could not create the update-check cache directory at %s: %w", dir, err)
	}

	temp, err := os.CreateTemp(dir, "."+updateCacheFileName+".tmp.*")
	if err != nil {
		return fmt.Errorf("could not create a temporary update-check cache file in %s: %w", dir, err)
	}
	tempPath := temp.Name()
	// The temp file is dot-prefixed and removed on every path that does not end
	// in the rename. After a successful rename this remove is a harmless
	// ENOENT, which is why it is not worth branching on.
	defer func() { _ = os.Remove(tempPath) }()

	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return fmt.Errorf("could not write the temporary update-check cache file at %s: %w", tempPath, err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("could not finish writing the temporary update-check cache file at %s: %w", tempPath, err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("could not move the update-check cache into place at %s: %w", path, err)
	}
	return nil
}

// IsUpdateCheckFresh reports whether a cached state is still within
// updateCheckTTL of now and can therefore answer "is there an update?" without
// another request.
//
// A nil state and one whose LastChecked is the zero time are both "never
// checked", never fresh: the zero time is what an absent or hand-written cache
// decodes to, and treating it as recent would suppress checking forever.
// The comparison is strict, so a state recorded exactly TTL ago is stale —
// "fresh for up to 24h" means the 24h boundary belongs to the next check.
func IsUpdateCheckFresh(state *UpdateCheckState, now time.Time) bool {
	if state == nil || state.LastChecked.IsZero() {
		return false
	}
	return now.Sub(state.LastChecked) < updateCheckTTL
}

// ShouldProbeUpdate reports whether a check should hit the network now.
//
// It answers only the cache question — "is the cached answer still usable?" —
// and deliberately knows nothing about the reasons a caller might still skip
// the probe (a development build with no release provenance, an explicit
// DFLOW_NO_UPDATE_CHECK opt-out, or a machine-readable run). Those are product
// rules the wiring layer owns, and folding them in here would hide them behind
// a function whose name promises a cache decision.
func ShouldProbeUpdate(state *UpdateCheckState, now time.Time) bool {
	return !IsUpdateCheckFresh(state, now)
}

// RecordUpdateCheck writes the state implied by an observed release: the
// release's normalized tag and page, stamped with now.
//
// A nil release is a no-op that returns nil. It is the honest encoding of "the
// probe produced no result", and letting it fall through to a write would
// record an empty version as if it were a real answer, which the next read
// would then compare against the installed binary.
func RecordUpdateCheck(path string, release *Release, now time.Time) error {
	if release == nil {
		return nil
	}
	return WriteUpdateCheckState(path, UpdateCheckState{
		LastChecked:   now,
		LatestVersion: release.Tag,
		ReleaseURL:    release.HTMLURL,
	})
}

// ResolveBaseURL returns the GitHub API base URL the probe should query:
// DFLOW_UPDATE_API_URL when it is set and non-blank, otherwise the public
// DefaultBaseURL.
//
// It reads the same variable `dflow update` honours, so pointing a user at a
// mirror or a test server with one environment variable redirects the
// notification too. Blank and whitespace-only values are ignored for the same
// reason the update command ignores them: an empty variable must not turn every
// lookup into a request to the empty string.
func ResolveBaseURL() string {
	if override := strings.TrimSpace(lookupEnv(updateBaseURLEnv)); override != "" {
		return override
	}
	return DefaultBaseURL
}

// ProbeLatestRelease asks GitHub for the newest published release on a bounded,
// short-timeout client suited to running behind a command.
//
// It builds on NewReleaseClientWithBaseURL rather than issuing its own request
// so error translation, tag normalization and the asset map stay in one place;
// the only thing this function changes is the timeout. baseURL is a parameter
// because that is how every other network entry point in this package is
// written, and a blank value falls back to ResolveBaseURL so a caller with no
// endpoint in mind cannot build a request against the empty string.
func ProbeLatestRelease(baseURL string) (*Release, error) {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = ResolveBaseURL()
	}
	client := NewReleaseClientWithBaseURL(baseURL, probeHTTPClient())
	return client.LatestRelease()
}

// probeHTTPClient builds the client the startup probe uses: the package's usual
// no-shared-instance policy, with probeHTTPTimeout instead of the interactive
// 30s budget. It is a separate constructor rather than a parameter to
// defaultHTTPClient so the two timeouts cannot be confused at a call site.
func probeHTTPClient() *http.Client {
	return &http.Client{Timeout: probeHTTPTimeout}
}
