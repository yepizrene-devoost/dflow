package utils

import (
	"encoding/json"
	"os"
)

// Format selects how the CLI renders its result.
//
// The zero value is FormatHuman, so existing behaviour is what a command gets
// unless it explicitly opts into a machine-readable format from its PreRunE.
type Format int

const (
	// FormatHuman is the default: styled, icon-prefixed lines meant for a person.
	FormatHuman Format = iota
	// FormatJSON makes stdout carry exactly one machine-readable document and
	// nothing else: no banner, no icons, no progress lines.
	FormatJSON
)

// currentFormat is the process-wide output format.
//
// A CLI process renders exactly one result and then exits, so a package-level
// value is the whole state needed: commands set it early (in PreRunE) and every
// output helper reads it.
var currentFormat = FormatHuman

// SetFormat sets the process-wide output format.
func SetFormat(f Format) {
	currentFormat = f
}

// CurrentFormat returns the process-wide output format.
func CurrentFormat() Format {
	return currentFormat
}

// EmitJSON writes value to stdout as one compact, newline-terminated JSON
// document.
//
// Callers must have suppressed every human-oriented message first: in JSON mode
// stdout is reserved for this document, so a stray banner or icon would break
// the machine-readable contract rather than merely look untidy.
func EmitJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	// Keep values such as `<`, `>` and `&` literal: the document is a contract,
	// not HTML, and plain text is easier to diff and debug.
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}
