package selfupdate

import (
	"strings"
	"testing"
)

// TestArchiveNameMatchesGoReleaser pins the platform-to-archive translation
// against the names `.goreleaser.yaml` actually produces and scripts/install.sh
// actually downloads. Every supported platform is exercised here because a
// test host can only ever be one of them.
func TestArchiveNameMatchesGoReleaser(t *testing.T) {
	cases := []struct {
		name   string
		goos   string
		goarch string
		want   string
	}{
		{name: "linux amd64", goos: "linux", goarch: "amd64", want: "dflow_Linux_x86_64.tar.gz"},
		{name: "linux arm64", goos: "linux", goarch: "arm64", want: "dflow_Linux_arm64.tar.gz"},
		{name: "darwin amd64", goos: "darwin", goarch: "amd64", want: "dflow_Darwin_x86_64.tar.gz"},
		{name: "darwin arm64", goos: "darwin", goarch: "arm64", want: "dflow_Darwin_arm64.tar.gz"},
		{name: "windows amd64", goos: "windows", goarch: "amd64", want: "dflow_Windows_x86_64.zip"},
		{name: "windows arm64", goos: "windows", goarch: "arm64", want: "dflow_Windows_arm64.zip"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ArchiveName(tc.goos, tc.goarch)
			if err != nil {
				t.Fatalf("ArchiveName(%q, %q) error = %v, want nil", tc.goos, tc.goarch, err)
			}
			if got != tc.want {
				t.Fatalf("ArchiveName(%q, %q) = %q, want %q", tc.goos, tc.goarch, got, tc.want)
			}
		})
	}
}

// TestArchiveNameRejectsUnsupportedPlatforms proves an unsupported machine gets
// an error that names the platform it asked about, instead of a lookup that
// silently returns an asset name no release will ever carry.
func TestArchiveNameRejectsUnsupportedPlatforms(t *testing.T) {
	cases := []struct {
		name   string
		goos   string
		goarch string
	}{
		{name: "unknown OS", goos: "plan9", goarch: "amd64"},
		{name: "unknown architecture on a supported OS", goos: "linux", goarch: "386"},
		{name: "unsupported arm variant", goos: "darwin", goarch: "arm"},
		{name: "unknown OS and architecture", goos: "freebsd", goarch: "riscv64"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ArchiveName(tc.goos, tc.goarch)
			if err == nil {
				t.Fatalf("ArchiveName(%q, %q) = %q, want an error", tc.goos, tc.goarch, got)
			}
			platform := tc.goos + "/" + tc.goarch
			if !strings.Contains(err.Error(), platform) {
				t.Fatalf("ArchiveName(%q, %q) error = %q, want it to name %q", tc.goos, tc.goarch, err.Error(), platform)
			}
		})
	}
}

// TestChecksumsNameStripsTheVPrefix pins the checksums asset name against the
// installer convention, including the tag-with-prefix form, so the updater and
// scripts/install.sh cannot disagree about which file to fetch.
func TestChecksumsNameStripsTheVPrefix(t *testing.T) {
	cases := []struct {
		name    string
		version string
		want    string
	}{
		{name: "tag with prefix", version: "v0.2.0", want: "dflow_0.2.0_checksums.txt"},
		{name: "bare version", version: "0.2.0", want: "dflow_0.2.0_checksums.txt"},
		{name: "surrounding whitespace", version: " v1.0.0 ", want: "dflow_1.0.0_checksums.txt"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ChecksumsName(tc.version); got != tc.want {
				t.Fatalf("ChecksumsName(%q) = %q, want %q", tc.version, got, tc.want)
			}
		})
	}
}
