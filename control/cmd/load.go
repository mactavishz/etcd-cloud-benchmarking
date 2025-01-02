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
	"go.uber.org/zap"
)

const (
	dialTimeout    = 5 * time.Second
	requestTimeout = 10 * time.Second
)

var LoadCmd = &cobra.Command{
	Use:   "load [flags]",
	Short: "Generate records and load them into the database to be used for benchmarking",
	Long:  "Generate number of records specified by NumKeys in the config and load them into the database via the provided Endpoints in the config",
	Args:  cobra.NoArgs,
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

	log.Println("Number of key-value paris generated: ", len(data))

	if GConfig.ctlConfig.NumKeys != len(data) {
		log.Fatalf("Failed to generate the required number of key-value pairs due to collision: %d\n", GConfig.ctlConfig.NumKeys)
	}

	if err != nil {
		log.Fatalf("Failed to generate data: %v\n", err)
	}

	keys := make([]string, 0, len(data))

	// Worker pool size
	const workerCount = 100
	tasks := make(chan struct {
		key   string
		value []byte
	}, workerCount)

	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dbClient, err := clientv3.New(clientv3.Config{
				Endpoints:   GConfig.ctlConfig.Endpoints,
				DialTimeout: requestTimeout,
				Logger:      zap.NewNop(),
			})
			if err != nil {
				log.Fatal(err)
			}
			defer dbClient.Close()
			for task := range tasks {
				func(t struct {
					key   string
					value []byte
				}) {
					ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
					defer cancel()
					_, err := dbClient.Put(ctx, t.key, string(t.value))
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
				}(task)
			}
		}()
	}

	// Send tasks to workers
	for key, value := range data {
		keys = append(keys, key)
		tasks <- struct {
			key   string
			value []byte
		}{key, value}
	}
	close(tasks) // Close the task channel to signal workers to stop

	wg.Wait()
	log.Println("Saving keys in the config folder")
	err = os.WriteFile(path.Join(GConfig.ctlConfigPath, constants.DEFAULT_KEY_FILE), []byte(strings.Join(keys, "\n")), 0644)
	if err != err {
		log.Fatalf("Error saving keys: %v\n", err)
	}
	log.Println("Keys saved successfully")
	log.Println("Data loaded successfully")
}
