package selfupdate

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// defaultDownloadTimeout bounds one asset download. A release archive is a few
// megabytes, so a minute of silence means the connection is dead, not slow.
// Without a timeout a stalled transfer would hang `dflow update` forever, and a
// user who reached for an update wants a prompt failure far more than a
// complete one.
const defaultDownloadTimeout = 2 * time.Minute

// DownloadFile streams the HTTP response at url into the file at destPath.
//
// The body is copied straight to disk instead of buffered in memory. Release
// archives are multiple megabytes, and there is no reason for a self-update to
// hold one in RAM only to write it out again.
//
// client is injectable so tests can point at an httptest server and never open
// a socket to GitHub. A nil client falls back to one carrying
// defaultDownloadTimeout, which keeps a caller from accidentally constructing a
// client that can hang forever.
//
// There is deliberately no size limit. The URL is only ever a release asset
// named by the release metadata (see Release.AssetURL), never arbitrary user
// input, and the download is discarded unless its published SHA-256 checksum
// matches (VerifyChecksum). A byte cap would therefore guard against a threat
// the checksum step already rejects, at the cost of rejecting a legitimately
// large archive. A future caller that feeds this an untrusted URL must impose
// its own limit before calling.
//
// A non-200 response is an error naming the URL and the status, and any
// partially written destination is removed so a failed download can never be
// mistaken for a complete one.
func DownloadFile(url, destPath string, client *http.Client) error {
	if client == nil {
		client = &http.Client{Timeout: defaultDownloadTimeout}
	}

	response, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("could not download %s: %w", url, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("could not download %s: the server returned HTTP %d %s", url, response.StatusCode, http.StatusText(response.StatusCode))
	}

	return streamToFile(destPath, response.Body)
}

// streamToFile writes reader into destPath and removes the file again on any
// failure, so a caller can never observe a truncated download as a success. The
// destination is written only after it is created, which means an error while
// creating it leaves nothing behind to clean up.
func streamToFile(destPath string, reader io.Reader) (err error) {
	file, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("could not create the download destination %s: %w", destPath, err)
	}

	complete := false
	defer func() {
		closeErr := file.Close()
		if !complete {
			_ = os.Remove(destPath)
			return
		}
		if closeErr != nil {
			err = fmt.Errorf("could not finish writing the download to %s: %w", destPath, closeErr)
		}
	}()

	if _, err := io.Copy(file, reader); err != nil {
		return fmt.Errorf("could not write the download to %s: %w", destPath, err)
	}

	complete = true
	return nil
}
