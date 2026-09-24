package utils

import (
	"fmt"
)

// version holds the channel/version marker, defaulting to "dev": the half of
// the version string that says which channel a build came from. The commit
// that binary was installed from is appended to it by VersionDisplay (see
// version.go). SetVersion overrides the marker at startup (e.g., during build).
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

// SetVersion overrides the channel/version marker.
// Typically used by main.go via build-time injection.
func SetVersion(v string) {
	version = v
}

// VersionMarker returns the channel/version marker on its own, without the
// installed commit. The machine-readable version document reports it next to
// the full revision so a caller never has to split a composed string.
func VersionMarker() string {
	return version
}

// GetVersion returns the version string to show a human: the channel/version
// marker followed by the commit this binary was installed from, marked
// `-dirty` when that tree had uncommitted changes. See VersionDisplay.
func GetVersion() string {
	return VersionDisplay()
}

// PrintBanner prints the dflow ASCII banner with the current version.
// Useful for CLI startup or version subcommand.
//
// It interpolates GetVersion, the same string every other entry point renders,
// so the banner cannot report a different build than `dflow version` does.
func PrintBanner() {
	fmt.Printf(banner, GetVersion())
}
