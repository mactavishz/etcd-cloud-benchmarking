package cmd

import (
	// "fmt"
	"github.com/spf13/cobra"
	"os"
)

var rootCmd = &cobra.Command{
	Use:   "benchctl",
	Short: "Benchctl is a CLI tool for managing benchmarking tasks",
	Long:  "A CLI tool for managing benchmarking tasks including preparing, running, monitoring, and collecting results",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			os.Exit(0)
		}
	},
}

func init() {
	rootCmd.AddCommand(LoadCmd)
}

func Execute() error {
	return rootCmd.Execute()
}
