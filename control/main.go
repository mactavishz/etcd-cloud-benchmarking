package main

import (
	"fmt"
	"os"

	cmd "git.tu-berlin.de/mactavishz/csb-project-ws2425/control/cmd"
)

func main() {
	// CLI
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
