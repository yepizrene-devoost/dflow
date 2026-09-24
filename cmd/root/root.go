// Package root defines the root command for the dflow CLI.
//
// This package initializes the top-level `dflow` command, sets up persistent behavior,
// and attaches all supported subcommands. It uses Cobra for command parsing.
package root

import (
	"fmt"
	"os"
	"strconv"
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

	// Cobra must not print errors or usage on its own: Execute renders the
	// failure exactly once, keeping the styled single-render contract.
	SilenceErrors: true,
	SilenceUsage:  true,

	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// The banner needs a terminal; the update notice does not, so the probe
		// is armed outside this guard. A piped run still gets the notice on
		// stderr, which is exactly the caller that most wants to hear about a
		// newer release.
		if utils.IsInteractive() && !shouldSkipBanner(os.Args[1:]) {
			utils.PrintBanner()
		}
		maybeStartUpdateProbe(cmd)
	},

	// PersistentPostRun is where the notice renders: after the command's own
	// output and never on the error path, which has already exited by then.
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		renderPendingUpdateNotice()
	},

	Run: func(cmd *cobra.Command, args []string) {
		if showVersion {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "dflow %s\n", utils.GetVersion())
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
	// Cobra parses flags before it runs PersistentPreRun or any command's
	// PreRunE, so a flag-parse or arity error on a `--json` invocation would
	// otherwise fall through to the human renderer. Pre-selecting the format
	// from the raw arguments keeps the contract "JSON in, JSON out" with no
	// exceptions. The commands' PreRunE declarations are unchanged: both paths
	// set the same value, and PreRunE remains the per-command declaration.
	if jsonRequested(os.Args[1:]) {
		utils.SetFormat(utils.FormatJSON)
	}

	if err := RootCmd.Execute(); err != nil {
		// Single render point: Cobra is silenced above, so the failure is
		// reported once here and the process exits non-zero.
		//
		// In JSON mode that same failure must still be machine-readable: it
		// becomes one {"error": ...} document on stdout, and the non-zero exit
		// code is unchanged.
		if utils.CurrentFormat() == utils.FormatJSON {
			if emitErr := utils.EmitJSON(map[string]string{"error": err.Error()}); emitErr != nil {
				fmt.Fprintf(os.Stderr, "failed to encode the error as JSON: %v\n", emitErr)
			}
			os.Exit(1)
		}

		utils.Error("%s", err.Error())
		os.Exit(1)
	}
}

func init() {
	RootCmd.AddCommand(CompletionCmd)
	RootCmd.AddCommand(commands.InitCmd)
	RootCmd.AddCommand(commands.StartCmd)
	RootCmd.AddCommand(commands.FinishCmd)
	RootCmd.AddCommand(commands.StatusCmd)
	RootCmd.AddCommand(commands.ConfigCmd)
	RootCmd.AddCommand(commands.DeleteCmd)
	RootCmd.AddCommand(commands.UpdateCmd)
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

		// A machine-readable invocation must never receive the banner; see
		// jsonRequested for why the raw argument is the signal here.
		if jsonRequested([]string{arg}) {
			return true
		}
	}
	return false
}

// jsonRequested reports whether the raw process arguments request the
// machine-readable output format.
//
// It reads the arguments instead of the parsed flags because those are the only
// signal available before Cobra parses them: the banner must not reach a
// machine-readable invocation, and a flag-parse failure on a `--json` call must
// still answer as JSON. The `--json=true` form is honoured via the same boolean
// parser used elsewhere, so an explicit `--json=false` is not a request.
func jsonRequested(args []string) bool {
	for _, arg := range args {
		if arg == "--json" {
			return true
		}
		if value, ok := strings.CutPrefix(arg, "--json="); ok {
			if enabled, err := strconv.ParseBool(value); err == nil && enabled {
				return true
			}
		}
	}
	return false
}
