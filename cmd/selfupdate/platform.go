package selfupdate

import (
	"fmt"
	"strings"
)

// Archive naming follows `.goreleaser.yaml` exactly:
//
//	dflow_<Os>_<Arch>.tar.gz   (Linux, macOS)
//	dflow_<Os>_<Arch>.zip      (Windows)
//
// where <Os> is the title-cased Go OS name and <Arch> rewrites amd64 to
// x86_64. These strings are not cosmetic: a mismatch produces a 404 on a
// release that exists, which is the least diagnosable failure an updater can
// have. Keeping the translation in one place lets the tests pin every platform,
// including the ones the test host can never be.
//
// The mapping is a switch rather than a package-level map on purpose — package
// maps are mutable global state, and this package commits to none.

// ArchiveName returns the GoReleaser archive file name for the given platform,
// or an error naming the platform when dflow publishes no build for it.
//
// goos and goarch are parameters rather than a direct `runtime.GOOS` read so
// the whole matrix is testable from any host; production callers pass the
// runtime values.
func ArchiveName(goos, goarch string) (string, error) {
	osToken, archToken, extension, err := archivePlatform(goos, goarch)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("dflow_%s_%s%s", osToken, archToken, extension), nil
}

// ChecksumsName returns the name of the checksums asset for a release version,
// mirroring scripts/install.sh: `dflow_<version>_checksums.txt`, where the
// version carries no leading `v`.
//
// A caller may pass the tag in either form ("v0.2.0" or "0.2.0"); the prefix is
// stripped here so the two spellings cannot produce two different asset names.
func ChecksumsName(version string) string {
	return fmt.Sprintf("dflow_%s_checksums.txt", strings.TrimPrefix(strings.TrimSpace(version), "v"))
}

// archivePlatform resolves the OS token, architecture token and file extension
// for a platform. The errors distinguish an unknown OS from an unknown
// architecture so a user on an unsupported machine learns which half is
// unsupported, not merely that something is.
func archivePlatform(goos, goarch string) (osToken, archToken, extension string, err error) {
	switch goos {
	case "linux":
		osToken = "Linux"
	case "darwin":
		osToken = "Darwin"
	case "windows":
		osToken = "Windows"
		extension = ".zip"
	default:
		return "", "", "", fmt.Errorf("unsupported platform %s/%s: dflow publishes Linux, macOS and Windows builds", goos, goarch)
	}

	switch goarch {
	case "amd64":
		archToken = "x86_64"
	case "arm64":
		archToken = "arm64"
	default:
		return "", "", "", fmt.Errorf("unsupported platform %s/%s: dflow publishes x86_64 and arm64 builds", goos, goarch)
	}

	if extension == "" {
		extension = ".tar.gz"
	}
	return osToken, archToken, extension, nil
}
