package main

import (
	cmd "csb/control/cmd"
	"fmt"
	"os"
	// "github.com/spf13/cobra"
)

func main() {
	// CLI
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
