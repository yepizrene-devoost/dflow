package selfupdate

import "testing"

// TestCompareVersions covers the three tiers the doc comment promises: clean
// numeric triples (including the classic "is 0.10.0 newer than 0.9.9?" trap of
// string-ordered versions), missing components counted as zero, optional `v`
// prefixes, and the documented lexicographic fallback for input that is not a
// numeric triple.
func TestCompareVersions(t *testing.T) {
	cases := []struct {
		name string
		a    string
		b    string
		want int
	}{
		{name: "equal", a: "0.2.0", b: "0.2.0", want: 0},
		{name: "minor carries past a single digit", a: "0.10.0", b: "0.9.9", want: 1},
		{name: "major beats minor and patch", a: "1.0.0", b: "0.99.99", want: 1},
		{name: "older major", a: "0.99.99", b: "1.0.0", want: -1},
		{name: "v-prefix against no prefix is equal", a: "v1.2.3", b: "1.2.3", want: 0},
		{name: "v-prefix against an older bare version", a: "v1.2.3", b: "1.2.2", want: 1},
		{name: "missing patch counts as zero", a: "1.2", b: "1.2.0", want: 0},
		{name: "missing patch against a real patch", a: "1.2", b: "1.2.1", want: -1},
		{name: "missing minor and patch", a: "2", b: "2.0.0", want: 0},
		{name: "missing minor still loses to a real minor", a: "1", b: "1.1", want: -1},
		{name: "empty sorts oldest", a: "", b: "0.0.1", want: -1},
		{name: "empty on the other side", a: "0.0.1", b: "", want: 1},
		{name: "two empties are equal", a: "", b: "", want: 0},
		{name: "pre-release suffix falls back to strings", a: "1.0.0-rc1", b: "1.0.0", want: 1},
		{name: "non-numeric component falls back to strings", a: "nightly", b: "nightly", want: 0},
		{name: "non-numeric compares lexicographically (digits sort first)", a: "abc", b: "1.0.0", want: 1},
		{name: "too many components falls back to strings", a: "1.0.0.0", b: "1.0.0", want: 1},
		{name: "v-prefix is stripped before the fallback too", a: "v1.0.0-rc1", b: "1.0.0-rc2", want: -1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CompareVersions(tc.a, tc.b); got != tc.want {
				t.Fatalf("CompareVersions(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
			// The comparator is antisymmetric: if a > b is claimed, b < a must
			// hold too, which catches an asymmetric fallback branch.
			if got := CompareVersions(tc.b, tc.a); got != -tc.want {
				t.Fatalf("CompareVersions(%q, %q) = %d, want %d (antisymmetry)", tc.b, tc.a, got, -tc.want)
			}
		})
	}
}

// TestIsNewer pins the argument order of the convenience wrapper, since
// swapping its arguments would invert the entire update decision.
func TestIsNewer(t *testing.T) {
	cases := []struct {
		name      string
		candidate string
		current   string
		want      bool
	}{
		{name: "newer release", candidate: "0.3.0", current: "0.2.1", want: true},
		{name: "same release", candidate: "0.2.1", current: "0.2.1", want: false},
		{name: "older release", candidate: "0.2.0", current: "0.2.1", want: false},
		{name: "v-prefixed candidate against bare current", candidate: "v1.0.0", current: "0.9.0", want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsNewer(tc.candidate, tc.current); got != tc.want {
				t.Fatalf("IsNewer(%q, %q) = %t, want %t", tc.candidate, tc.current, got, tc.want)
			}
		})
	}
}
