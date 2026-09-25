// This file is the write half of the discovery layer: it renders the reference
// dflow embeds in an agent's instruction file and keeps that reference current.
// Placement lives here; the workflow document itself is rendered elsewhere.
package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// The block is delimited by HTML comment markers so a later run can replace
// exactly these bytes and never guess at section boundaries.
const (
	referenceStartMarker = "<!-- dflow:workflow-reference -->"
	referenceEndMarker   = "<!-- /dflow:workflow-reference -->"
	referenceHeading     = "## dflow Workflow"
)

// ReferenceBlock renders the reference dflow writes into an instruction file.
//
// The result is a self-contained markdown section with a trailing newline.
func ReferenceBlock(workflowPath string) string {
	return referenceStartMarker + "\n" +
		referenceHeading + "\n" +
		"Read `" + workflowPath + "` for branch types, merge rules, and finish flow.\n" +
		referenceEndMarker + "\n"
}

// EnsureInstructionReference makes path carry the dflow workflow reference.
//
// It is idempotent and conservative about foreign content:
//   - an absent file is created with the block;
//   - a file without markers and without a "## dflow Workflow" heading gets the
//     block appended after its existing bytes;
//   - a byte-identical block is left untouched;
//   - a delimited block with different content has only that range replaced;
//   - a lone marker, or a legacy heading without markers, is left untouched.
//
// It reports whether the file changed.
func EnsureInstructionReference(path, workflowPath string) (bool, error) {
	block := ReferenceBlock(workflowPath)

	existing, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if createErr := createInstructionFile(path, block); createErr != nil {
				return false, createErr
			}
			return true, nil
		}
		return false, fmt.Errorf("read instruction file %s: %w", path, err)
	}
	content := string(existing)

	startIdx, endIdx, bothMarkers := locateReferenceBlock(content)
	if hasMarker(content) && !bothMarkers {
		// Rule 5: never guess a boundary from a half-written block.
		return false, nil
	}

	if bothMarkers {
		// The replacement range runs through the end marker's line, including
		// its newline when present, so the block's own trailing newline does not
		// double up.
		blockEnd := endIdx + len(referenceEndMarker)
		if blockEnd < len(content) && content[blockEnd] == '\n' {
			blockEnd++
		}
		if content[startIdx:blockEnd] == block {
			return false, nil
		}
		updated := content[:startIdx] + block + content[blockEnd:]
		if err := writeInstructionFile(path, updated); err != nil {
			return false, err
		}
		return true, nil
	}

	if strings.Contains(content, referenceHeading) {
		// Rule 6: a project already carries the reference section (the one
		// issue #31's init wrote carried no marker). Do not duplicate it.
		return false, nil
	}

	if err := writeInstructionFile(path, appendReference(content, block)); err != nil {
		return false, err
	}
	return true, nil
}

// locateReferenceBlock finds a well-formed, ordered marker pair. bothMarkers is
// false when either marker is missing or they appear in the wrong order.
func locateReferenceBlock(content string) (start, end int, bothMarkers bool) {
	start = strings.Index(content, referenceStartMarker)
	if start < 0 {
		return 0, 0, false
	}
	rel := strings.Index(content[start+len(referenceStartMarker):], referenceEndMarker)
	if rel < 0 {
		return 0, 0, false
	}
	return start, start + len(referenceStartMarker) + rel, true
}

// hasMarker reports whether content carries either reference marker.
func hasMarker(content string) bool {
	return strings.Contains(content, referenceStartMarker) || strings.Contains(content, referenceEndMarker)
}

// appendReference appends block after content with exactly one blank line
// between. Existing bytes are preserved; a missing trailing newline is added as
// part of the separator.
func appendReference(content, block string) string {
	if content == "" {
		return block
	}
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return content + "\n" + block
}

// createInstructionFile creates path and its parents, then writes the block.
//
// The create goes through the same atomic replacement as the overwrite: an
// interrupted create must not leave a truncated file either.
func createInstructionFile(path, block string) error {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create instruction directory %s: %w", dir, err)
		}
	}
	if err := writeFileAtomic(path, []byte(block), 0644); err != nil {
		return fmt.Errorf("write instruction file %s: %w", path, err)
	}
	return nil
}

// writeInstructionFile overwrites path with content at mode 0644.
//
// The overwrite goes through the same atomic replacement the skill installer
// uses, so an interrupted run leaves the previous file whole rather than a
// truncated one.
func writeInstructionFile(path, content string) error {
	if err := writeFileAtomic(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write instruction file %s: %w", path, err)
	}
	return nil
}
