package utils

import (
	"fmt"
)

// version holds the current CLI version, defaulting to "dev".
// It can be overridden using SetVersion at runtime (e.g., during build).
var version = "dev"

// banner defines the ASCII art displayed when the CLI starts.
// The placeholder (%s) is replaced by the CLI version.
const banner = `
                ██████╗ ███████╗██╗      ██████╗ ██╗    ██╗
                ██╔══██╗██╔════╝██║     ██╔═══██╗██║    ██║
                ██║  ██║█████╗  ██║     ██║   ██║██║ █╗ ██║
                ██║  ██║██╔══╝  ██║     ██║   ██║██║███╗██║
                ██████╔╝██║     ███████╗╚██████╔╝╚███╔███╔╝
                ╚═════╝ ╚═╝     ╚══════╝ ╚═════╝  ╚══╝╚══╝ 
                   dflow %s - Git branching made simple
`

// SetVersion overrides the internal version string.
// Typically used by main.go via build-time injection.
func SetVersion(v string) {
	version = v
}

// GetVersion returns the current CLI version string.
func GetVersion() string {
	return version
}

// PrintBanner prints the dflow ASCII banner with the current version.
// Useful for CLI startup or version subcommand.
func PrintBanner() {
	fmt.Printf(banner, version)
}
