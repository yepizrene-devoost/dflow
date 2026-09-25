package utils

import (
	"io"
	"os"
	"runtime/debug"
	"strings"
	"testing"
)

// fullRevision is a 40-character commit hash with a recognisable prefix, so an
// assertion can name the exact substring the abbreviated display must carry.
const fullRevision = "60d9e07a1b2c3d4e5f60718293a4b5c6d7e8f901"

// shortRevision is the seven-character abbreviation of fullRevision, the same
// width `git log --oneline` prints.
const shortRevision = "60d9e07"

// withBuildInfo substitutes the build metadata every VCS reader sees and
// restores the real reader afterwards.
//
// readBuildInfo is the only seam the version code reads, so a test never needs
// a real Git checkout, a rebuilt binary or a linker flag. resetVCSStamp drops
// the memoized stamp before and after the case, because the stamp is read once
// per process and would otherwise leak between cases.
func withBuildInfo(t *testing.T, ok bool, settings ...debug.BuildSetting) {
	t.Helper()

	original := readBuildInfo
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Settings: settings}, ok
	}
	resetVCSStamp()
	t.Cleanup(func() {
		readBuildInfo = original
		resetVCSStamp()
	})
}

// withVersionMarker replaces the channel/version marker for one test and
// restores the built-in default afterwards, so a case that calls SetVersion
// cannot leak into the next one.
func withVersionMarker(t *testing.T, marker string) {
	t.Helper()

	original := version
	SetVersion(marker)
	t.Cleanup(func() { SetVersion(original) })
}

// TestVersionDisplayReadsVCSStamp is the core contract: the marker alone when
// the binary carries no stamp, and marker plus abbreviated revision otherwise,
// with `-dirty` only when the stamp says the build tree was modified.
//
// The cases are deliberately ordered stamped -> dirty -> unstamped -> short and
// run in one process, which is the order-independence requirement: a cached
// stamp would leak from the first case into the rest.
func TestVersionDisplayReadsVCSStamp(t *testing.T) {
	cases := []struct {
		name         string
		ok           bool
		revision     string
		modified     string
		wantDisplay  string
		wantRevision string
		wantDirty    bool
		marker       string
	}{
		{
			name:         "stamped and clean",
			ok:           true,
			revision:     fullRevision,
			modified:     "false",
			wantDisplay:  "dev " + shortRevision,
			wantRevision: fullRevision,
			wantDirty:    false,
		},
		{
			name:         "stamped and dirty",
			ok:           true,
			revision:     fullRevision,
			modified:     "true",
			wantDisplay:  "dev " + shortRevision + "-dirty",
			wantRevision: fullRevision,
			wantDirty:    true,
		},
		{
			name:        "no build info at all",
			ok:          false,
			wantDisplay: "dev",
		},
		{
			name:        "stamp present without a revision",
			ok:          true,
			modified:    "true",
			wantDisplay: "dev",
		},
		{
			name:         "revision shorter than the abbreviation",
			ok:           true,
			revision:     "abc1234",
			modified:     "false",
			wantDisplay:  "dev abc1234",
			wantRevision: "abc1234",
		},

		// Release marker (v-prefixed) — the marker stands alone when the build is
		// clean and carries a stamp, because the revision is redundant for a
		// published release. A dirty build keeps its provenance.
		{
			name:         "release marker, clean build",
			ok:           true,
			revision:     fullRevision,
			modified:     "false",
			wantDisplay:  "v0.4.0",
			wantRevision: fullRevision,
			wantDirty:    false,
			marker:       "v0.4.0",
		},
		{
			name:         "release marker, dirty build",
			ok:           true,
			revision:     fullRevision,
			modified:     "true",
			wantDisplay:  "v0.4.0 " + shortRevision + "-dirty",
			wantRevision: fullRevision,
			wantDirty:    true,
			marker:       "v0.4.0",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var settings []debug.BuildSetting
			if tc.revision != "" {
				settings = append(settings, debug.BuildSetting{Key: "vcs.revision", Value: tc.revision})
			}
			if tc.modified != "" {
				settings = append(settings, debug.BuildSetting{Key: "vcs.modified", Value: tc.modified})
			}
			withBuildInfo(t, tc.ok, settings...)
			if tc.marker != "" {
				withVersionMarker(t, tc.marker)
			}

			if got := Revision(); got != tc.wantRevision {
				t.Fatalf("Revision() = %q, want %q", got, tc.wantRevision)
			}
			if got := RevisionDirty(); got != tc.wantDirty {
				t.Fatalf("RevisionDirty() = %t, want %t", got, tc.wantDirty)
			}
			if got := VersionDisplay(); got != tc.wantDisplay {
				t.Fatalf("VersionDisplay() = %q, want %q", got, tc.wantDisplay)
			}
			if got := GetVersion(); got != tc.wantDisplay {
				t.Fatalf("GetVersion() = %q, want %q", got, tc.wantDisplay)
			}
		})
	}
}

// TestRevisionShortAbbreviates pins the abbreviation rule: seven characters for
// anything longer, the stamp whole when it is already short, and no panic or
// filler for an empty stamp.
func TestRevisionShortAbbreviates(t *testing.T) {
	cases := []struct {
		name     string
		revision string
		want     string
	}{
		{name: "full length hash", revision: fullRevision, want: shortRevision},
		{name: "hash exactly at the abbreviation", revision: "abc1234", want: "abc1234"},
		{name: "hash shorter than the abbreviation", revision: "abc", want: "abc"},
		{name: "no stamp", revision: "", want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withBuildInfo(t, true, debug.BuildSetting{Key: "vcs.revision", Value: tc.revision})

			if got := RevisionShort(); got != tc.want {
				t.Fatalf("RevisionShort() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestVersionMarkerReportsTheChannelAlone guards the two halves apart: the
// machine-readable document reports the marker on its own, so SetVersion must
// change the marker without the stamp leaking into it.
func TestVersionMarkerReportsTheChannelAlone(t *testing.T) {
	withVersionMarker(t, "v0.2.0")
	withBuildInfo(t, true,
		debug.BuildSetting{Key: "vcs.revision", Value: fullRevision},
		debug.BuildSetting{Key: "vcs.modified", Value: "false"},
	)

	if got := VersionMarker(); got != "v0.2.0" {
		t.Fatalf("VersionMarker() = %q, want %q", got, "v0.2.0")
	}
	// A release marker on a clean build: the tag names the commit, so the
	// revision is redundant and the marker stands alone.
	if got := VersionDisplay(); got != "v0.2.0" {
		t.Fatalf("VersionDisplay() = %q, want %q", got, "v0.2.0")
	}

	// A release marker on a dirty build: provenance is information the marker
	// does not carry, so the revision appears.
	withBuildInfo(t, true,
		debug.BuildSetting{Key: "vcs.revision", Value: fullRevision},
		debug.BuildSetting{Key: "vcs.modified", Value: "true"},
	)
	if got := VersionDisplay(); got != "v0.2.0 "+shortRevision+"-dirty" {
		t.Fatalf("VersionDisplay() = %q, want %q", got, "v0.2.0 "+shortRevision+"-dirty")
	}

	// A snapshot marker carries no release provenance, so it must be passed
	// through as-is — never confused with a tagged release.
	withVersionMarker(t, "snapshot-abc1234")
	withBuildInfo(t, true,
		debug.BuildSetting{Key: "vcs.revision", Value: fullRevision},
		debug.BuildSetting{Key: "vcs.modified", Value: "false"},
	)

	if got := VersionMarker(); got != "snapshot-abc1234" {
		t.Fatalf("VersionMarker() = %q, want %q", got, "snapshot-abc1234")
	}
	if got := VersionDisplay(); got != "snapshot-abc1234 "+shortRevision {
		t.Fatalf("VersionDisplay() = %q, want %q", got, "snapshot-abc1234 "+shortRevision)
	}
}

// TestBuildInfoIsReReadAfterReset proves the memoized stamp can be refreshed:
// two different stamps read in the same process must not collapse into the
// first one, which is what makes the other tests deterministic and
// order-independent.
func TestBuildInfoIsReReadAfterReset(t *testing.T) {
	first := "1111111111111111111111111111111111111111"
	second := "2222222222222222222222222222222222222222"

	withBuildInfo(t, true, debug.BuildSetting{Key: "vcs.revision", Value: first})
	if got := Revision(); got != first {
		t.Fatalf("first Revision() = %q, want %q", got, first)
	}

	withBuildInfo(t, true, debug.BuildSetting{Key: "vcs.revision", Value: second})
	if got := Revision(); got != second {
		t.Fatalf("Revision() after re-reading build info = %q, want %q", got, second)
	}
}

// TestPrintBannerCarriesTheSameBuild guards the last entry point: the startup
// banner interpolates GetVersion, so it cannot report a different build than
// `dflow version` does.
func TestPrintBannerCarriesTheSameBuild(t *testing.T) {
	withBuildInfo(t, true,
		debug.BuildSetting{Key: "vcs.revision", Value: fullRevision},
		debug.BuildSetting{Key: "vcs.modified", Value: "true"},
	)

	output := captureStdout(t, PrintBanner)
	want := "dflow dev " + shortRevision + "-dirty"
	if !strings.Contains(output, want) {
		t.Fatalf("banner does not carry the same build as GetVersion (%q), want %q in:\n%s", GetVersion(), want, output)
	}
	if !strings.Contains(output, "Git branching made simple") {
		t.Fatalf("captured output is not the banner:\n%s", output)
	}
}

// captureStdout runs fn with os.Stdout redirected to a pipe and returns what it
// wrote. PrintBanner writes through fmt.Printf, so the redirection is the only
// way to observe it.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	original := os.Stdout
	os.Stdout = writer
	t.Cleanup(func() { os.Stdout = original })

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("close pipe writer: %v", err)
	}
	captured, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read captured banner: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("close pipe reader: %v", err)
	}
	return string(captured)
}
