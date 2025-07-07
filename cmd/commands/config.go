package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yepizrene-devoost/dflow/cmd/utils"
	"github.com/yepizrene-devoost/dflow/pkg/validators"
)

var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage dflow configuration for this project",
	Long: `Manage local project-specific configuration used by dflow commands, such as:

  - Author name and email used in changelog footers.
  - Listing all project-level dflow git configs.

  Examples:
    dflow config set-author <your name> --email=<your email>
    dflow config get-author
    dflow config list

  These settings are stored using the local .git config and are specific to each project.`,
}

var setAuthorCmd = &cobra.Command{
	Use:   "set-author [name]",
	Short: "Set project-local author name and email for dflow",
	Args:  cobra.MaximumNArgs(1),
	RunE: validators.WithChecks(false, func(cmd *cobra.Command, args []string) error {
		var name string
		var email string
		var err error

		// get name from args or prompt
		if len(args) > 0 {
			name = args[0]
		} else {
			fmt.Print("👤 Enter author name: ")
			reader := bufio.NewReader(os.Stdin)
			name, err = reader.ReadString('\n')

			if err != nil {
				return fmt.Errorf("failed to read author name: %w", err)
			}

			name = strings.TrimSpace(name)
		}

		// get email from flag or prompt input
		email, _ = cmd.Flags().GetString("email")
		if email == "" {
			fmt.Print("📧 Enter author email: ")
			reader := bufio.NewReader(os.Stdin)
			email, err = reader.ReadString('\n')

			if err != nil {
				return fmt.Errorf("failed to read email: %w", err)
			}

			email = strings.TrimSpace(email)
		}

		// save local config
		if err = exec.Command("git", "config", "dflow.author", name).Run(); err != nil {
			return fmt.Errorf("failed to set dflow.author: %w", err)
		}

		if err = exec.Command("git", "config", "dflow.email", email).Run(); err != nil {
			return fmt.Errorf("failed to set dflow.email: %w", err)
		}

		utils.Success("Author and email saved to project-local git config")

		return nil
	}),
}

var getAuthorCmd = &cobra.Command{
	Use:   "get-author",
	Short: "Show project-local dflow author and email",
	RunE: validators.WithChecks(false, func(cmd *cobra.Command, args []string) error {
		author, err1 := exec.Command("git", "config", "--get", "dflow.author").Output()
		email, err2 := exec.Command("git", "config", "--get", "dflow.email").Output()

		if err1 != nil || err2 != nil {
			utils.Error("Author or email not set. Use `dflow config set-author`")

			return nil
		}

		fmt.Printf("👤 Author: %s\n", strings.TrimSpace(string(author)))
		fmt.Printf("📧 Email: %s\n", strings.TrimSpace(string(email)))

		return nil
	}),
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all dflow configuration values for this project",
	RunE: validators.WithChecks(false, func(cmd *cobra.Command, args []string) error {
		output, err := exec.Command("git", "config", "--get-regexp", "^dflow\\.").Output()

		if err != nil {
			utils.Warn("No dflow configuration found in this project.")

			return nil
		}

		fmt.Print(string(output))

		return nil
	}),
}

func init() {
	setAuthorCmd.Flags().String("email", "", "Email for changelogs (required)")

	ConfigCmd.AddCommand(setAuthorCmd)
	ConfigCmd.AddCommand(getAuthorCmd)
	ConfigCmd.AddCommand(listCmd)

	ConfigCmd.Run = func(cmd *cobra.Command, args []string) {
		if err := cmd.Help(); err != nil {
			fmt.Fprintf(os.Stderr, "Error showing help: %v\n", err)
			os.Exit(1)
		}
	}
}
