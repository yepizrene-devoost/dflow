package root

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yepizrene-devoost/dflow/cmd/utils"
)

var showVersion bool

// versionReport is the machine-readable report of `dflow version`.
//
// Field order is the document order encoding/json produces. The marker and the
// revision are reported as separate fields so a caller never has to split the
// composed human string, and revision is empty rather than "unknown" so the
// document stays a value, not a sentence.
type versionReport struct {
	Version  string `json:"version"`
	Revision string `json:"revision"`
	Dirty    bool   `json:"dirty"`
}

// VersionCmd prints the current dflow version.
var VersionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"ver"},
	Short:   "Show the current dflow version",
	Long: `Show the channel/version marker and the commit this binary was installed from.

The first token is the channel/version marker: "dev" for a development install,
or the release version injected at build time. The second token is the
abbreviated commit the binary reports from its own VCS stamp, suffixed "-dirty"
when the build tree had uncommitted changes. A binary built without a VCS stamp
shows the marker alone.

Pass --revision to print only the full 40-character commit hash, or the literal
"unknown" when the binary carries no stamp. Pass --json for one machine-readable
document instead; it takes precedence over --revision and reports the marker,
the full revision and the dirty flag.`,
	Example: `  dflow version
  dflow version --revision
  dflow version --json
  dflow --version
  dflow -V`,
	Args: cobra.NoArgs,
	// The output format must be decided before Run renders, so a failure is
	// reported in the requested format. Cobra runs PersistentPreRun, then
	// PreRunE, then RunE: this mirrors the existing --json contract in status.go.
	//
	// Only the format is settled here. `--revision` is deliberately resolved
	// inside RunE instead: unlike the format it does not change how the command
	// reports, so it needs no package-level state to survive the hop between the
	// two hooks, and keeping it next to the branch that consumes it means an
	// unreadable flag is returned there rather than stored where RunE would have
	// to trust it.
	PreRunE: func(cmd *cobra.Command, args []string) error {
		jsonOutput, err := cmd.Flags().GetBool("json")
		if err != nil {
			return err
		}
		if jsonOutput {
			utils.SetFormat(utils.FormatJSON)
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if utils.CurrentFormat() == utils.FormatJSON {
			report := versionReport{
				Version:  utils.VersionMarker(),
				Revision: utils.Revision(),
				Dirty:    utils.RevisionDirty(),
			}
			// A document that cannot be encoded must not look like success: the
			// machine caller reads stdout and a zero exit code as "the report is
			// there". Returning the error keeps the contract "JSON in, JSON out"
			// because Execute renders the failure as one {"error": ...} document.
			return utils.EmitJSON(report)
		}

		// Resolved here rather than in PreRunE: --revision only picks which of the
		// two human representations Run renders, so the lookup belongs next to the
		// branch that uses it and a failure is returned instead of being parked in
		// package state. --json still wins, because this is reached only when no
		// machine-readable format was requested.
		revisionOnly, err := cmd.Flags().GetBool("revision")
		if err != nil {
			return err
		}

		if revisionOnly {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), revisionToken())
			return nil
		}

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "dflow %s\n", utils.GetVersion())
		return nil
	},
}

// revisionToken renders the script-facing revision: the full commit hash, or
// the literal "unknown" when the binary carries no VCS stamp, so a caller can
// branch on one token instead of on an empty line.
func revisionToken() string {
	if revision := utils.Revision(); revision != "" {
		return revision
	}
	return "unknown"
}

func init() {
	VersionCmd.Flags().Bool("revision", false, "Print only the full commit revision on stdout")
	VersionCmd.Flags().Bool("json", false, "Print the version as a single JSON document on stdout")
}
