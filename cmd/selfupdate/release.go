package selfupdate

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// DefaultBaseURL is the public GitHub REST API root the updater queries unless
// a caller injects another one. Tests point this at an `httptest` server; the
// production path uses the same host the installer documents.
const DefaultBaseURL = "https://api.github.com"

// releaseRepository is the GitHub slug dflow publishes its releases from. It is
// a constant rather than a parameter because the updater only ever updates
// dflow: letting a caller point it at another repository would turn a bug into
// a way to download an arbitrary archive.
const releaseRepository = "yepizrene-devoost/dflow"

// defaultHTTPTimeout bounds one release lookup. Without a timeout a stalled
// connection would hang `dflow update` forever, and a user who reached for a
// check wants a prompt answer far more than a complete one.
const defaultHTTPTimeout = 30 * time.Second

// Release is the subset of a GitHub release the updater needs: its tag, its
// human-facing page and its downloadable assets.
//
// The zero value is not a valid release — build one through the client so the
// tag is normalized and the asset map is populated.
type Release struct {
	// Tag is the release tag with a leading `v`, e.g. "v0.2.0". Normalizing on
	// read means every downstream comparison and asset name sees one shape.
	Tag string
	// Version is the tag without the leading `v`, e.g. "0.2.0". It is the token
	// in the checksums asset name.
	Version string
	// HTMLURL is the GitHub release page, shown to the user so they can read
	// the changelog before or after updating.
	HTMLURL string

	// assets maps an asset name to its browser_download_url. A map rather than
	// a slice because every consumer looks an asset up by name; see AssetURL.
	assets map[string]string
}

// AssetURL returns the download URL of the named release asset and whether the
// release actually carries it.
//
// The second return is the load-bearing part: a release missing its platform
// archive is a real (and common) failure, and callers must distinguish "no such
// asset" from "asset with an empty URL" instead of downloading an empty string.
func (r Release) AssetURL(name string) (string, bool) {
	url, ok := r.assets[name]
	return url, ok
}

// LatestRelease is the parsed payload of the GitHub "latest release" endpoint.
// Each field maps to one JSON key; unknown fields are ignored because GitHub
// grows its payload over time and a new field must never break an old binary.
type releaseResponse struct {
	TagName string          `json:"tag_name"`
	HTMLURL string          `json:"html_url"`
	Assets  []assetResponse `json:"assets"`
}

// assetResponse is one entry of a release's asset list.
type assetResponse struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// defaultHTTPClient builds the client every production path shares: no
// package-level instance, because this package commits to no global mutable
// state, and a fresh client per constructor call costs nothing.
func defaultHTTPClient() *http.Client {
	return &http.Client{Timeout: defaultHTTPTimeout}
}

// ReleaseClient queries GitHub for dflow's latest release. Its base URL and
// `http.Client` are injected rather than hard-coded so tests can drive it
// against an `httptest` server without touching the real network.
type ReleaseClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewReleaseClient returns a client for the public GitHub API with the shared
// default timeout. It is the production constructor; use
// NewReleaseClientWithBaseURL when the endpoint must be substituted.
func NewReleaseClient() *ReleaseClient {
	return NewReleaseClientWithBaseURL(DefaultBaseURL, defaultHTTPClient())
}

// NewReleaseClientWithBaseURL returns a client that talks to baseURL over
// httpClient. A trailing slash on baseURL is tolerated so a caller can write
// either "https://api.github.com" or "https://api.github.com/".
//
// A nil httpClient falls back to the shared default timeout-bearing client,
// which keeps a caller from accidentally constructing a client that can hang
// forever.
func NewReleaseClientWithBaseURL(baseURL string, httpClient *http.Client) *ReleaseClient {
	if httpClient == nil {
		httpClient = defaultHTTPClient()
	}
	return &ReleaseClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

// defaultHTTPClient builds the one client shape every production path uses:
// no shared instance, because an http.Client is safe for concurrent use but a
// package-level one would be mutable global state, which this package commits
// to none of. The timeout bounds one release lookup: without it a stalled
// connection would hang `dflow update` forever, and a user who reached for a
// check wants a prompt answer far more than a complete one.

// LatestRelease fetches and parses the newest published release.
//
// The endpoint is unauthenticated on purpose: the installer already documents
// this policy, dflow's releases are public, and requiring a token would turn a
// one-command update into a credential setup. The trade-off is the per-IP rate
// limit, which releaseStatusError translates into a message a user can act on
// instead of a bare status code.
func (c *ReleaseClient) LatestRelease() (*Release, error) {
	endpoint := fmt.Sprintf("%s/repos/%s/releases/latest", c.baseURL, releaseRepository)

	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("could not build the GitHub API request: %w", err)
	}
	// Pin the media type instead of relying on the account default, so the
	// payload shape stays what this parser was written against.
	request.Header.Set("Accept", "application/vnd.github+json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("could not reach the GitHub API at %s: %w", endpoint, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, releaseStatusError(response.StatusCode, endpoint)
	}

	var payload releaseResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("could not decode the GitHub release response from %s: %w", endpoint, err)
	}

	return newRelease(payload)
}

// newRelease validates and normalizes a decoded payload into a Release. An
// empty tag is the one malformed shape worth rejecting loudly: without it there
// is no version to compare and no checksums asset name to build, so pretending
// the response was usable would only push the failure somewhere less obvious.
func newRelease(payload releaseResponse) (*Release, error) {
	tag := strings.TrimSpace(payload.TagName)
	if tag == "" {
		return nil, fmt.Errorf("the GitHub release response did not include a tag name")
	}
	tag = normalizeTag(tag)

	assets := make(map[string]string, len(payload.Assets))
	for _, asset := range payload.Assets {
		// An asset without a name or a download URL is not addressable; skip it
		// rather than register a key that resolves to an empty URL.
		if asset.Name == "" || asset.BrowserDownloadURL == "" {
			continue
		}
		assets[asset.Name] = asset.BrowserDownloadURL
	}

	return &Release{
		Tag:     tag,
		Version: strings.TrimPrefix(tag, "v"),
		HTMLURL: payload.HTMLURL,
		assets:  assets,
	}, nil
}

// normalizeTag gives a release tag its canonical leading `v`, mirroring the
// installer's normalize_tag so both tools derive identical asset URLs from the
// same release.
func normalizeTag(tag string) string {
	if strings.HasPrefix(tag, "v") {
		return tag
	}
	return "v" + tag
}

// releaseStatusError turns a non-200 GitHub response into a message aimed at
// the user rather than the developer: each branch names the likely cause and
// what to do about it, because "HTTP 403" on its own reads like a bug in dflow.
func releaseStatusError(status int, endpoint string) error {
	switch {
	case status == http.StatusNotFound:
		return fmt.Errorf("the GitHub API returned 404 Not Found for %s: no release is published for %s yet", endpoint, releaseRepository)
	case status == http.StatusForbidden || status == http.StatusTooManyRequests:
		return fmt.Errorf("the GitHub API returned HTTP %d for %s: the unauthenticated rate limit (60 requests per hour per IP) was probably exceeded, so wait a few minutes and try again", status, endpoint)
	case status >= http.StatusInternalServerError:
		return fmt.Errorf("the GitHub API returned HTTP %d for %s: GitHub is having trouble on its side, so try again later", status, endpoint)
	default:
		return fmt.Errorf("the GitHub API returned HTTP %d for %s", status, endpoint)
	}
}
