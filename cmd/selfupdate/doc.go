// Package selfupdate is the machinery behind `dflow update`: it discovers the
// latest release, checks whether it is newer than the installed binary, then
// downloads, verifies, extracts and atomically replaces that binary.
//
// It is deliberately the whole engine and not the command. No cobra command and
// no flag parsing live here, because the risky part of a self-update
// (overwriting a running executable) deserves to be wired and reviewed on its
// own, and because everything except that wiring can then be unit-tested
// without `os.Executable()`, process state or the real network.
//
// Every network entry point takes its base URL and `http.Client` from the
// caller, and every platform decision (asset name, archive format, Windows
// rename-aside) takes its `goos` as a parameter instead of reading
// `runtime.GOOS`. That is what lets the test suite drive `httptest` servers and
// exercise the Windows path from any host, and it is why the suite never opens
// a socket to GitHub.
//
// The asset names, checksum format, staging pattern and failure wording encoded
// here are not a new convention: they mirror `scripts/install.sh`,
// `scripts/install.ps1`, `.goreleaser.yaml` and the version marker in
// `cmd/utils`, because a release the installer can consume must be the same
// release the updater can, and a binary that `go install` placed must not be
// silently replaced.
package selfupdate
