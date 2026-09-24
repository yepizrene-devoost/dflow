package selfupdate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

// sha256HexLen is the length of a hex-encoded SHA-256 digest. It is a named
// constant so the "is this digest even well-formed?" check reads as intent
// rather than as a magic number.
const sha256HexLen = sha256.Size * 2

// VerifyChecksum checks a downloaded archive against the checksums file
// published alongside its release, mirroring the verification scripts/install.sh
// performs before it installs anything.
//
// archivePath is the downloaded archive, checksumsPath is the downloaded
// `dflow_<version>_checksums.txt`, and assetName is the archive's name as it
// appears in that file (the same name ArchiveName produces).
//
// Trust boundary: the checksums file is downloaded from the same release as
// the archive, so this check proves the archive matches what its publisher
// published. It reliably detects truncated, corrupted, or mismatched
// transfers, but it cannot detect a compromised release itself: matching
// bytes from a bad origin verify cleanly. The origin — GitHub release
// infrastructure for the yepizrene-devoost/dflow repository — is trusted the
// same way scripts/install.sh trusts it, and because this release flow
// publishes no signatures, this checksum match is the strongest integrity
// guarantee available to both installers.
//
// The comparison is case-insensitive because a digest is hex and some checksum
// tools emit upper case; a case difference is not a corrupted download. Every
// failure it returns is fatal to the update by design: an unverifiable archive
// is worse than no update, so the caller must never install one.
func VerifyChecksum(archivePath, checksumsPath, assetName string) error {
	contents, err := os.ReadFile(checksumsPath)
	if err != nil {
		return fmt.Errorf("could not read the checksums file %s: %w", checksumsPath, err)
	}

	expected, found := findChecksum(string(contents), assetName)
	if !found {
		return fmt.Errorf("the checksums file %s has no entry for %s: the release looks incomplete, so I will not trust the download", checksumsPath, assetName)
	}
	if !isSHA256Digest(expected) {
		return fmt.Errorf("the checksum entry for %s in %s is not a valid SHA-256 digest: %q", assetName, checksumsPath, expected)
	}

	actual, err := fileSHA256(archivePath)
	if err != nil {
		return err
	}

	if !strings.EqualFold(expected, actual) {
		// The expected/actual pair is spelled out because that is the only way
		// a user can tell a truncated download from a swapped archive.
		return fmt.Errorf("checksum mismatch for %s:\n  expected: %s\n  actual:   %s\nrefusing to trust a download that does not match the published checksum", assetName, expected, actual)
	}
	return nil
}

// findChecksum locates the digest for one asset in a checksums file.
//
// The format is the standard sha256sum one — `<digest>  <filename>`, two spaces —
// which is also what scripts/install.sh parses with awk. Splitting on any run
// of whitespace accepts both the two-space form and older single-space output,
// while a line that does not yield exactly two fields carries no usable entry
// and is skipped: the caller then gets the explicit "no entry" error rather
// than a comparison against a half-parsed digest.
func findChecksum(contents, assetName string) (string, bool) {
	for _, line := range strings.Split(contents, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		if fields[1] == assetName {
			return fields[0], true
		}
	}
	return "", false
}

// isSHA256Digest reports whether value looks like a hex-encoded SHA-256 digest.
// A published file can carry a corrupted line, and "expected" text that cannot
// be a digest should fail as malformed rather than as a confusing mismatch.
func isSHA256Digest(value string) bool {
	if len(value) != sha256HexLen {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

// fileSHA256 returns the lower-case hex SHA-256 of the file at path. It returns
// a wrapped error when the archive cannot be read, because a missing or
// unreadable download is a real outcome, not a programming mistake.
func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("could not open the downloaded archive %s: %w", path, err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("could not read the downloaded archive %s: %w", path, err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
