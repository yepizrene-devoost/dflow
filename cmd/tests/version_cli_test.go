package tests

import (
	"strings"
	"testing"
	"time"
)

// TestVersionCLI covers the user-visible contract of `dflow version` against the
// real binary: every entry point prints the same line, --revision is a single
// script-friendly token, and --json is one machine-readable document.
//
// The binary is built from this checkout, so it may or may not carry a VCS
// stamp depending on how the sources were obtained. The assertions pin the
// shape (a full 40-character hash, or the literal "unknown" / an empty JSON
// revision) rather than a specific commit, which would break on every commit.
func TestVersionCLI(t *testing.T) {
	setUpCLIEnv(t)
	binary := buildDflowCLI(t)
	workDir := t.TempDir()

	// humanLine is the single line the four human entry points print, kept here
	// so the JSON case can prove the two representations describe one build.
	var humanLine string

	t.Run("entry points agree", func(t *testing.T) {
		for _, args := range [][]string{
			{"version"},
			{"ver"},
			{"--version"},
			{"-V"},
		} {
			output, exitCode := startCLIRawOutput(t, 15*time.Second, workDir, binary, args...)
			if exitCode != 0 {
				t.Fatalf("dflow %s exited %d, want 0\n%s", strings.Join(args, " "), exitCode, output)
			}

			line := singleTrimmedLine(t, output)
			if !strings.HasPrefix(line, "dflow ") {
				t.Fatalf("dflow %s printed %q, want a line starting with %q", strings.Join(args, " "), line, "dflow ")
			}
			if humanLine == "" {
				humanLine = line
				continue
			}
			if line != humanLine {
				t.Fatalf("dflow %s printed %q, want the same line as the other entry points (%q)", strings.Join(args, " "), line, humanLine)
			}
		}

		if humanLine == "" {
			t.Fatalf("no version entry point produced output")
		}
	})

	t.Run("--revision", func(t *testing.T) {
		output, exitCode := startCLIRawOutput(t, 15*time.Second, workDir, binary, "version", "--revision")
		if exitCode != 0 {
			t.Fatalf("version --revision exited %d, want 0\n%s", exitCode, output)
		}

		token := singleTrimmedLine(t, output)
		if strings.ContainsAny(token, " \t") {
			t.Fatalf("version --revision must print exactly one token, got %q", token)
		}
		if token != "unknown" && !isFullRevision(token) {
			t.Fatalf("version --revision printed %q, want 40 hex characters or %q", token, "unknown")
		}
	})

	t.Run("--json", func(t *testing.T) {
		output, exitCode := startCLIRawOutput(t, 15*time.Second, workDir, binary, "version", "--json")
		if exitCode != 0 {
			t.Fatalf("version --json exited %d, want 0\n%s", exitCode, output)
		}
		assertNoHumanChrome(t, output)

		doc := decodeSingleJSONDocument(t, output)
		if len(doc) != 3 {
			t.Fatalf("version --json has %d keys, want exactly 3:\n%v", len(doc), doc)
		}

		marker, ok := doc["version"].(string)
		if !ok {
			t.Fatalf("version field = %#v, want a string", doc["version"])
		}
		revision, ok := doc["revision"].(string)
		if !ok {
			t.Fatalf("revision field = %#v, want a string", doc["revision"])
		}
		dirty, ok := doc["dirty"].(bool)
		if !ok {
			t.Fatalf("dirty field = %#v, want a boolean", doc["dirty"])
		}

		if revision != "" && !isFullRevision(revision) {
			t.Fatalf("revision field = %q, want an empty string or 40 hex characters", revision)
		}
		if revision == "" && dirty {
			t.Fatalf("revision is empty but dirty = true, want dirty false without a stamp")
		}

		// The JSON document and the human line must describe the same build:
		// rebuild the line from the machine-readable fields and compare.
		want := marker
		if revision != "" {
			want += " " + abbreviateRevision(revision)
			if dirty {
				want += "-dirty"
			}
		}
		if humanLine != "dflow "+want {
			t.Fatalf("human line %q does not match the JSON document's build (%q)", humanLine, "dflow "+want)
		}
	})

	// The combination the issue leaves unspecified is pinned here so it cannot
	// change silently: a machine-readable request wins over the single-token
	// form, because a caller that asked for JSON must never receive a bare hash.
	t.Run("--json wins over --revision", func(t *testing.T) {
		output, exitCode := startCLIRawOutput(t, 15*time.Second, workDir, binary, "version", "--revision", "--json")
		if exitCode != 0 {
			t.Fatalf("version --revision --json exited %d, want 0\n%s", exitCode, output)
		}
		assertNoHumanChrome(t, output)

		doc := decodeSingleJSONDocument(t, output)
		if len(doc) != 3 {
			t.Fatalf("version --revision --json has %d keys, want the 3-key version document:\n%v", len(doc), doc)
		}
	})
}

// singleTrimmedLine asserts the output is exactly one newline-terminated line
// and returns it without the trailing newline, so a banner, a second line or
// stray whitespace cannot pass unnoticed.
func singleTrimmedLine(t *testing.T, output string) string {
	t.Helper()

	if !strings.HasSuffix(output, "\n") {
		t.Fatalf("output must end with a newline, got:\n%q", output)
	}
	line := strings.TrimSuffix(output, "\n")
	if strings.ContainsAny(line, "\r\n") {
		t.Fatalf("output must be a single line, got:\n%q", output)
	}
	return line
}

// isFullRevision reports whether token is a complete 40-character Git hash.
func isFullRevision(token string) bool {
	if len(token) != 40 {
		return false
	}
	for _, r := range token {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		case r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}

// abbreviateRevision shortens a full hash to the seven characters the human
// line carries, mirroring the same rule the CLI applies.
func abbreviateRevision(revision string) string {
	const width = 7
	if len(revision) <= width {
		return revision
	}
	return revision[:width]
}
