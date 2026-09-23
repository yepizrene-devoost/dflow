package selfupdate

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// archiveContents is the payload every checksum case hashes. The exact bytes do
// not matter; what matters is that the test computes the expected digest
// independently of the production helper, so a shared bug cannot hide.
const archiveContents = "pretend this is a gzipped dflow binary"

// testAssetName is the archive name the fixtures publish a digest for. It is
// the Linux amd64 name ArchiveName produces, so the checksum tests speak the
// same asset vocabulary as the updater.
const testAssetName = "dflow_Linux_x86_64.tar.gz"

// checksumFixture is one on-disk archive plus the checksums file that is
// supposed to describe it. Both live in a t.TempDir(), so a case never depends
// on or mutates another one.
type checksumFixture struct {
	archivePath   string
	checksumsPath string
	assetName     string
	digest        string
}

// newChecksumFixture writes the archive plus a checksums file with the given
// contents. The digest is computed here with crypto/sha256 rather than with the
// function under test.
func newChecksumFixture(t *testing.T, checksumsContents string) checksumFixture {
	t.Helper()

	dir := t.TempDir()
	archivePath := filepath.Join(dir, testAssetName)
	if err := os.WriteFile(archivePath, []byte(archiveContents), 0o600); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	checksumsPath := filepath.Join(dir, "dflow_0.2.0_checksums.txt")
	if err := os.WriteFile(checksumsPath, []byte(checksumsContents), 0o600); err != nil {
		t.Fatalf("write checksums: %v", err)
	}

	return checksumFixture{
		archivePath:   archivePath,
		checksumsPath: checksumsPath,
		assetName:     testAssetName,
		digest:        archiveDigest(),
	}
}

// archiveDigest is the hex SHA-256 of archiveContents, computed with the
// standard library so the test never trusts the code under test.
func archiveDigest() string {
	sum := sha256.Sum256([]byte(archiveContents))
	return hex.EncodeToString(sum[:])
}

// fixtureDigestLine builds a valid two-space sha256sum line for archiveContents
// under a caller-chosen asset name.
func fixtureDigestLine(assetName string) string {
	return archiveDigest() + "  " + assetName + "\n"
}

// TestVerifyChecksumAcceptsAMatchingDigest is the happy path, including the
// upper-case digest some checksum tools emit. A case difference must not fail,
// because the bytes are identical.
func TestVerifyChecksumAcceptsAMatchingDigest(t *testing.T) {
	cases := []struct {
		name              string
		checksumsContents string
	}{
		{name: "lower-case digest", checksumsContents: fixtureDigestLine(testAssetName)},
		{name: "upper-case digest", checksumsContents: strings.ToUpper(archiveDigest()) + "  " + testAssetName + "\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newChecksumFixture(t, tc.checksumsContents)
			if err := VerifyChecksum(fixture.archivePath, fixture.checksumsPath, fixture.assetName); err != nil {
				t.Fatalf("VerifyChecksum() error = %v, want nil", err)
			}
		})
	}
}

// TestVerifyChecksumReportsAMismatch proves the failure names both digests:
// without the expected/actual pair a user cannot tell a truncated download from
// a swapped archive.
func TestVerifyChecksumReportsAMismatch(t *testing.T) {
	wrongDigest := strings.Repeat("ab", sha256.Size)
	fixture := newChecksumFixture(t, wrongDigest+"  "+testAssetName+"\n")

	err := VerifyChecksum(fixture.archivePath, fixture.checksumsPath, fixture.assetName)
	if err == nil {
		t.Fatal("VerifyChecksum() error = nil for a mismatched digest, want an error")
	}
	for _, want := range []string{"checksum mismatch", wrongDigest, fixture.digest} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("VerifyChecksum() error = %q, want it to contain %q", err.Error(), want)
		}
	}
}

// TestVerifyChecksumRequiresTheAsset guards the missing-entry case: a checksums
// file that describes other platforms is not a pass, because installing an
// unlisted archive is exactly the supply-chain hole verification exists to
// close.
func TestVerifyChecksumRequiresTheAsset(t *testing.T) {
	fixture := newChecksumFixture(t, fixtureDigestLine("dflow_Darwin_arm64.tar.gz"))

	err := VerifyChecksum(fixture.archivePath, fixture.checksumsPath, fixture.assetName)
	if err == nil {
		t.Fatal("VerifyChecksum() error = nil for a missing entry, want an error")
	}
	for _, want := range []string{fixture.assetName, "no entry"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("VerifyChecksum() error = %q, want it to contain %q", err.Error(), want)
		}
	}
}

// TestVerifyChecksumSkipsMalformedLinesAndRejectsMalformedEntries covers the
// two ways a checksums file can be broken: a line that is not a digest/name
// pair carries no entry and must be skipped so a later valid line still wins,
// while a well-formed line whose digest is not SHA-256 hex is a corrupted
// publication and must fail loudly.
func TestVerifyChecksumSkipsMalformedLinesAndRejectsMalformedEntries(t *testing.T) {
	t.Run("malformed line before a valid one is skipped", func(t *testing.T) {
		fixture := newChecksumFixture(t, "this line has no filename\n"+fixtureDigestLine(testAssetName))

		if err := VerifyChecksum(fixture.archivePath, fixture.checksumsPath, fixture.assetName); err != nil {
			t.Fatalf("VerifyChecksum() error = %v, want nil", err)
		}
	})

	t.Run("matched entry with a non-hex digest fails", func(t *testing.T) {
		fixture := newChecksumFixture(t, "not-a-real-digest  "+testAssetName+"\n")

		err := VerifyChecksum(fixture.archivePath, fixture.checksumsPath, fixture.assetName)
		if err == nil {
			t.Fatal("VerifyChecksum() error = nil for a malformed digest, want an error")
		}
		if !strings.Contains(err.Error(), "valid SHA-256 digest") {
			t.Fatalf("VerifyChecksum() error = %q, want it to mention a valid SHA-256 digest", err.Error())
		}
	})

	t.Run("only malformed lines is a missing entry", func(t *testing.T) {
		fixture := newChecksumFixture(t, "dangling-digest-without-filename\n")

		err := VerifyChecksum(fixture.archivePath, fixture.checksumsPath, fixture.assetName)
		if err == nil {
			t.Fatal("VerifyChecksum() error = nil, want a missing-entry error")
		}
		if !strings.Contains(err.Error(), "no entry") {
			t.Fatalf("VerifyChecksum() error = %q, want a missing-entry error", err.Error())
		}
	})
}

// TestVerifyChecksumSurfacesReadFailures guards the two I/O paths: an absent
// checksums file and an absent archive must both produce an error rather than a
// panic or a false pass.
func TestVerifyChecksumSurfacesReadFailures(t *testing.T) {
	fixture := newChecksumFixture(t, fixtureDigestLine(testAssetName))

	t.Run("missing checksums file", func(t *testing.T) {
		err := VerifyChecksum(fixture.archivePath, filepath.Join(t.TempDir(), "absent.txt"), fixture.assetName)
		if err == nil {
			t.Fatal("VerifyChecksum() error = nil for a missing checksums file, want an error")
		}
		if !strings.Contains(err.Error(), "could not read the checksums file") {
			t.Fatalf("VerifyChecksum() error = %q, want a read error", err.Error())
		}
	})

	t.Run("missing archive", func(t *testing.T) {
		err := VerifyChecksum(filepath.Join(t.TempDir(), "absent.tar.gz"), fixture.checksumsPath, fixture.assetName)
		if err == nil {
			t.Fatal("VerifyChecksum() error = nil for a missing archive, want an error")
		}
		if !strings.Contains(err.Error(), "could not open the downloaded archive") {
			t.Fatalf("VerifyChecksum() error = %q, want an open error", err.Error())
		}
	})
}
