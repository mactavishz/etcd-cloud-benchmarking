package cmd

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	"context"
	dg "csb/data-generator"
	"time"

	"github.com/spf13/cobra"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var endpoints []string
var count int

const (
	dialTimeout    = 5 * time.Second
	requestTimeout = 10 * time.Second
)

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
			load_db(count, endpoints)
		}
	},
}

func init() {
	LoadCmd.Flags().StringSliceVar(&endpoints, "endpoints", []string{"127.0.0.1:2379"}, "List of endpoints of the database to load data into")
}

func load_db(count int, endpoints []string) {
	dataGenerator := dg.NewGenerator(GConfig.rg)
	data := dataGenerator.GenerateData(count)

	dbClient, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: requestTimeout,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer dbClient.Close()

	var wg sync.WaitGroup

	for key, value := range data {
		wg.Add(1)
		go func(key string, value []byte) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			defer cancel()
			_, err := dbClient.Put(ctx, key, string(value))
			if err != nil {
				switch err {
				case context.Canceled:
					log.Fatalf("ctx is canceled by another routine: %v\n", err)
				case context.DeadlineExceeded:
					log.Fatalf("ctx is attached with a deadline is exceeded: %v\n", err)
				case rpctypes.ErrEmptyKey:
					log.Fatalf("client-side error: %v\n", err)
				default:
					log.Fatalf("bad cluster endpoints, which are not etcd servers: %v\n", err)
				}
			}
		}(key, value)
	}
	wg.Wait()
	fmt.Println("Data loaded successfully")
}
