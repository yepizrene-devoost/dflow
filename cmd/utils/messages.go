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

// Internal helper with optional icon override (like Spinner.Stop).
func printWithIcon(defaultIcon string, formattedMessage string, args ...interface{}) {
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
