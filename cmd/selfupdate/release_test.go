package selfupdate

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// releasesPath is the API path the client must hit. Asserting on it (rather
// than only on the parsed result) is what pins the endpoint contract: a typo in
// the path would still parse a canned response in a looser test.
const releasesPath = "/repos/yepizrene-devoost/dflow/releases/latest"

// newTestReleaseClient wires a ReleaseClient to a throwaway httptest server so
// no test ever opens a socket to GitHub.
func newTestReleaseClient(t *testing.T, handler http.HandlerFunc) *ReleaseClient {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return NewReleaseClientWithBaseURL(server.URL, server.Client())
}

// TestLatestReleaseParsesTheGitHubResponse is the happy path: a real release
// payload is decoded into the fields the updater needs, unknown fields are
// ignored, assets are looked up by name, and the request targets the expected
// endpoint with the pinned media type.
func TestLatestReleaseParsesTheGitHubResponse(t *testing.T) {
	var requestedPath, requestedAccept string

	client := newTestReleaseClient(t, func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		requestedAccept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "application/json")
		// "unknown_future_field" is part of the contract: GitHub grows its
		// payload over time, and a parser that choked on a new key would break
		// the updater the day GitHub ships one.
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tag_name":             "v0.2.0",
			"html_url":             "https://github.com/yepizrene-devoost/dflow/releases/tag/v0.2.0",
			"unknown_future_field": true,
			"assets": []map[string]string{
				{"name": "dflow_Linux_x86_64.tar.gz", "browser_download_url": "https://example.test/linux.tar.gz"},
				{"name": "dflow_0.2.0_checksums.txt", "browser_download_url": "https://example.test/checksums.txt"},
			},
		})
	})

	release, err := client.LatestRelease()
	if err != nil {
		t.Fatalf("LatestRelease() error = %v, want nil", err)
	}

	if requestedPath != releasesPath {
		t.Fatalf("requested path = %q, want %q", requestedPath, releasesPath)
	}
	if requestedAccept != "application/vnd.github+json" {
		t.Fatalf("Accept header = %q, want %q", requestedAccept, "application/vnd.github+json")
	}
	if release.Tag != "v0.2.0" {
		t.Fatalf("Tag = %q, want %q", release.Tag, "v0.2.0")
	}
	if release.Version != "0.2.0" {
		t.Fatalf("Version = %q, want %q", release.Version, "0.2.0")
	}
	if release.HTMLURL != "https://github.com/yepizrene-devoost/dflow/releases/tag/v0.2.0" {
		t.Fatalf("HTMLURL = %q", release.HTMLURL)
	}

	if url, ok := release.AssetURL("dflow_Linux_x86_64.tar.gz"); !ok || url != "https://example.test/linux.tar.gz" {
		t.Fatalf("AssetURL(linux) = (%q, %t), want the linux URL and true", url, ok)
	}
	if url, ok := release.AssetURL("dflow_Windows_x86_64.zip"); ok || url != "" {
		t.Fatalf("AssetURL(missing) = (%q, %t), want (\"\", false)", url, ok)
	}
}

// TestNewReleaseNormalizesTags pins the tag shape every downstream name and
// comparison depends on: the tag gains a leading `v` when GitHub omits it, and
// the version is always the tag without one.
func TestNewReleaseNormalizesTags(t *testing.T) {
	cases := []struct {
		name        string
		tagName     string
		wantTag     string
		wantVersion string
	}{
		{name: "already prefixed", tagName: "v0.2.0", wantTag: "v0.2.0", wantVersion: "0.2.0"},
		{name: "missing the prefix", tagName: "0.3.0", wantTag: "v0.3.0", wantVersion: "0.3.0"},
		{name: "surrounding whitespace", tagName: "  v1.0.0  ", wantTag: "v1.0.0", wantVersion: "1.0.0"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			release, err := newRelease(releaseResponse{TagName: tc.tagName})
			if err != nil {
				t.Fatalf("newRelease(%q) error = %v, want nil", tc.tagName, err)
			}
			if release.Tag != tc.wantTag {
				t.Fatalf("Tag = %q, want %q", release.Tag, tc.wantTag)
			}
			if release.Version != tc.wantVersion {
				t.Fatalf("Version = %q, want %q", release.Version, tc.wantVersion)
			}
		})
	}
}

// TestLatestReleaseDecodesTheReleaseBody pins the new body decode: the rendered
// changelog GitHub returns must reach Release.Body, and its surrounding
// whitespace (GitHub commonly wraps it in blank lines) must not leak into the
// summary built from it.
func TestLatestReleaseDecodesTheReleaseBody(t *testing.T) {
	const changelog = "## What's new\n\n- fixed a bug"

	client := newTestReleaseClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tag_name": "v0.3.0",
			"body":     "\n\n" + changelog + "\n\n",
		})
	})

	release, err := client.LatestRelease()
	if err != nil {
		t.Fatalf("LatestRelease() error = %v, want nil", err)
	}
	if release.Body != changelog {
		t.Fatalf("Body = %q, want %q", release.Body, changelog)
	}
}

// TestNewReleaseNormalizesTheBody keeps the body normalization honest at the
// boundary the summarizer relies on: the whole body's surrounding whitespace is
// trimmed once, and a body GitHub omits (or sends blank) stays empty rather
// than turning into a placeholder.
func TestNewReleaseNormalizesTheBody(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{name: "trims surrounding whitespace", body: "\n\n## Notes\n- one\n\n", want: "## Notes\n- one"},
		{name: "empty stays empty", body: "", want: ""},
		{name: "whitespace-only becomes empty", body: "\n   \t\n", want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			release, err := newRelease(releaseResponse{TagName: "v0.3.0", Body: tc.body})
			if err != nil {
				t.Fatalf("newRelease(body %q) error = %v, want nil", tc.body, err)
			}
			if release.Body != tc.want {
				t.Fatalf("Body = %q, want %q", release.Body, tc.want)
			}
		})
	}
}

// TestNewReleaseRejectsAnEmptyTag guards the one malformed payload worth
// failing on: without a tag there is no version to compare, so a silent success
// would surface later as a nonsensical "0.0.0" update.
func TestNewReleaseRejectsAnEmptyTag(t *testing.T) {
	if _, err := newRelease(releaseResponse{TagName: "   "}); err == nil {
		t.Fatal("newRelease(empty tag) error = nil, want an error")
	}
}

// TestLatestReleaseHTTPFailures maps each documented GitHub failure mode to the
// user-oriented message the updater promises: a missing release, a rate limit,
// and an API outage must be distinguishable without the user reading HTTP
// status codes.
func TestLatestReleaseHTTPFailures(t *testing.T) {
	cases := []struct {
		name        string
		status      int
		wantInError []string
	}{
		{
			name:        "404 means no release yet",
			status:      http.StatusNotFound,
			wantInError: []string{"GitHub API", "404"},
		},
		{
			name:        "403 hints at rate limiting",
			status:      http.StatusForbidden,
			wantInError: []string{"GitHub API", "403", "rate limit"},
		},
		{
			name:        "429 also hints at rate limiting",
			status:      http.StatusTooManyRequests,
			wantInError: []string{"GitHub API", "429", "rate limit"},
		},
		{
			name:        "5xx blames GitHub",
			status:      http.StatusInternalServerError,
			wantInError: []string{"GitHub API", "500", "GitHub is having trouble"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := newTestReleaseClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
			})

			_, err := client.LatestRelease()
			if err == nil {
				t.Fatalf("LatestRelease() error = nil for HTTP %d, want an error", tc.status)
			}
			for _, want := range tc.wantInError {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("LatestRelease() error = %q, want it to contain %q", err.Error(), want)
				}
			}
		})
	}
}

// TestLatestReleaseRejectsMalformedJSON proves a 200 with an unparseable body
// fails instead of yielding a zero-value release that would look like version
// 0.0.0 to the rest of the updater.
func TestLatestReleaseRejectsMalformedJSON(t *testing.T) {
	client := newTestReleaseClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{not json"))
	})

	_, err := client.LatestRelease()
	if err == nil {
		t.Fatal("LatestRelease() error = nil for malformed JSON, want an error")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Fatalf("LatestRelease() error = %q, want it to mention decoding", err.Error())
	}
}

// TestNewReleaseClientDefaultsToGitHub pins the production default so a later
// refactor cannot quietly point the updater at a test host or an empty URL.
func TestNewReleaseClientDefaultsToGitHub(t *testing.T) {
	client := NewReleaseClient()
	if client.baseURL != DefaultBaseURL {
		t.Fatalf("baseURL = %q, want %q", client.baseURL, DefaultBaseURL)
	}
	if client.httpClient == nil || client.httpClient.Timeout <= 0 {
		t.Fatalf("httpClient = %+v, want a client with a positive timeout", client.httpClient)
	}
}
