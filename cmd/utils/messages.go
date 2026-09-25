package utils

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

// The shape of an icon-prefixed line, shared by the renderer and the wrapper.
const (
	// iconPrefixWidth is the column count of printWithIcon's "%-3s " prefix:
	// the icon padded to three columns plus one separating space.
	iconPrefixWidth = 4
	// continuationIndent hangs a wrapped line under the message column rather
	// than under the icon, so a paragraph reads as one block. Its width is
	// iconPrefixWidth, and the two constants are kept adjacent so they cannot
	// drift apart unnoticed.
	continuationIndent = "    "
	// minWrapWidth is the narrowest terminal printWithIcon will wrap for.
	//
	// A very narrow terminal would otherwise break every icon line into a
	// column of two-word fragments, which is harder to read than letting a
	// long line ride past the edge. A reported width below this floor is
	// treated as this floor, so the wrap point has a readable lower bound.
	minWrapWidth = 80
	// maxWrapWidth is the widest content width printWithIcon will wrap for.
	//
	// A very wide terminal would otherwise let an icon line run to nearly its
	// full width — 196 content columns on a 200-column terminal — while the
	// release-notes digest the same update report prints wraps at its own,
	// narrower measure. Capping the content width keeps a long icon line and a
	// digest bullet on one common measure, so the report reads as one document:
	// a message longer than the cap wraps at the cap even when the terminal
	// could show more. It caps content, not the terminal, so it never competes
	// with minWrapWidth's floor on the terminal width.
	maxWrapWidth = 100
)

// Error prints a message with a red cross (❌) prefix.
// Used to display fatal or important errors to the user.
func Error(formattedMessage string, args ...interface{}) {
	printWithIcon("❌", formattedMessage, args...)
}

// Info prints a message with an information icon (ℹ️) prefix.
// Used to display general information about the process.
func Info(formattedMessage string, args ...interface{}) {
	printWithIcon("ℹ️", formattedMessage, args...)
}

// Success prints a message with a checkmark icon (✅) prefix.
// Used to indicate successful completion of an operation.
func Success(formattedMessage string, args ...interface{}) {
	printWithIcon("✅", formattedMessage, args...)
}

// Warn prints a message with a warning icon (⚠️) prefix.
// Used to display non-fatal issues or important alerts.
func Warn(formattedMessage string, args ...interface{}) {
	printWithIcon("⚠️", formattedMessage, args...)
}

// Icon renders formattedMessage with an explicit leading icon, for output whose
// icon is not the level default. The icon is a declared parameter, never
// inferred, so a short value argument can never be mistaken for one.
func Icon(icon string, formattedMessage string, args ...interface{}) {
	printWithIcon(icon, formattedMessage, args...)
}

// Plain prints the formatted message followed by a single newline, with no icon.
//
// Use Plain for output that is the requested result rather than commentary
// around it: the author and email lines of `config get-author`, or the raw
// listing from `config list`, for example. It exists so that every user-facing
// write in the command path goes through this package, which keeps a future
// output mode a change here instead of a hunt through the command layer.
func Plain(formattedMessage string, args ...interface{}) {
	if CurrentFormat() == FormatJSON {
		return
	}
	fmt.Println(fmt.Sprintf(formattedMessage, args...))
}

// The SGR sequences that turn bold on and off around a styled line. Naming them
// keeps the escape bytes out of the formatting call and gives the style one place
// to change if it ever does.
const (
	boldStart = "\033[1m"
	boldEnd   = "\033[0m"
)

// PlainBold prints the formatted message followed by a single newline, exactly
// like Plain, but wraps the message in the ANSI bold sequence when stdout is a
// terminal, so a line that heads a block of output — a release-notes section
// title, for example — visibly anchors the lines beneath it.
//
// The styling is gated twice, and both gates matter:
//
//   - The terminal gate, through the stdoutIsTTY seam, keeps a pipe, a file or a
//     command substitution free of escape codes: off a terminal the output is
//     byte-identical to Plain, so a redirected or scripted run stays exactly as
//     copyable as it was before this helper existed.
//   - The shared JSON suppression every helper honours: a machine-readable run
//     gets its document and nothing else, so the styled line is dropped rather
//     than bolded into stdout.
//
// Unlike printWithIcon, PlainBold never wraps. It exists for caller-built lines
// whose width the caller has already reasoned about — the release-notes digest
// sizes each line to the terminal itself — so wrapping here would size it twice.
func PlainBold(formattedMessage string, args ...interface{}) {
	if CurrentFormat() == FormatJSON {
		return
	}

	message := fmt.Sprintf(formattedMessage, args...)
	if !stdoutIsTTY() {
		fmt.Println(message)
		return
	}
	fmt.Printf("%s%s%s\n", boldStart, message, boldEnd)
}

// Notice prints one unadorned line to stderr, with no icon.
//
// Use Notice for out-of-band information about the run as a whole rather than a
// result of the command the user asked for: the startup "a new release is
// available" line, for example. It writes to stderr, never stdout, because
// stdout is the command's result stream and may carry a machine-readable
// document; an advisory appended there would corrupt that contract or shift
// field positions for a script that reads it. stderr is exactly the stream a
// caller can ignore without losing the result.
//
// It shares Plain's JSON suppression for the same reason every other helper
// does: a machine-readable run gets exactly its document and nothing else, and
// a notice on stderr would still be noise a wrapper has to filter.
func Notice(formattedMessage string, args ...interface{}) {
	if CurrentFormat() == FormatJSON {
		return
	}
	fmt.Fprintln(os.Stderr, fmt.Sprintf(formattedMessage, args...))
}

// Prompt prints an inline input label without a trailing newline.
//
// Use Prompt for labels that precede a read from stdin, so the cursor stays on
// the same line as the label. It lives here for the same reason as Plain: the
// command path must not write to stdout directly.
func Prompt(label string, args ...interface{}) {
	if CurrentFormat() == FormatJSON {
		return
	}
	fmt.Print(fmt.Sprintf(label, args...))
}

// printWithIcon renders one icon-prefixed message. The icon is always the one
// the caller declared, either the level default from Error, Info, Success or
// Warn or the explicit icon from Icon; the arguments are format values only.
//
// It also owns two output contracts:
//
//   - Suppression: in JSON mode the document is the only thing stdout may carry,
//     so every icon-prefixed line yields to it.
//   - Wrapping: when stdout is a terminal, a message longer than the available
//     width is broken at word boundaries and the continuation lines are indented
//     with continuationIndent, so the text hangs under the message column
//     instead of running to column zero under the icon. The first line keeps the
//     historical "%-3s %s" shape byte for byte.
//
// Wrapping is deliberately conditional on a measured terminal. When the width is
// unknown — stdout is a pipe, a file or a command substitution — the message is
// printed as one line, so piped and scripted output stays unwrapped and remains
// easy to copy verbatim. A measured terminal width below minWrapWidth is treated
// as the floor, so the wrap point has a readable lower bound, and a content
// width above maxWrapWidth is treated as the cap, so a very wide terminal shares
// one measure with the release-notes digest; a single token wider than the
// available width is left intact and overflows its line rather than being split.
// See wrapIconMessage for the exact fill rule.
func printWithIcon(icon string, formattedMessage string, args ...interface{}) {
	if CurrentFormat() == FormatJSON {
		return
	}

	msg := fmt.Sprintf(formattedMessage, args...)

	width, ok := stdoutWidth()
	if !ok {
		fmt.Printf("%-3s %s\n", icon, msg)
		return
	}
	if width < minWrapWidth {
		width = minWrapWidth
	}
	wrapWidth := width - iconPrefixWidth
	if wrapWidth > maxWrapWidth {
		wrapWidth = maxWrapWidth
	}

	for i, line := range wrapIconMessage(msg, wrapWidth) {
		if i == 0 {
			fmt.Printf("%-3s %s\n", icon, line)
			continue
		}
		fmt.Printf("%s%s\n", continuationIndent, line)
	}
}

// wrapIconMessage breaks a message into the physical lines printWithIcon should
// render, given the content width available after the icon prefix.
//
// It fills each line greedily from the words in order, so a break always
// replaces a space and no word is ever split across lines. A message that
// already fits is returned as a single element, unchanged, byte for byte; only
// a message that must wrap is re-flowed, and that re-flow normalises whitespace
// runs to a single space.
//
// The one exception is an overlong token: a single word wider than contentWidth
// is placed on its own line and overflows. A URL, a path or a hash is one value,
// and cutting it would corrupt it, so the terminal's own edge wrapping is the
// lesser evil.
//
// Widths are counted in runes, matching the rest of the output layer (the
// release-notes clamp, for example). A double-width glyph — CJK text or an emoji
// — occupies two terminal columns while counting as one rune, so a message
// carrying one can still meet the terminal edge and be wrapped there by the
// terminal itself; display-column measurement is deliberately not attempted.
func wrapIconMessage(msg string, contentWidth int) []string {
	if contentWidth < 1 {
		contentWidth = 1
	}
	if utf8.RuneCountInString(msg) <= contentWidth {
		return []string{msg}
	}

	var lines []string
	current := ""
	for _, word := range strings.Fields(msg) {
		if current == "" {
			current = word
			continue
		}
		if utf8.RuneCountInString(current)+1+utf8.RuneCountInString(word) <= contentWidth {
			current += " " + word
			continue
		}
		lines = append(lines, current)
		current = word
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
