package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var ConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage dflow configuration for this project",
	Long: `Manage local project-specific configuration used by dflow commands, such as:

  - Author name and email used in changelog footers.
  - Listing all project-level dflow git configs.

  Examples:
    dflow config set-author <your name>
    dflow config set-author --email=<your email>
    dflow config get-author
    dflow config list

  These settings are stored using the local .git config and are specific to each project.`,
}

var setAuthorCmd = &cobra.Command{
	Use:   "set-author [name]",
	Short: "Set project-local author name and email for dflow",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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

		fmt.Println("✅ Author and email saved to project-local git config")
		return nil
	},
}

var getAuthorCmd = &cobra.Command{
	Use:   "get-author",
	Short: "Show project-local dflow author and email",
	RunE: func(cmd *cobra.Command, args []string) error {
		author, err1 := exec.Command("git", "config", "--get", "dflow.author").Output()
		email, err2 := exec.Command("git", "config", "--get", "dflow.email").Output()

		if err1 != nil || err2 != nil {
			fmt.Println("❌ Author or email not set. Use `dflow config set-author`.")
			return nil
		}

		fmt.Printf("Author: %sEmail: %s", author, email)
		return nil
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all dflow configuration values for this project",
	RunE: func(cmd *cobra.Command, args []string) error {
		output, err := exec.Command("git", "config", "--get-regexp", "^dflow\\.").Output()
		if err != nil {
			fmt.Println("⚠️ No dflow configuration found in this project.")
			return nil
		}
		fmt.Print(string(output))
		return nil
	},
}

func init() {
	setAuthorCmd.Flags().String("email", "", "Email for changelogs (required)")

	ConfigCmd.AddCommand(setAuthorCmd)
	ConfigCmd.AddCommand(getAuthorCmd)
	ConfigCmd.AddCommand(listCmd)

	ConfigCmd.Run = func(cmd *cobra.Command, args []string) {
		cmd.Help()
	}
}
