package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

// Current version of the CLI tool
var cliVersion = "v1.0.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of the CLI",
	Long:  `All software has versions. This is CLI's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "CLI Version %s\n", cliVersion) // Ensure output goes to cmd.OutOrStdout()
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
