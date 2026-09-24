package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// binaryContent stands in for the extracted executable. The bytes are a
// placeholder; what the assertions care about is that they arrive intact and
// executable.
const binaryContent = "ELF-ish bytes that pretend to be a dflow binary"

// archiveEntry is one fixture entry: a name and its contents. Fixtures are
// built inline in each test rather than committed, so a test carries its own
// archive and can never drift from a binary blob in the repository.
type archiveEntry struct {
	name    string
	content string
}

// buildTarGz writes entries into a fresh .tar.gz under a t.TempDir() and
// returns its path. Real tar and gzip writers are used so extraction runs
// against the same format GoReleaser produces.
func buildTarGz(t *testing.T, entries ...archiveEntry) string {
	t.Helper()

	archivePath := filepath.Join(t.TempDir(), "dflow_Linux_x86_64.tar.gz")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create tar.gz fixture: %v", err)
	}

	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)

	for _, entry := range entries {
		header := &tar.Header{
			Name:     entry.name,
			Mode:     0o644,
			Size:     int64(len(entry.content)),
			Typeflag: tar.TypeReg,
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("write tar header %q: %v", entry.name, err)
		}
		if _, err := tarWriter.Write([]byte(entry.content)); err != nil {
			t.Fatalf("write tar entry %q: %v", entry.name, err)
		}
	}

	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close tar.gz fixture: %v", err)
	}
	return archivePath
}

// buildZip is the zip counterpart of buildTarGz.
func buildZip(t *testing.T, entries ...archiveEntry) string {
	t.Helper()

	archivePath := filepath.Join(t.TempDir(), "dflow_Windows_x86_64.zip")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create zip fixture: %v", err)
	}

	zipWriter := zip.NewWriter(file)
	for _, entry := range entries {
		writer, err := zipWriter.Create(entry.name)
		if err != nil {
			t.Fatalf("create zip entry %q: %v", entry.name, err)
		}
		if _, err := writer.Write([]byte(entry.content)); err != nil {
			t.Fatalf("write zip entry %q: %v", entry.name, err)
		}
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close zip fixture: %v", err)
	}
	return archivePath
}

// assertExtractedBinary checks that destPath holds the binary and is
// executable, the two properties the installer guarantees with chmod 755.
func assertExtractedBinary(t *testing.T, destPath string) {
	t.Helper()

	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("read the extracted binary: %v", err)
	}
	if string(got) != binaryContent {
		t.Fatalf("extracted contents = %q, want %q", got, binaryContent)
	}

	info, err := os.Stat(destPath)
	if err != nil {
		t.Fatalf("stat the extracted binary: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o755 {
		t.Fatalf("extracted mode = %o, want 755", mode)
	}
}

// TestExtractBinaryFromTarGz covers the Linux and macOS format, including a
// nested entry: GoReleaser currently puts the binary at the archive root, but a
// future layout that nests it must still work because the lookup is by base
// name.
func TestExtractBinaryFromTarGz(t *testing.T) {
	cases := []struct {
		name   string
		goos   string
		entry  string
		extras []archiveEntry
	}{
		{
			name:  "linux root entry",
			goos:  "linux",
			entry: binaryBaseName,
		},
		{
			name:  "darwin nested entry",
			goos:  "darwin",
			entry: "dflow-0.2.0/" + binaryBaseName,
		},
		{
			name:   "an unrelated entry before the binary is skipped",
			goos:   "linux",
			entry:  binaryBaseName,
			extras: []archiveEntry{{name: "LICENSE", content: "MIT"}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			archivePath := buildTarGz(t, append(tc.extras, archiveEntry{name: tc.entry, content: binaryContent})...)
			dest := filepath.Join(t.TempDir(), binaryBaseName)

			if err := ExtractBinary(archivePath, dest, tc.goos); err != nil {
				t.Fatalf("ExtractBinary() error = %v, want nil", err)
			}
			assertExtractedBinary(t, dest)
		})
	}
}

// TestExtractBinaryFromZip covers the Windows format. The real Windows archive
// carries "dflow.exe" (scripts/install.ps1 looks for exactly that), and the bare
// "dflow" spelling is accepted too so the updater is not brittle about it.
func TestExtractBinaryFromZip(t *testing.T) {
	cases := []struct {
		name  string
		entry string
	}{
		{name: "dflow.exe as GoReleaser ships it", entry: binaryBaseName + ".exe"},
		{name: "bare dflow", entry: binaryBaseName},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			archivePath := buildZip(t,
				archiveEntry{name: "LICENSE", content: "MIT"},
				archiveEntry{name: tc.entry, content: binaryContent},
			)
			dest := filepath.Join(t.TempDir(), "dflow.exe")

			if err := ExtractBinary(archivePath, dest, "windows"); err != nil {
				t.Fatalf("ExtractBinary() error = %v, want nil", err)
			}
			assertExtractedBinary(t, dest)
		})
	}
}

// TestExtractBinaryRejectsZipSlip proves a traversal entry is refused before
// anything is written, and that the refused archive leaves no file behind the
// destination directory.
func TestExtractBinaryRejectsZipSlip(t *testing.T) {
	cases := []struct {
		name  string
		build func(*testing.T, ...archiveEntry) string
		goos  string
		entry string
	}{
		{
			name:  "tar.gz traversal",
			build: buildTarGz,
			goos:  "linux",
			entry: "../evil/" + binaryBaseName,
		},
		{
			name:  "zip traversal",
			build: buildZip,
			goos:  "windows",
			entry: "../../evil/" + binaryBaseName + ".exe",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			extractDir := filepath.Join(dir, "extract")
			if err := os.Mkdir(extractDir, 0o755); err != nil {
				t.Fatalf("create extraction dir: %v", err)
			}
			dest := filepath.Join(extractDir, "dflow")

			archivePath := tc.build(t, archiveEntry{name: tc.entry, content: binaryContent})
			err := ExtractBinary(archivePath, dest, tc.goos)
			if err == nil {
				t.Fatal("ExtractBinary() error = nil for a traversal entry, want an error")
			}
			if !strings.Contains(err.Error(), "outside the extraction directory") {
				t.Fatalf("ExtractBinary() error = %q, want it to mention the extraction directory", err.Error())
			}
			if _, statErr := os.Stat(filepath.Join(dir, "evil")); !os.IsNotExist(statErr) {
				t.Fatalf("traversal entry was written outside the extraction dir: stat error = %v", statErr)
			}
		})
	}
}

// TestExtractBinaryRejectsATraversalEntryThatIsNotTheBinary proves the guard
// covers every entry, not just the executable: an archive with a traversal
// LICENSE is refused even though a clean dflow follows it.
func TestExtractBinaryRejectsATraversalEntryThatIsNotTheBinary(t *testing.T) {
	archivePath := buildTarGz(t,
		archiveEntry{name: "../evil/LICENSE", content: "MIT"},
		archiveEntry{name: binaryBaseName, content: binaryContent},
	)
	dest := filepath.Join(t.TempDir(), binaryBaseName)

	err := ExtractBinary(archivePath, dest, "linux")
	if err == nil {
		t.Fatal("ExtractBinary() error = nil for a traversal entry, want an error")
	}
	if !strings.Contains(err.Error(), "outside the extraction directory") {
		t.Fatalf("ExtractBinary() error = %q, want it to mention the extraction directory", err.Error())
	}
}

// TestExtractBinaryRejectsAnArchiveWithoutTheBinary guards the missing-entry
// case, mirroring the installer's refusal to install an archive with no dflow.
func TestExtractBinaryRejectsAnArchiveWithoutTheBinary(t *testing.T) {
	cases := []struct {
		name  string
		build func(*testing.T, ...archiveEntry) string
		goos  string
	}{
		{name: "tar.gz without dflow", build: buildTarGz, goos: "linux"},
		{name: "zip without dflow", build: buildZip, goos: "windows"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			archivePath := tc.build(t, archiveEntry{name: "LICENSE", content: "MIT"})
			dest := filepath.Join(t.TempDir(), "dflow")

			err := ExtractBinary(archivePath, dest, tc.goos)
			if err == nil {
				t.Fatal("ExtractBinary() error = nil, want an error")
			}
			if !strings.Contains(err.Error(), "did not contain") {
				t.Fatalf("ExtractBinary() error = %q, want a missing-binary error", err.Error())
			}
		})
	}
}

// TestExtractBinaryRejectsUnsupportedPlatform pins the goos routing: an unknown
// platform fails with a message naming it instead of guessing at a format.
func TestExtractBinaryRejectsUnsupportedPlatform(t *testing.T) {
	archivePath := buildTarGz(t, archiveEntry{name: binaryBaseName, content: binaryContent})
	dest := filepath.Join(t.TempDir(), "dflow")

	err := ExtractBinary(archivePath, dest, "plan9")
	if err == nil {
		t.Fatal("ExtractBinary() error = nil for an unsupported platform, want an error")
	}
	if !strings.Contains(err.Error(), "plan9") {
		t.Fatalf("ExtractBinary() error = %q, want it to name the platform", err.Error())
	}
}
