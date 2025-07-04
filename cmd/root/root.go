package root

import (
	"fmt"
	"os"
	"github.com/spf13/cobra"
)

var version = "0.1.0"

const banner = `
                             ██████╗ ███████╗██╗   ██╗ ██████╗  ██████╗ ███████╗████████╗
                             ██╔══██╗██╔════╝██║   ██║██╔═══██╗██╔═══██╗██╔════╝╚══██╔══╝
                             ██║  ██║█████╗  ██║   ██║██║   ██║██║   ██║███████╗   ██║   
                             ██║  ██║██╔══╝  ╚██╗ ██╔╝██║   ██║██║   ██║╚════██║   ██║   
                             ██████╔╝███████╗ ╚████╔╝ ╚██████╔╝╚██████╔╝███████║   ██║   
                             ╚═════╝ ╚══════╝  ╚═══╝   ╚═════╝  ╚═════╝ ╚══════╝   ╚═╝   

                                     dflow v%s - Git branching made simple
`

var rootCmd = &cobra.Command{
	Use:   "dflow",
	Short: "dflow is a Git branching flow manager for Devoost",
	Long:  "A CLI tool to manage Git feature/release/hotfix flows inspired by Git Flow",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf(banner, version)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
