package root

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	initcmd "github.com/yepizrene-devoost/dflow/cmd/init"
	startcmd "github.com/yepizrene-devoost/dflow/cmd/start"
	"github.com/yepizrene-devoost/dflow/cmd/utils"
)

var rootCmd = &cobra.Command{
	Use:   "dflow",
	Short: "dflow is a Git branching flow manager for Devoost",
	Long:  "A CLI tool to manage Git feature/release/hotfix flows inspired by Git Flow",

	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		utils.PrintBanner()
	},

	Run: func(cmd *cobra.Command, args []string) {
		cmd.SetArgs([]string{"--help"})
		cmd.Execute()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(initcmd.InitCmd)
	rootCmd.AddCommand(startcmd.StartCmd)

	// Personaliza la ayuda
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		cmd.Root().SetHelpFunc(nil) // Evita loop
		_ = cmd.Help()
	})

}
