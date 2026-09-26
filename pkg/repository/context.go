// Package repository resolves the canonical Git repository context for dflow.
package repository

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const dflowCWDEnv = "DFLOW_CWD"

// Context identifies the repository and configuration paths used by dflow.
type Context struct {
	WorktreeRoot string
	GitDir       string
	ConfigPath   string
}

// Discover resolves the repository containing the invocation directory.
// DFLOW_CWD, when non-empty, is resolved relative to the process working
// directory; otherwise the process working directory is used.
func Discover() (Context, error) {
	start, err := resolveStart()
	if err != nil {
		return Context{}, err
	}

	root, gitDir, err := resolveGit(start)
	if err != nil {
		return Context{}, err
	}

	return Context{
		WorktreeRoot: root,
		GitDir:       gitDir,
		ConfigPath:   filepath.Join(root, ".dflow.yaml"),
	}, nil
}

func resolveStart() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve process working directory: %w", err)
	}

	return resolveStartPath(os.Getenv(dflowCWDEnv), cwd)
}

func resolveStartPath(configured, cwd string) (string, error) {
	start := configured
	if strings.TrimSpace(start) == "" {
		start = cwd
	} else if !filepath.IsAbs(start) {
		start = filepath.Join(cwd, start)
	}

	info, err := os.Stat(start)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("DFLOW_CWD %q does not exist: %w", start, err)
		}
		if strings.Contains(strings.ToLower(err.Error()), "invalid argument") {
			return "", fmt.Errorf("DFLOW_CWD %q is invalid: %w", start, err)
		}
		return "", fmt.Errorf("inspect DFLOW_CWD %q: %w", start, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("DFLOW_CWD %q is not a directory", start)
	}

	return filepath.Abs(start)
}

func resolveGit(start string) (string, string, error) {
	cmd := exec.Command("git", "-C", start, "rev-parse", "--show-toplevel", "--git-dir")
	output, err := cmd.CombinedOutput()
	if err != nil {
		diagnostics := strings.TrimSpace(string(output))
		if diagnostics == "" {
			return "", "", fmt.Errorf("resolve Git repository from %q: %w", start, err)
		}
		return "", "", fmt.Errorf("resolve Git repository from %q: %w: %s", start, err, diagnostics)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) != 2 || strings.TrimSpace(lines[0]) == "" || strings.TrimSpace(lines[1]) == "" {
		return "", "", fmt.Errorf("resolve Git repository from %q: unexpected git output", start)
	}

	root, err := filepath.Abs(strings.TrimSpace(lines[0]))
	if err != nil {
		return "", "", fmt.Errorf("canonicalize Git worktree root: %w", err)
	}
	gitDir := strings.TrimSpace(lines[1])
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(start, gitDir)
	}
	gitDir, err = filepath.Abs(gitDir)
	if err != nil {
		return "", "", fmt.Errorf("canonicalize Git directory: %w", err)
	}

	return filepath.Clean(root), filepath.Clean(gitDir), nil
}
