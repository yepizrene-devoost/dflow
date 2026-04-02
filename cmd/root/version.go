package root

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yepizrene-devoost/dflow/cmd/utils"
)

var showVersion bool

// VersionCmd prints the current dflow version.
var VersionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"ver"},
	Short:   "Show the current dflow version",
	Long:    "Show the current dflow version as injected at build time.",
	Example: `  dflow version
  dflow --version
  dflow -V`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "dflow %s\n", utils.GetVersion())
	},
}
