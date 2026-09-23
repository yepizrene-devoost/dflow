package selfupdate

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

// EnsureWritable checks that the dflow binary at targetPath can be replaced,
// and returns an error a user can act on when it cannot.
//
// Three things have to hold for an update to succeed, and each fails for a
// different reason, so each is checked separately:
//
//  1. The binary exists and is a regular file. A missing target is not a
//     write-permission problem, it is a "there is nothing to update" problem.
//  2. The binary itself is openable for writing. The open uses O_WRONLY with no
//     O_TRUNC on purpose: this is an access probe, and a failed check must never
//     damage a working binary. One failure is not a permission problem: on
//     Linux and other Unix kernels, any write-open of an executable a process
//     is currently running fails with ETXTBSY. `dflow update` always probes
//     its own running binary, and the rename strategy in InstallBinary never
//     needs the file open for writing, so that signature passes this check —
//     see runningExecutableProbe.
//  3. The parent directory accepts a new file, which is probed by creating and
//     removing a temporary file there. Replacement happens by renaming a staged
//     file into that directory, so a writable binary on a read-only or full
//     filesystem would still fail the swap.
//
// A read-only system path such as /usr/local/bin without sudo fails step 2 or 3
// with an actionable message naming the path and the remedy, rather than a bare
// "permission denied" that reads like a bug in dflow.
//
// Note for callers: this function accepts the ETXTBSY signature described
// above, so probing the running binary does not block the update on Unix.
func EnsureWritable(targetPath string) error {
	info, err := os.Stat(targetPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("there is no dflow binary at %s to update: install it first with the installer script (scripts/install.sh), or point the install location at an existing binary", targetPath)
		}
		return fmt.Errorf("could not inspect the dflow binary at %s: %w", targetPath, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory, not a dflow binary: choose an install location that holds the dflow executable", targetPath)
	}

	handle, err := os.OpenFile(targetPath, os.O_WRONLY, 0)
	if err != nil {
		// ETXTBSY is the "this file is a running executable" signature, not a
		// permission problem, and it is the expected outcome when dflow probes
		// itself: the swap below replaces the file by rename, which needs no
		// write-open. Every other failure is a real access problem.
		if !runningExecutableProbe(err) {
			return notWritableError(targetPath, "binary", err)
		}
	} else if closeErr := handle.Close(); closeErr != nil {
		return fmt.Errorf("could not close the dflow binary at %s after checking it: %w", targetPath, closeErr)
	}

	dir := filepath.Dir(targetPath)
	probe, err := os.CreateTemp(dir, "."+binaryBaseName+".writetest.*")
	if err != nil {
		return notWritableError(dir, "directory", err)
	}
	probePath := probe.Name()
	_ = probe.Close()
	if removeErr := os.Remove(probePath); removeErr != nil {
		return fmt.Errorf("could not remove the temporary file %s after checking that %s is writable: %w", probePath, dir, removeErr)
	}
	return nil
}

// runningExecutableProbe reports whether err is the signature of probing a
// currently-running executable for write access: a write-open of an executable
// a process is running returns ETXTBSY on Unix kernels regardless of the
// permission bits. Since `dflow update` probes the very binary it runs from
// and replaces it by rename, this signature means "all good, the swap will
// work" rather than "the binary cannot be replaced".
func runningExecutableProbe(err error) bool {
	return errors.Is(err, syscall.ETXTBSY)
}

// notWritableError turns a failed write access probe into a message aimed at
// the user: it names the path that is not writable, says whether it is the
// binary or its directory, and offers the two real remedies.
func notWritableError(path, kind string, cause error) error {
	return fmt.Errorf("cannot update dflow: the %s at %s is not writable (%v). Reinstall dflow with the installer script (scripts/install.sh), or install it in a directory you own, such as ~/.local/bin", kind, path, cause)
}

// InstallBinary swaps newBinaryPath into targetPath, replacing whatever binary
// is already there.
//
// The new binary is staged as a dot-file next to the target (the same pattern
// scripts/install.sh uses) and chmod'ed to 0755 before the swap, so the landing
// binary is executable at the instant it appears and a failure leaves the old
// binary untouched. The stage file is removed on every failure path.
//
// goos selects the swap strategy and is a parameter rather than a
// `runtime.GOOS` read so the Windows path is exercisable from any test host:
//
//   - Everywhere else: one os.Rename(stage, target), which atomically replaces
//     the target even while it is running.
//   - Windows: a running .exe cannot be replaced in place, so the running target
//     is first renamed aside to `target + ".old"`, then the stage is renamed into
//     place. The aside file is expected to remain: a running Windows process
//     keeps it locked until that process exits, and failing the update over a
//     leftover file would be a far worse outcome. A stale aside from a previous
//     update is cleared (best effort) before the rename so leftovers never
//     accumulate or block the next update.
func InstallBinary(newBinaryPath, targetPath, goos string) error {
	dir := filepath.Dir(targetPath)

	stage, err := os.CreateTemp(dir, "."+binaryBaseName+".tmp.*")
	if err != nil {
		return notWritableError(dir, "directory", err)
	}
	stagePath := stage.Name()

	installed := false
	defer func() {
		if !installed {
			_ = os.Remove(stagePath)
		}
	}()

	if err := copyIntoStage(stage, newBinaryPath); err != nil {
		_ = stage.Close()
		return err
	}
	// Chmod before the rename so the binary is executable the moment it lands,
	// never for a window in between.
	if err := stage.Chmod(binaryFileMode); err != nil {
		_ = stage.Close()
		return fmt.Errorf("could not mark the staged dflow binary %s as executable: %w", stagePath, err)
	}
	if err := stage.Close(); err != nil {
		return fmt.Errorf("could not finish staging the new dflow binary at %s: %w", stagePath, err)
	}

	if goos == "windows" {
		if err := swapOnWindows(stagePath, targetPath); err != nil {
			return err
		}
	} else if err := os.Rename(stagePath, targetPath); err != nil {
		return fmt.Errorf("could not replace the dflow binary at %s with the staged copy: %w", targetPath, err)
	}

	installed = true
	return nil
}

// copyIntoStage streams the new binary into the already-open stage file. No
// size limit is imposed: the source is a file the updater just extracted and
// verified from a trusted release archive.
func copyIntoStage(stage *os.File, newBinaryPath string) error {
	source, err := os.Open(newBinaryPath)
	if err != nil {
		return fmt.Errorf("could not open the new dflow binary %s: %w", newBinaryPath, err)
	}
	defer source.Close()

	if _, err := io.Copy(stage, source); err != nil {
		return fmt.Errorf("could not stage the new dflow binary at %s: %w", stage.Name(), err)
	}
	return nil
}

// swapOnWindows performs the rename-aside sequence Windows needs. It never
// leaves the machine without a binary: if the second rename fails, the original
// is moved back before the error is returned.
func swapOnWindows(stagePath, targetPath string) error {
	oldPath := targetPath + ".old"

	// A leftover aside from a previous update must go first: Windows' os.Rename
	// refuses to overwrite an existing file, and a stale .old would block the
	// rename below. Best effort — if it is still locked, the rename reports it.
	_ = os.Remove(oldPath)

	if err := os.Rename(targetPath, oldPath); err != nil {
		return fmt.Errorf("could not move the running dflow binary %s aside to %s: %w", targetPath, oldPath, err)
	}

	if err := os.Rename(stagePath, targetPath); err != nil {
		// Put the previous binary back so a failed update never leaves the user
		// with no dflow at all.
		if restoreErr := os.Rename(oldPath, targetPath); restoreErr != nil {
			return fmt.Errorf("could not install the new dflow binary at %s (%v), and could not restore the previous one from %s: %w", targetPath, err, oldPath, restoreErr)
		}
		return fmt.Errorf("could not install the new dflow binary at %s: %w", targetPath, err)
	}

	return nil
}
