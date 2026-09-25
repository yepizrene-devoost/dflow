package agent

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

const testWorkflowPath = ".agents/workflows/dflow.md"

// referenceBlock pins the exact bytes ReferenceBlock must render. The byte
// content is the contract: downstream agents parse markers, not prose.
func referenceBlockPin(workflowPath string) string {
	return "<!-- dflow:workflow-reference -->\n" +
		"## dflow Workflow\n" +
		"Read `" + workflowPath + "` for branch types, merge rules, and finish flow.\n" +
		"<!-- /dflow:workflow-reference -->\n"
}

func TestReferenceBlockExactBytes(t *testing.T) {
	got := ReferenceBlock(testWorkflowPath)
	want := referenceBlockPin(testWorkflowPath)

	if got != want {
		t.Errorf("ReferenceBlock() =\n%q\nwant\n%q", got, want)
	}
}

func TestReferenceBlockUsesWorkflowPath(t *testing.T) {
	got := ReferenceBlock("custom/path.md")
	if want := referenceBlockPin("custom/path.md"); got != want {
		t.Errorf("ReferenceBlock(\"custom/path.md\") =\n%q\nwant\n%q", got, want)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

// Rule 1 + Rule 7: absent file is created, including its parent directory.
func TestEnsureInstructionReferenceCreatesNestedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "docs", "AGENT.md")

	changed, err := EnsureInstructionReference(path, testWorkflowPath)
	if err != nil {
		t.Fatalf("EnsureInstructionReference() error: %v", err)
	}
	if !changed {
		t.Error("changed = false, want true when creating the file")
	}
	if got, want := readFile(t, path), referenceBlockPin(testWorkflowPath); got != want {
		t.Errorf("created file =\n%q\nwant\n%q", got, want)
	}
}

func TestEnsureInstructionReferenceCreatesMode0644(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")

	if _, err := EnsureInstructionReference(path, testWorkflowPath); err != nil {
		t.Fatalf("EnsureInstructionReference() error: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Errorf("created file mode = %o, want 644", got)
	}
}

// Rule 2: absent markers and heading -> append, preserving existing bytes and
// separating with exactly one blank line.
func TestEnsureInstructionReferenceAppendsAfterExistingBytes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	existing := "# Title\n\nSome existing rules.\n"
	writeFile(t, path, existing)

	changed, err := EnsureInstructionReference(path, testWorkflowPath)
	if err != nil {
		t.Fatalf("EnsureInstructionReference() error: %v", err)
	}
	if !changed {
		t.Error("changed = false, want true when appending")
	}
	want := existing + "\n" + referenceBlockPin(testWorkflowPath)
	if got := readFile(t, path); got != want {
		t.Errorf("appended file =\n%q\nwant\n%q", got, want)
	}
}

func TestEnsureInstructionReferenceAppendsToFileWithoutTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	existing := "# Title"
	writeFile(t, path, existing)

	changed, err := EnsureInstructionReference(path, testWorkflowPath)
	if err != nil {
		t.Fatalf("EnsureInstructionReference() error: %v", err)
	}
	if !changed {
		t.Error("changed = false, want true when appending")
	}
	want := existing + "\n\n" + referenceBlockPin(testWorkflowPath)
	if got := readFile(t, path); got != want {
		t.Errorf("appended file =\n%q\nwant\n%q", got, want)
	}
}

// Rule 3: byte-identical block -> no-op, file untouched.
func TestEnsureInstructionReferenceIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	block := referenceBlockPin(testWorkflowPath)
	writeFile(t, path, block)

	changed, err := EnsureInstructionReference(path, testWorkflowPath)
	if err != nil {
		t.Fatalf("EnsureInstructionReference() error: %v", err)
	}
	if changed {
		t.Error("changed = true, want false for an already-identical block")
	}
	if got := readFile(t, path); got != block {
		t.Errorf("file changed: got\n%q\nwant\n%q", got, block)
	}
}

// Rule 4: both markers present but content differs -> replace only that range.
func TestEnsureInstructionReferenceReplacesChangedBlockRange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	oldBlock := ReferenceBlock("old/path.md")
	newBlock := ReferenceBlock("new/path.md")
	writeFile(t, path, "before\n"+oldBlock+"after\n")

	changed, err := EnsureInstructionReference(path, "new/path.md")
	if err != nil {
		t.Fatalf("EnsureInstructionReference() error: %v", err)
	}
	if !changed {
		t.Error("changed = false, want true when the block content differs")
	}
	want := "before\n" + newBlock + "after\n"
	if got := readFile(t, path); got != want {
		t.Errorf("replaced file =\n%q\nwant\n%q", got, want)
	}
}

// The overwrite path replaces the file rather than truncating it in place: a
// reader must never observe a half-written AGENTS.md, which is what an
// interrupted in-place rewrite leaves behind. A rename hands the target the
// temporary file's identity, so the inode changes; truncating and rewriting the
// same file would keep it.
//
// The check is POSIX-only because inode numbers are.
func TestEnsureInstructionReferenceReplacesTheFileRatherThanTruncatingIt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("inode numbers are a POSIX filesystem detail")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	// A delimited block naming another workflow path is what makes the replacement
	// range differ, which is the branch that overwrites the file.
	writeFile(t, path, "before\n"+ReferenceBlock("old/path.md")+"after\n")

	before := inodeOf(t, path)
	changed, err := EnsureInstructionReference(path, testWorkflowPath)
	if err != nil {
		t.Fatalf("EnsureInstructionReference() error: %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true when the block content differs")
	}

	if after := inodeOf(t, path); before == after {
		t.Errorf("the instruction file kept inode %d, so it was rewritten in place instead of being replaced atomically", before)
	}

	want := "before\n" + referenceBlockPin(testWorkflowPath) + "after\n"
	if got := readFile(t, path); got != want {
		t.Errorf("replaced file =\n%q\nwant\n%q", got, want)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	if len(names) != 1 || names[0] != "AGENTS.md" {
		t.Fatalf("%s holds %v, want exactly [AGENTS.md]", dir, names)
	}
}

// Rule 5: start marker without end marker -> no-op, never guess the boundary.
func TestEnsureInstructionReferenceStartWithoutEndIsNoOp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	content := "before\n<!-- dflow:workflow-reference -->\nRead `x` for things.\n"
	writeFile(t, path, content)

	changed, err := EnsureInstructionReference(path, testWorkflowPath)
	if err != nil {
		t.Fatalf("EnsureInstructionReference() error: %v", err)
	}
	if changed {
		t.Error("changed = true, want false when the end marker is missing")
	}
	if got := readFile(t, path); got != content {
		t.Errorf("file changed: got\n%q\nwant\n%q", got, content)
	}
}

// A stray end marker without a start marker is just as malformed: no-op.
func TestEnsureInstructionReferenceEndWithoutStartIsNoOp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	content := "before\n<!-- /dflow:workflow-reference -->\nafter\n"
	writeFile(t, path, content)

	changed, err := EnsureInstructionReference(path, testWorkflowPath)
	if err != nil {
		t.Fatalf("EnsureInstructionReference() error: %v", err)
	}
	if changed {
		t.Error("changed = true, want false when the start marker is missing")
	}
	if got := readFile(t, path); got != content {
		t.Errorf("file changed: got\n%q\nwant\n%q", got, content)
	}
}

// Rule 6: no markers but a legacy "## dflow Workflow" heading -> no-op.
func TestEnsureInstructionReferenceLegacyHeadingIsNoOp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	content := "# Title\n\n## dflow Workflow\nRead `.agents/workflows/dflow.md` for branch types.\n"
	writeFile(t, path, content)

	changed, err := EnsureInstructionReference(path, testWorkflowPath)
	if err != nil {
		t.Fatalf("EnsureInstructionReference() error: %v", err)
	}
	if changed {
		t.Error("changed = true, want false when a legacy heading already exists")
	}
	if got := readFile(t, path); got != content {
		t.Errorf("file changed: got\n%q\nwant\n%q", got, content)
	}
}
