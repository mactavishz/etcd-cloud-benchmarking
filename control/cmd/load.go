package cmd

import (
	"fmt"
	"os"
	"strconv"

	dg "csb/data-generator"

	"github.com/spf13/cobra"
)

var endpoints []string
var count int

var LoadCmd = &cobra.Command{
	Use:   "load [flags] <count>",
	Short: "Generate records and load them into the database to be used for benchmarking",
	Long:  "Generate number of records specified by <count> and load them into the database via the provided endpoints",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		if len(endpoints) == 0 {
			fmt.Println("Please provide at least one endpoint")
			os.Exit(1)
		} else {
			count, err = strconv.Atoi(args[0])
			if err != nil {
				fmt.Printf("Invalid count: %s\n", args[0])
			}
			fmt.Printf("Loading %d records into the database via the following endpoints: %v\n", count, endpoints)
			load_db(count)
		}
	},
}

func init() {
	LoadCmd.Flags().StringSliceVar(&endpoints, "endpoints", []string{"127.0.0.1:2379"}, "List of endpoints of the database to load data into")
}

func load_db(count int) {
	dataGenerator := dg.NewGenerator(GConfig.rg)
	data := dataGenerator.GenerateData(count)
	for k := range data {
		fmt.Println("Key: ", k)
	}
}
