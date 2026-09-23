// Package selfupdate is the network-and-arithmetic half of `dflow update`.
//
// It answers three questions and nothing else: which release is newest, which
// archive belongs on this machine, and does the downloaded bytes match the
// published checksum. It deliberately stops before touching the filesystem
// state a user cares about — no binary is staged, renamed or replaced here, and
// no command is registered. That split exists so the risky part of a
// self-update (overwriting a running executable) can be wired and reviewed on
// its own, while the parts that can be exhaustively unit-tested stay free of
// `os.Executable()`, process state and the real network.
//
// Every network entry point takes its base URL and `http.Client` from the
// caller, which is what lets the test suite point at an `httptest` server and
// never open a socket to GitHub.
//
// The asset and checksum names encoded here are not a new convention: they
// mirror `scripts/install.sh` and `.goreleaser.yaml` exactly, because a release
// that the installer can consume must be the same release the updater can.
package selfupdate
