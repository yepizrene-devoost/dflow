package utils

import (
	"runtime/debug"
	"strings"
	"sync"
)

// revisionAbbrevLen is how many leading characters of the commit hash the
// human-facing version string shows. Seven is what `git log --oneline` prints,
// so a user can copy the token straight back into a Git command.
const revisionAbbrevLen = 7

// vcsStamp holds the VCS metadata the Go toolchain embeds in a binary built
// from a Git checkout. revision is the full commit hash the sources had at
// build time; modified reports whether that tree carried uncommitted changes.
//
// A binary built outside a Git checkout (from a release archive, or through
// `go install module@version`) embeds no stamp, which leaves revision empty.
// Every reader must treat an empty revision as "this build cannot name its
// commit" and fall back to the version marker alone: a missing second token is
// honest, an empty one reads like a defect.
type vcsStamp struct {
	revision string
	modified bool
}

// readBuildInfo is the source of build metadata. It is a variable so tests can
// substitute a stamp the running test binary does not carry; production reads
// the stamp the linker embedded, which is exactly why no build flag is needed.
var readBuildInfo = debug.ReadBuildInfo

// vcsStampOnce guards the one read per process, and vcsStampValue memoizes its
// result.
var (
	vcsStampOnce  sync.Once
	vcsStampValue vcsStamp
)

// currentVCSStamp reads the embedded stamp once per process: build metadata
// cannot change while the binary runs, and the version string is rendered from
// several entry points.
//
// The memo is a sync.Once plus a value rather than sync.OnceValue so tests can
// discard it with resetVCSStamp and observe a substituted readBuildInfo. The
// production behavior is identical: read once, reuse forever.
func currentVCSStamp() vcsStamp {
	vcsStampOnce.Do(func() {
		vcsStampValue = parseVCSStamp(readBuildInfo())
	})
	return vcsStampValue
}

// resetVCSStamp forgets the memoized stamp so the next read runs readBuildInfo
// again. Production code never calls it; tests use it between cases that
// substitute the reader, which keeps them deterministic and order-independent.
func resetVCSStamp() {
	vcsStampOnce = sync.Once{}
	vcsStampValue = vcsStamp{}
}

// parseVCSStamp extracts the Git stamp from the build metadata the toolchain
// embedded. A binary built outside a Git checkout reports no build info and
// yields the zero stamp, which leaves revision empty.
func parseVCSStamp(info *debug.BuildInfo, ok bool) vcsStamp {
	if !ok {
		return vcsStamp{}
	}

	stamp := vcsStamp{}
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			stamp.revision = setting.Value
		case "vcs.modified":
			stamp.modified = setting.Value == "true"
		}
	}
	return stamp
}

// Revision returns the full commit hash this binary was built from, or the
// empty string when it carries no VCS stamp. It is the script-facing answer
// behind `dflow version --revision`.
func Revision() string {
	return currentVCSStamp().revision
}

// RevisionDirty reports whether the build tree carried uncommitted changes.
// Without a stamp there is nothing to judge, so the answer is false.
func RevisionDirty() bool {
	stamp := currentVCSStamp()
	return stamp.revision != "" && stamp.modified
}

// RevisionShort returns the abbreviated commit hash used in the human-facing
// version string, or the empty string when there is no stamp. A stamp shorter
// than the abbreviation is returned whole rather than truncated further.
func RevisionShort() string {
	revision := Revision()
	if len(revision) <= revisionAbbrevLen {
		return revision
	}
	return revision[:revisionAbbrevLen]
}

// isReleaseMarker reports whether the version marker identifies a published
// release — the same predicate HasReleaseProvenance uses, duplicated here to
// avoid an import cycle between cmd/utils and cmd/selfupdate. A release marker
// is neither empty, nor "dev", nor a GoReleaser snapshot prefix (`snapshot-*`).
func isReleaseMarker(marker string) bool {
	m := strings.TrimSpace(marker)
	if m == "" || m == "dev" {
		return false
	}
	return !strings.HasPrefix(m, "snapshot")
}

// VersionDisplay composes the full human-facing version: the channel/version
// marker (`dev` for a development install, or the release version injected at
// build time) followed by the commit that binary was installed from, marked
// `-dirty` when the build tree had uncommitted changes. With no VCS stamp the
// marker stands alone, which is the behavior such a binary has always shown.
//
// For a release marker (the tag form, e.g. `v0.4.0`), the revision is shown
// only when it carries information the marker does not: a dirty build keeps its
// provenance, while a clean build stands alone because the tag already names the
// commit.
//
// A snapshot marker (`snapshot-<short>`) behaves like `dev`: the marker is not
// a published release, so the revision is always appended to preserve provenance.
//
// GetVersion is the single caller of this function and every entry point goes
// through GetVersion, so `version`, `ver`, the root `--version`/`-V` flag and
// the banner cannot drift apart.
func VersionDisplay() string {
	stamp := currentVCSStamp()
	if stamp.revision == "" {
		return version
	}

	// A release marker on a clean build: the tag names the commit, so the
	// revision is redundant. A dirty build keeps its provenance.
	if isReleaseMarker(version) && !stamp.modified {
		return version
	}

	display := version + " " + RevisionShort()
	if stamp.modified {
		display += "-dirty"
	}
	return display
}
