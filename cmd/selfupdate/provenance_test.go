package selfupdate

import "testing"

// TestHasReleaseProvenance pins which version markers identify a published
// release. Only the release-shaped markers are comparable; the two build-local
// markers must make the caller warn and continue rather than replace the
// binary out from under whoever installed it.
func TestHasReleaseProvenance(t *testing.T) {
	cases := []struct {
		name   string
		marker string
		want   bool
	}{
		{name: "release version", marker: "0.2.0", want: true},
		{name: "larger release version", marker: "10.0.0", want: true},
		{name: "development build", marker: "dev", want: false},
		{name: "empty marker", marker: "", want: false},
		{name: "go releaser snapshot", marker: "snapshot-abc1234", want: false},
		{name: "whitespace around dev", marker: "  dev  ", want: false},
		{name: "whitespace around a release", marker: " 0.2.0 ", want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := HasReleaseProvenance(tc.marker); got != tc.want {
				t.Fatalf("HasReleaseProvenance(%q) = %t, want %t", tc.marker, got, tc.want)
			}
		})
	}
}
