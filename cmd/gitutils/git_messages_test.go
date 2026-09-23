package gitutils

import "testing"

// TestRemoteDeleteFailureKeepsBothVariantsByteIdentical pins the exact wording
// a failed `git push origin --delete` reports, because that wording is public:
// it reaches the terminal, and the end-to-end CLI assertions in cmd/tests only
// match substrings of it.
//
// Both variants are pinned whole rather than by substring, so a change in the
// composition that shares the remote-delete sentence between them cannot alter
// a single character the user sees without failing here. The diagnostics carry a
// path and a colon so a moved separator or a whitespace change is visible too.
func TestRemoteDeleteFailureKeepsBothVariantsByteIdentical(t *testing.T) {
	const branch = "feature/delete-me"
	const diagnostics = "error: failed to push some refs to '/srv/git/origin: mirror.git'"

	cases := []struct {
		name         string
		localExisted bool
		diagnostics  string
		want         string
	}{
		{
			name:         "local copy deleted, remote refused",
			localExisted: true,
			diagnostics:  diagnostics,
			want: "deleted local branch 'feature/delete-me' but failed to delete remote branch " +
				"'feature/delete-me': " + diagnostics,
		},
		{
			name:         "no local copy, remote refused",
			localExisted: false,
			diagnostics:  diagnostics,
			want:         "failed to delete remote branch 'feature/delete-me': " + diagnostics,
		},
		{
			name:         "local copy deleted, padded diagnostics are trimmed",
			localExisted: true,
			diagnostics:  "\n  " + diagnostics + " \n",
			want: "deleted local branch 'feature/delete-me' but failed to delete remote branch " +
				"'feature/delete-me': " + diagnostics,
		},
		{
			name:         "no local copy, padded diagnostics are trimmed",
			localExisted: false,
			diagnostics:  "\n  " + diagnostics + " \n",
			want:         "failed to delete remote branch 'feature/delete-me': " + diagnostics,
		},
		{
			name:         "empty diagnostics keep the separator",
			localExisted: true,
			diagnostics:  " \n ",
			want: "deleted local branch 'feature/delete-me' but failed to delete remote branch " +
				"'feature/delete-me': ",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := remoteDeleteFailure(branch, tc.localExisted, tc.diagnostics)
			if err == nil {
				t.Fatalf("remoteDeleteFailure returned no error")
			}
			if got := err.Error(); got != tc.want {
				t.Fatalf("remoteDeleteFailure(branch, %t, %q).Error()\n got %q\nwant %q",
					tc.localExisted, tc.diagnostics, got, tc.want)
			}
		})
	}
}
