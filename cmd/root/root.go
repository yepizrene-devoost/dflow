// Package root defines the root command for the dflow CLI.
//
// This package initializes the top-level `dflow` command, sets up persistent behavior,
// and attaches all supported subcommands. It uses Cobra for command parsing.
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
// for all subcommands like `start`, `finish`, `init`, `config`, and `delete`.
var RootCmd = &cobra.Command{
	Use:   "dflow",
	Short: "Manage Git branches with a configurable workflow",
	Long: `A CLI tool to manage Git branch workflows inspired by Git Flow.

It helps repositories define branch rules, start work branches with consistent
prefixes, manage project-local metadata, and automate repetitive branching
tasks with a customizable flow model.`,
	Example: `  dflow init
  dflow start feat login-form
  dflow finish
  dflow start bug checkout-on-uat
  dflow config set-author "Jane Doe" --email=jane@example.com
  dflow version`,

	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if shouldSkipBanner(os.Args[1:]) {
			return
		}
		utils.PrintBanner()
	},

	Run: func(cmd *cobra.Command, args []string) {
		if showVersion {
			fmt.Fprintf(cmd.OutOrStdout(), "dflow %s\n", utils.GetVersion())
			return
		}
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
	RootCmd.AddCommand(commands.FinishCmd)
	RootCmd.AddCommand(commands.ConfigCmd)
	RootCmd.AddCommand(commands.DeleteCmd)
	RootCmd.AddCommand(VersionCmd)
	RootCmd.PersistentFlags().BoolVarP(&showVersion, "version", "V", false, "Show the current dflow version")

	// customize help
	RootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		cmd.Root().SetHelpFunc(nil)
		_ = cmd.Help()
	})

}

func shouldSkipBanner(args []string) bool {
	for _, arg := range args {
		if strings.HasPrefix(arg, "__complete") || arg == "completion" || arg == "--help" || arg == "-h" || arg == "help" || arg == "--version" || arg == "-V" || arg == "version" || arg == "ver" {
			return true
		}
	}
	return false
}
