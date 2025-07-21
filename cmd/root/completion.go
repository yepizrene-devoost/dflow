// Package root defines the root command for the dflow CLI.
//
// This package initializes the top-level `dflow` command, sets up persistent behavior
// (like displaying the banner), and attaches all subcommands such as `init`, `start`,
// and `config`. It uses Cobra for command parsing.
package root

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

// CompletionCmd defines the parent command for generating and installing shell
// autocompletion scripts. It groups subcommands for each supported shell (bash, zsh, etc.)
// and a helper command to install completion automatically.
var CompletionCmd = &cobra.Command{
	Use:   "completion",
	Short: "Shell completion related commands",
	Long: `The 'completion' command provides tools to enable shell autocompletion for the dflow CLI.

You can either generate a shell-specific script and source it manually, or install it persistently
for supported shells (bash, zsh, fish, powershell).

Examples:

  Generate a zsh script and source it directly:
    $ source <(dflow completion zsh)

  Generate and install autocompletion permanently for your current shell:
    $ dflow completion install

  Output a completion script to a file:
    $ dflow completion bash > ~/.bash_completion

Supported shells:
  - bash
  - zsh
  - fish
  - powershell

For persistent installation, see 'dflow completion install'.`,
}

// GenBashCmd generates a Bash completion script and writes it to stdout.
// Intended to be used with:
//
//	$ source <(dflow completion bash)
var GenBashCmd = &cobra.Command{
	Use:   "bash",
	Short: "Generate bash completion script",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RootCmd.GenBashCompletion(os.Stdout)
	},
}

// GenZshCmd generates a Zsh completion script and writes it to stdout.
// Intended to be used with:
//
//	$ source <(dflow completion zsh)
var GenZshCmd = &cobra.Command{
	Use:   "zsh",
	Short: "Generate zsh completion script",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RootCmd.GenZshCompletion(os.Stdout)
	},
}

// GenFishCmd generates a Fish shell completion script and writes it to stdout.
// Intended to be used with:
//
//	$ source <(dflow completion fish)
var GenFishCmd = &cobra.Command{
	Use:   "fish",
	Short: "Generate fish completion script",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RootCmd.GenFishCompletion(os.Stdout, true)
	},
}

// GenPowerShellCmd generates a PowerShell completion script and writes it to stdout.
// Intended to be used with:
//
//	PS> dflow completion powershell | Out-String | Invoke-Expression
var GenPowerShellCmd = &cobra.Command{
	Use:   "powershell",
	Short: "Generate powershell completion script",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RootCmd.GenPowerShellCompletion(os.Stdout)
	},
}

// CompletionInstallCmd attempts to detect the user's current shell and automatically
// installs the corresponding completion script into their shell config file.
//
// For Bash, it appends to ~/.bash_profile or ~/.bashrc.
// For Zsh, it writes to ~/.zsh/completions/_dflow.
var CompletionInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install shell autocompletion for your current shell",
	RunE: func(cmd *cobra.Command, args []string) error {
		shell := detectShell()
		if shell == "" {
			return fmt.Errorf("could not detect shell")
		}

		usr, err := user.Current()
		if err != nil {
			return err
		}

		var path string
		var script string

		switch shell {
		case "zsh":
			path = filepath.Join(usr.HomeDir, ".zsh/completions/_dflow")
			scriptBuilder := &strings.Builder{}
			if err := RootCmd.GenZshCompletion(scriptBuilder); err != nil {
				return err
			}
			script = scriptBuilder.String()
		case "bash":
			if runtime.GOOS == "darwin" {
				path = filepath.Join(usr.HomeDir, ".bash_profile")
			} else {
				path = filepath.Join(usr.HomeDir, ".bashrc")
			}
			scriptBuilder := &strings.Builder{}
			if err := RootCmd.GenBashCompletion(scriptBuilder); err != nil {
				return err
			}
			// En bash se recomienda agregar el `source <(dflow completion bash)`
			// o guardar el script en `.bash_completion.d`
			script = scriptBuilder.String()
		default:
			return fmt.Errorf("unsupported shell: %s", shell)
		}

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		if err := os.WriteFile(path, []byte(script), 0644); err != nil {
			return err
		}

		fmt.Printf("✅ Autocompletion installed for %s at %s\n", shell, path)
		fmt.Println("ℹ️  Restart your terminal or source the file to activate it.")
		return nil
	},
}

// detectShell returns the name of the current user's shell, such as "bash", "zsh", etc.
// It parses the $SHELL environment variable.
func detectShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return ""
	}
	parts := strings.Split(shell, "/")
	return parts[len(parts)-1] // get last part of path
}

func init() {
	// Agregas tu subcomando personalizado
	CompletionCmd.AddCommand(CompletionInstallCmd)

	// Agregas el comando built-in de cobra (bash/zsh/fish/powershell)
	CompletionCmd.AddCommand(GenBashCmd)
	CompletionCmd.AddCommand(GenZshCmd)
	CompletionCmd.AddCommand(GenFishCmd)
	CompletionCmd.AddCommand(GenPowerShellCmd)
}
