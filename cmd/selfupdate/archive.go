package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// binaryBaseName is the file name GoReleaser gives the dflow executable on
// every platform except Windows. On Windows the Go toolchain appends ".exe",
// and scripts/install.ps1 looks for exactly "dflow.exe", so the updater must
// accept that name there too or it would report a perfectly good release as
// missing its binary.
const binaryBaseName = "dflow"

// binaryFileMode is the mode every installed dflow binary carries: executable
// by anyone who can read it, exactly the 755 scripts/install.sh chmods its
// staged file to.
const binaryFileMode os.FileMode = 0o755

// ExtractBinary copies the dflow executable out of a release archive into
// destPath, creating the destination with mode 0755.
//
// The archive format is chosen from goos rather than by sniffing the bytes
// (".tar.gz" for Linux and macOS, ".zip" for Windows, matching
// .goreleaser.yaml's `format_overrides`). Routing on the platform keeps the
// decision explicit and testable: a Windows release is a zip even if the caller
// saved it under another name, and a corrupted download should be reported as a
// corrupted download, not as an unrecognized archive.
//
// Only the executable is taken; the LICENSE, README and CHANGELOG entries the
// archive also carries are ignored. The executable is found by its base name so
// a release that nests it in a directory still works, even though GoReleaser
// currently places it at the archive root. An archive without it fails, mirroring
// the installer's "the archive did not contain a 'dflow' binary" refusal.
//
// Every entry name is validated before it is used: one that would resolve
// outside the destination directory (a "zip slip" path traversal) is rejected
// outright rather than written. A destination that already exists is
// overwritten; this write is not atomic, so a caller that must never leave a
// broken binary behind should extract to a staging path and hand it to
// InstallBinary.
func ExtractBinary(archivePath, destPath, goos string) error {
	switch goos {
	case "windows":
		return extractBinaryFromZip(archivePath, destPath, goos)
	case "linux", "darwin":
		return extractBinaryFromTarGz(archivePath, destPath, goos)
	default:
		return fmt.Errorf("cannot extract a dflow binary for unsupported platform %q: dflow publishes Linux, macOS and Windows builds", goos)
	}
}

// extractBinaryFromTarGz walks a gzip-compressed tar archive and extracts the
// first entry that is the dflow executable. The reader is left at the matched
// entry so its bytes can be streamed into the destination without buffering the
// whole archive.
func extractBinaryFromTarGz(archivePath, destPath, goos string) error {
	archive, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("could not open the archive %s: %w", archivePath, err)
	}
	defer archive.Close()

	gzipReader, err := gzip.NewReader(archive)
	if err != nil {
		return fmt.Errorf("could not read %s as a gzip archive: %w", archivePath, err)
	}
	defer gzipReader.Close()

	reader := tar.NewReader(gzipReader)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("could not read %s as a tar archive: %w", archivePath, err)
		}

		// Every entry name is validated before it is even considered: an archive
		// that carries a traversal entry is refused outright, whatever that entry
		// is, so a malicious release cannot smuggle a path past the binary check.
		if _, err := safeEntryPath(filepath.Dir(destPath), header.Name); err != nil {
			return err
		}

		// Directories, symlinks and every other special entry are skipped: only
		// a regular file can be the binary we install.
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			continue
		}
		if !isBinaryEntry(header.Name, goos) {
			continue
		}
		return writeBinary(destPath, reader)
	}

	return missingBinaryError(archivePath)
}

// extractBinaryFromZip is the Windows counterpart of extractBinaryFromTarGz.
func extractBinaryFromZip(archivePath, destPath, goos string) error {
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("could not open %s as a zip archive: %w", archivePath, err)
	}
	defer archive.Close()

	for _, entry := range archive.File {
		if _, err := safeEntryPath(filepath.Dir(destPath), entry.Name); err != nil {
			return err
		}
		if entry.FileInfo().IsDir() {
			continue
		}
		if !isBinaryEntry(entry.Name, goos) {
			continue
		}

		body, err := entry.Open()
		if err != nil {
			return fmt.Errorf("could not read the %q entry from %s: %w", entry.Name, archivePath, err)
		}
		defer body.Close()

		return writeBinary(destPath, body)
	}

	return missingBinaryError(archivePath)
}

// isBinaryEntry reports whether an archive entry is the dflow executable.
// Archive entry names are always slash-separated, so path.Base (not
// filepath.Base) is the right splitter on every host.
func isBinaryEntry(name, goos string) bool {
	base := path.Base(name)
	if base == binaryBaseName {
		return true
	}
	// Windows releases carry "dflow.exe"; see scripts/install.ps1.
	return goos == "windows" && base == binaryBaseName+".exe"
}

// missingBinaryError is the refusal for an archive that carries no dflow
// executable. It mirrors the installer's message because a user hitting it may
// have already run that script and knows its wording.
func missingBinaryError(archivePath string) error {
	return fmt.Errorf("the archive %s did not contain a %q binary: the release looks incomplete", archivePath, binaryBaseName)
}

// safeEntryPath resolves an archive entry name inside dir and rejects any name
// that would land outside it. This is the "zip slip" guard: without it an
// archive entry named "../../etc/cron.d/x" could write anywhere the updater can.
//
// The name is cleaned with path because archive entries are slash-separated on
// every platform, then joined with filepath for the host filesystem. An
// absolute name, a leading ".." component, or a resolved path whose relative
// form climbs out of dir all fail with the same clear refusal.
func safeEntryPath(dir, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("refusing to extract: the archive contains an entry with an empty name")
	}

	cleaned := path.Clean(name)
	if path.IsAbs(cleaned) || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", unsafeEntryError(name)
	}

	resolved := filepath.Join(dir, filepath.FromSlash(cleaned))
	relative, err := filepath.Rel(dir, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", unsafeEntryError(name)
	}
	return resolved, nil
}

// unsafeEntryError names the offending entry so a user or a reviewer can see
// exactly which path triggered the refusal.
func unsafeEntryError(name string) error {
	return fmt.Errorf("refusing to extract %q: the archive entry points outside the extraction directory", name)
}

// writeBinary streams an archive entry into destPath with the executable mode.
// The mode is re-applied after the write because OpenFile's mode is masked by
// the process umask, and a binary that is not executable is worse than no
// update at all.
func writeBinary(destPath string, entry io.Reader) (err error) {
	file, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, binaryFileMode)
	if err != nil {
		return fmt.Errorf("could not create the extracted binary at %s: %w", destPath, err)
	}
	defer func() {
		if closeErr := file.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("could not finish writing the extracted binary at %s: %w", destPath, closeErr)
		}
	}()

	if _, err := io.Copy(file, entry); err != nil {
		return fmt.Errorf("could not write the extracted binary to %s: %w", destPath, err)
	}
	if err := os.Chmod(destPath, binaryFileMode); err != nil {
		return fmt.Errorf("could not mark the extracted binary at %s as executable: %w", destPath, err)
	}
	return nil
}
