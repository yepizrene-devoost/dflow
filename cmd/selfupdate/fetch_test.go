package selfupdate

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// fetchedBody stands in for a release archive. Its exact content does not
// matter; what matters is that the bytes on disk equal the bytes served.
const fetchedBody = "pretend this is a multi-megabyte release archive"

// TestDownloadFileStreamsTheBodyToDisk is the happy path: the served bytes land
// in the destination file intact.
func TestDownloadFileStreamsTheBodyToDisk(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, fetchedBody)
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "archive.tar.gz")
	if err := DownloadFile(server.URL, dest, server.Client()); err != nil {
		t.Fatalf("DownloadFile() error = %v, want nil", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read the downloaded file: %v", err)
	}
	if string(got) != fetchedBody {
		t.Fatalf("downloaded contents = %q, want %q", got, fetchedBody)
	}
}

// TestDownloadFileRejectsNon200 pins the failure contract: a non-200 response
// is an error that names both the URL and the status, so a user can tell a
// missing release (404) from a GitHub outage (5xx) without guessing.
func TestDownloadFileRejectsNon200(t *testing.T) {
	cases := []struct {
		name   string
		status int
	}{
		{name: "not found", status: http.StatusNotFound},
		{name: "server error", status: http.StatusInternalServerError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
			}))
			defer server.Close()

			dest := filepath.Join(t.TempDir(), "archive.tar.gz")
			err := DownloadFile(server.URL, dest, server.Client())
			if err == nil {
				t.Fatalf("DownloadFile() error = nil for HTTP %d, want an error", tc.status)
			}
			for _, want := range []string{server.URL, strconv.Itoa(tc.status)} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("DownloadFile() error = %q, want it to contain %q", err.Error(), want)
				}
			}
			if _, statErr := os.Stat(dest); !os.IsNotExist(statErr) {
				t.Fatalf("destination exists after a failed download: stat error = %v, want not-exist", statErr)
			}
		})
	}
}

// TestDownloadFileRemovesAPartialDownload proves a transfer that dies midway
// leaves nothing behind: a truncated file that survives would be handed to the
// checksum step as if it were a complete download.
//
// The server writes a few bytes and then aborts the connection with
// http.ErrAbortHandler, which forces the client's read to fail rather than
// reach a clean EOF.
func TestDownloadFileRemovesAPartialDownload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "partial")
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		panic(http.ErrAbortHandler)
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "archive.tar.gz")
	err := DownloadFile(server.URL, dest, server.Client())
	if err == nil {
		t.Fatal("DownloadFile() error = nil for an aborted transfer, want an error")
	}

	if _, statErr := os.Stat(dest); !os.IsNotExist(statErr) {
		t.Fatalf("partial download survives at %s: stat error = %v, want not-exist", dest, statErr)
	}
}
