package selfupdate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// writeFile writes a file with the given contents and returns its path.
func writeFile(t *testing.T, path, contents string) string {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

// assertNoStageFiles proves InstallBinary left no ".dflow.tmp.*" staging file
// behind next to the target, on either the success or the failure path.
func assertNoStageFiles(t *testing.T, dir string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir %s: %v", dir, err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "."+binaryBaseName+".tmp.") {
			t.Fatalf("staging file %s survived in %s", entry.Name(), dir)
		}
	}
}

// assertFileContents checks the file at path holds want.
func assertFileContents(t *testing.T, path, want string) {
	t.Helper()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(got) != want {
		t.Fatalf("contents of %s = %q, want %q", path, got, want)
	}
}

// TestEnsureWritableAcceptsAWritableBinary is the happy path: a regular binary
// in a writable directory passes every check.
func TestEnsureWritableAcceptsAWritableBinary(t *testing.T) {
	target := writeFile(t, filepath.Join(t.TempDir(), binaryBaseName), "old binary")

	if err := EnsureWritable(target); err != nil {
		t.Fatalf("EnsureWritable() error = %v, want nil", err)
	}
}

// TestEnsureWritableRejectsAMissingBinary guards the "nothing to update" case:
// the error must point at the installer rather than read as a permission bug.
func TestEnsureWritableRejectsAMissingBinary(t *testing.T) {
	target := filepath.Join(t.TempDir(), binaryBaseName)

	err := EnsureWritable(target)
	if err == nil {
		t.Fatal("EnsureWritable() error = nil for a missing binary, want an error")
	}
	for _, want := range []string{target, "installer script"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("EnsureWritable() error = %q, want it to contain %q", err.Error(), want)
		}
	}
}

// TestEnsureWritableRejectsAReadOnlyBinary exercises the file half of the
// check: a binary the process cannot open for writing fails with a message that
// names the file and the remedy.
func TestEnsureWritableRejectsAReadOnlyBinary(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root, which bypasses permission bits")
	}

	target := writeFile(t, filepath.Join(t.TempDir(), binaryBaseName), "old binary")
	if err := os.Chmod(target, 0o444); err != nil {
		t.Fatalf("chmod target: %v", err)
	}

	err := EnsureWritable(target)
	if err == nil {
		t.Fatal("EnsureWritable() error = nil for a read-only binary, want an error")
	}
	for _, want := range []string{target, "binary", "not writable", "installer script"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("EnsureWritable() error = %q, want it to contain %q", err.Error(), want)
		}
	}
}

// TestEnsureWritableRejectsAReadOnlyDirectory exercises the directory half: the
// binary itself is writable, but the swap would need a new file in a directory
// that refuses one. This is the shape of a system-owned install location.
func TestEnsureWritableRejectsAReadOnlyDirectory(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root, which bypasses permission bits")
	}

	dir := t.TempDir()
	target := writeFile(t, filepath.Join(dir, binaryBaseName), "old binary")
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatalf("chmod dir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	err := EnsureWritable(target)
	if err == nil {
		t.Fatal("EnsureWritable() error = nil for a read-only directory, want an error")
	}
	for _, want := range []string{dir, "directory", "not writable", "installer script"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("EnsureWritable() error = %q, want it to contain %q", err.Error(), want)
		}
	}
}

// TestInstallBinaryOnUnixSwapsTheBinary covers the atomic rename path: the
// target ends up holding the new bytes and no staging file survives.
func TestInstallBinaryOnUnixSwapsTheBinary(t *testing.T) {
	for _, goos := range []string{"linux", "darwin"} {
		t.Run(goos, func(t *testing.T) {
			dir := t.TempDir()
			target := writeFile(t, filepath.Join(dir, binaryBaseName), "old binary")
			newBinary := writeFile(t, filepath.Join(dir, "downloaded"), "new binary")

			if err := InstallBinary(newBinary, target, goos); err != nil {
				t.Fatalf("InstallBinary() error = %v, want nil", err)
			}
			assertFileContents(t, target, "new binary")
			assertNoStageFiles(t, dir)
		})
	}
}

// TestInstallBinaryOnWindowsLeavesTheAside covers the Windows sequence on a
// host where the rename can actually succeed: a regular file stands in for the
// running executable. The previous binary is left aside as "<target>.old"
// because a running Windows process keeps it locked, and the stale aside from a
// previous update is replaced rather than accumulating.
func TestInstallBinaryOnWindowsLeavesTheAside(t *testing.T) {
	dir := t.TempDir()
	target := writeFile(t, filepath.Join(dir, binaryBaseName+".exe"), "old binary")
	newBinary := writeFile(t, filepath.Join(dir, "downloaded"), "new binary")
	// A leftover from an earlier update must be cleared before the rename.
	writeFile(t, target+".old", "stale aside")

	if err := InstallBinary(newBinary, target, "windows"); err != nil {
		t.Fatalf("InstallBinary() error = %v, want nil", err)
	}

	assertFileContents(t, target, "new binary")
	assertFileContents(t, target+".old", "old binary")
	assertNoStageFiles(t, dir)
}

// TestInstallBinaryRemovesTheStageOnFailure pins the failure cleanup: when the
// new binary cannot be read, the update fails without touching the target and
// without leaving a staging file behind.
func TestInstallBinaryRemovesTheStageOnFailure(t *testing.T) {
	dir := t.TempDir()
	target := writeFile(t, filepath.Join(dir, binaryBaseName), "old binary")
	missing := filepath.Join(dir, "not-downloaded")

	err := InstallBinary(missing, target, "linux")
	if err == nil {
		t.Fatal("InstallBinary() error = nil for a missing source, want an error")
	}
	assertFileContents(t, target, "old binary")
	assertNoStageFiles(t, dir)
}

// TestRunningExecutableProbeClassifiesErrors pins the one probe failure that
// must never block an update: a write-open of the running dflow binary fails
// with ETXTBSY on Unix kernels regardless of the permission bits, and the
// rename strategy in InstallBinary works fine in that situation. The
// end-to-end shape (executing a process and then probing its binary) is
// environment-dependent and slow, so the classification is tested directly
// against real syscall errors; the read-only-binary test above already proves
// that other failures still reach the not-writable path.
func TestRunningExecutableProbeClassifiesErrors(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"running executable", syscall.ETXTBSY, true},
		{"running executable wrapped", fmt.Errorf("open: %w", syscall.ETXTBSY), true},
		{"permission denied", os.ErrPermission, false},
		{"not exist", os.ErrNotExist, false},
		{"nil error", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := runningExecutableProbe(tc.err); got != tc.want {
				t.Fatalf("runningExecutableProbe(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
