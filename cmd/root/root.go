// Package root defines the root command for the dflow CLI.
//
// This package initializes the top-level `dflow` command, sets up persistent behavior
// (like displaying the banner), and attaches all subcommands such as `init`, `start`,
// and `config`. It uses Cobra for command parsing.
package root

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yepizrene-devoost/dflow/cmd/commands"
	"github.com/yepizrene-devoost/dflow/cmd/utils"
)

// RootCmd is the base command for the dflow CLI.
//
// It defines global behavior such as the banner, help fallback, and command registration
// for all subcommands like `start`, `init`, `config`, and `delete`.
var RootCmd = &cobra.Command{
	Use:   "dflow",
	Short: "Manage Git branches with Devoost's adjusted flow",
	Long: `dflow is a Git branching CLI tailored to Devoost's workflow.

It helps teams initialize repository branch rules, start work branches with
consistent prefixes, manage project-local metadata, and automate repetitive
branching tasks on top of an adjusted Git Flow model.`,
	Example: `  dflow init
  dflow start feat login-form
  dflow start bug checkout-on-uat
  dflow config set-author "Jane Doe" --email=jane@example.com`,

	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if shouldSkipBanner(os.Args[1:]) {
			return
		}
		utils.PrintBanner()
	},

	Run: func(cmd *cobra.Command, args []string) {
		cmd.SetArgs([]string{"--help"})
		if err := cmd.Execute(); err != nil {
			fmt.Fprintf(os.Stderr, "Command execution failed: %v\n", err)
			os.Exit(1)
		}
	},
}

// Execute runs the root command for the dflow CLI.
//
// It should be called from the `main` function in main.go to start the CLI.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		// fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.AddCommand(CompletionCmd)
	RootCmd.AddCommand(commands.InitCmd)
	RootCmd.AddCommand(commands.StartCmd)
	RootCmd.AddCommand(commands.ConfigCmd)
	RootCmd.AddCommand(commands.DeleteCmd)

	// customize help
	RootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		cmd.Root().SetHelpFunc(nil)
		_ = cmd.Help()
	})

}

func shouldSkipBanner(args []string) bool {
	for _, arg := range args {
		if strings.HasPrefix(arg, "__complete") || arg == "completion" || arg == "--help" || arg == "-h" || arg == "help" {
			return true
		}
	}
	return false
}
