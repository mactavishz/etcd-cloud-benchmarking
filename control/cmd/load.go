package cmd

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"sync"

	"context"
	constants "csb/control/constants"
	dg "csb/data-generator"
	"time"

	"github.com/spf13/cobra"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	dialTimeout    = 5 * time.Second
	requestTimeout = 10 * time.Second
)

var LoadCmd = &cobra.Command{
	Use:   "load [flags]",
	Short: "Generate records and load them into the database to be used for benchmarking",
	Long:  "Generate number of records specified by NumKeys in the config and load them into the database via the provided Endpoints in the config",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		log.SetPrefix("[LOAD] ")
		if GConfig.ctlConfig == nil {
			fmt.Println("Config not found, please run 'benchctl config init' first")
			os.Exit(1)
		}
		count := GConfig.ctlConfig.NumKeys
		endpoints := GConfig.ctlConfig.Endpoints
		if len(endpoints) == 0 {
			log.Fatalln("Please provide at least one endpoint")
		} else {
			log.Printf("Loading %d records into the database via the following endpoints: %v\n", count, endpoints)
			load_db()
		}
	},
}

func init() {
	// LoadCmd.Flags().StringSliceVar(&endpoints, "endpoints", []string{"127.0.0.1:2379"}, "List of endpoints of the database to load data into")
}

func load_db() {
	dataGenerator := dg.NewGenerator(GConfig.rg)
	data, err := dataGenerator.GenerateData(GConfig.ctlConfig.NumKeys, GConfig.ctlConfig.KeySize, GConfig.ctlConfig.ValueSize)

	if err != nil {
		log.Fatalf("Failed to generate data: %v\n", err)
	}

	keys := make([]string, 0, len(data))
	dbClient, err := clientv3.New(clientv3.Config{
		Endpoints:   GConfig.ctlConfig.Endpoints,
		DialTimeout: requestTimeout,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer dbClient.Close()

	var wg sync.WaitGroup

	for key, value := range data {
		keys = append(keys, key)
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
	log.Println("Saving keys in the config folder")
	err = os.WriteFile(path.Join(GConfig.ctlConfigPath, constants.DEFAULT_KEY_FILE), []byte(strings.Join(keys, "\n")), 0644)
	if err != err {
		log.Fatalf("Error saving keys: %v\n", err)
	}
	log.Println("Keys saved successfully")
	log.Println("Data loaded successfully")
}
