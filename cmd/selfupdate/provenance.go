package selfupdate

import "strings"

// devMarker is the version marker a binary carries when it was built without
// the linker injecting the release version (a plain `go build`, or
// `go install`). It is the same default cmd/utils documents.
const devMarker = "dev"

// snapshotPrefix is the version prefix GoReleaser stamps on snapshot builds
// (`snapshot-<short commit>` in .goreleaser.yaml). A snapshot is a cut of the
// development branch, not a published release, so it has no release to compare
// against.
const snapshotPrefix = "snapshot"

// HasReleaseProvenance reports whether a binary's version marker identifies a
// published release whose version can be compared against the latest one.
//
// It is false for an empty marker and for the two markers a non-release build
// carries:
//
//   - "dev" — the linker never injected a version, so there is no version to
//     compare. Such a binary was almost certainly produced by `go build` or
//     installed by `go install`, which means the user (or the Go toolchain)
//     owns the file, not the installer. The caller must warn and continue
//     rather than silently replacing it.
//   - "snapshot-<commit>" — a GoReleaser snapshot, a development cut that was
//     never published as a release, so "latest release" is not its upgrade
//     path either.
//
// Only when this returns true is it meaningful for the caller to compare the
// marker against the latest release version and offer an update.
func HasReleaseProvenance(versionMarker string) bool {
	marker := strings.TrimSpace(versionMarker)
	if marker == "" || marker == devMarker {
		return false
	}
	return !strings.HasPrefix(marker, snapshotPrefix)
}
