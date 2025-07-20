package utils

import "fmt"

// Error prints a message with a red cross (❌) prefix.
// Used to display fatal or important errors to the user.
func Error(formatedMessage string, args ...interface{}) {
	msg := fmt.Sprintf(formatedMessage, args...)
	fmt.Printf("%-3s	%s\n", "❌", msg)
}

// Info prints a message with an information icon (ℹ️) prefix.
// Used to display general information about the process.
func Info(formatedMessage string, args ...interface{}) {
	msg := fmt.Sprintf(formatedMessage, args...)
	fmt.Printf("%-3s	%s\n", "ℹ️", msg)
}

// Success prints a message with a checkmark icon (✅) prefix.
// Used to indicate successful completion of an operation.
func Success(formatedMessage string, args ...interface{}) {
	msg := fmt.Sprintf(formatedMessage, args...)
	fmt.Printf("%-3s	%s\n", "✅", msg)
}

// Warn prints a message with a warning icon (⚠️) prefix.
// Used to display non-fatal issues or important alerts.
func Warn(formatedMessage string, args ...interface{}) {
	msg := fmt.Sprintf(formatedMessage, args...)
	fmt.Printf("%-3s	%s\n", "⚠️", msg)
}
