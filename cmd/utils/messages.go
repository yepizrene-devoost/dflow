package utils

import "fmt"

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

// Internal helper with optional icon override (like Spinner.Stop).
//
// It is also the single suppression point for Error, Info, Success and Warn: in
// JSON mode the document is the only thing stdout may carry, so every
// icon-prefixed line yields to it.
func printWithIcon(defaultIcon string, formattedMessage string, args ...interface{}) {
	if CurrentFormat() == FormatJSON {
		return
	}

	finalIcon := defaultIcon
	// Si el último argumento es un string extra (ícono), úsalo
	if len(args) > 0 {
		if last, ok := args[len(args)-1].(string); ok && isCustomIcon(last) {
			finalIcon = last
			args = args[:len(args)-1] // eliminar icono de args
		}
	}
	msg := fmt.Sprintf(formattedMessage, args...)
	fmt.Printf("%-3s %s\n", finalIcon, msg)
}

// isCustomIcon checks if the string is likely an emoji or custom icon.
func isCustomIcon(s string) bool {
	return len(s) > 0 && len([]rune(s)) <= 2 // Emoji típicamente es 1–2 runas
}
