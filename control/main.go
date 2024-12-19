package main

import (
	cmd "csb/control/cmd"
	"fmt"

	// "github.com/spf13/cobra"
	"os"
)

func main() {
	// CLI
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
