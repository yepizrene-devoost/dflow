package utils

import (
	"fmt"
	"os"
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

// printWithIcon renders one icon-prefixed line. The icon is always the one the
// caller declared, either the level default from Error, Info, Success or Warn or
// the explicit icon from Icon; the arguments are format values only.
//
// It is also the single suppression point for every icon-prefixed helper: in
// JSON mode the document is the only thing stdout may carry, so every
// icon-prefixed line yields to it.
func printWithIcon(icon string, formattedMessage string, args ...interface{}) {
	if CurrentFormat() == FormatJSON {
		return
	}

	msg := fmt.Sprintf(formattedMessage, args...)
	fmt.Printf("%-3s %s\n", icon, msg)
}
