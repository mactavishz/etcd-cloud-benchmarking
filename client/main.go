package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	pb "csb/api/benchmarkpb"
	grpcserver "csb/client/grpc"
	lg "csb/client/logger"
	runner "csb/client/runner"
	constants "csb/control/constants"
	dg "csb/data-generator"
	"log"

	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"

	"google.golang.org/grpc"
)

var logger *lg.Logger

func init() {
	var err error
	logger, err = lg.NewLogger("run.log")
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
}

func exit(code int) {
	logger.Close()
	os.Exit(code)
}

func main() {
	defer exit(0)
	port := flag.Int("p", constants.DEFAULT_GRPC_SERVER_PORT, "The GRPC server port")
	flag.Parse()

	// wg := &sync.WaitGroup{}
	readyChan := make(chan struct{})
	sigChan := make(chan os.Signal, 1)
	termChan := make(chan struct{})
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		logger.Printf("failed to listen: %v", err)
		exit(1)
	}

	gServer := grpc.NewServer()
	benchmarkServiceServer := grpcserver.NewBenchmarkServiceServer(gServer, logger, termChan)
	pb.RegisterBenchmarkServiceServer(gServer, benchmarkServiceServer)
	go waitUntilReady(benchmarkServiceServer, readyChan)

	go func() {
		logger.Printf("Benchmark client's GRPC Server starting on port %d", *port)
		if err := gServer.Serve(lis); err != nil {
			logger.Printf("failed to serve: %v", err)
			exit(1)
		}
		logger.Println("GRPC Server stopped")
	}()

	go func() {
		<-readyChan
		benchCfg := benchmarkServiceServer.GetConfig()
		logger.Printf("Generating and loading data into the database ...")
		benchmarkServiceServer.SendBenchmarkStatus("Start generating and loading data into the database")
		load_db(benchmarkServiceServer)
		if benchCfg.Scenario == constants.SCENARIO_KV_STORE {
			logger.Println("Running KV store benchmark ...")
			benchmarkServiceServer.SendBenchmarkStatus("Start running KV store benchmark ...")
			runBenchmarkKV(benchmarkServiceServer)
		} else {
			logger.Println("Running Lock service benchmark ...")
			benchmarkServiceServer.SendBenchmarkStatus("Start running Lock service benchmark")
			runBenchmarkLockService(benchmarkServiceServer)
		}
		err = benchmarkServiceServer.SendCTRLMessage(&pb.CTRLMessage{
			Payload: &pb.CTRLMessage_BenchmarkFinished{},
		})
		if err != nil {
			logger.Printf("Failed to send benchmark finished message: %v", err)
		}
	}()

	for {
		select {
		case <-sigChan:
			logger.Println("Manually shutting down server ...")
			gServer.Stop()
			return
		case <-termChan:
			logger.Println("Gracefully shutting down server ...")
			gServer.GracefulStop()
			return
		}
	}
}

func load_db(s *grpcserver.BenchmarkServiceServer) {
	var (
		dialTimeout    = 60 * time.Second
		requestTimeout = 30 * time.Second
	)
	ctlConfig := s.GetConfig()
	rg := rand.New(rand.NewSource(ctlConfig.Seed))
	dataGenerator := dg.NewGenerator(rg)
	s.SendBenchmarkStatus("Generating synthetic data ...")
	data, err := dataGenerator.GenerateData(ctlConfig.NumKeys, ctlConfig.KeySize, ctlConfig.ValueSize)

	logger.Println("Number of key-value paris generated: ", len(data))
	s.SendBenchmarkStatus(fmt.Sprintf("Number of key-value pairs generated: %d", len(data)))

	if ctlConfig.NumKeys != len(data) {
		logger.Printf("Failed to generate the required number of key-value pairs due to collision: %d\n", ctlConfig.NumKeys)
	}

	if err != nil {
		logger.Printf("Failed to generate data: %v\n", err)
		exit(1)
	}

	keys := make([]string, 0, len(data))

	// Worker pool size
	const workerCount = 100
	tasks := make(chan struct {
		key   string
		value []byte
	}, workerCount)

	var wg sync.WaitGroup

	logger.Println("Loading data into the database ...")
	s.SendBenchmarkStatus("Loading data into the database")
	// Start workers
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dbClient, err := clientv3.New(clientv3.Config{
				Endpoints:   ctlConfig.Endpoints,
				DialTimeout: dialTimeout,
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
							logger.Printf("ctx is canceled by another routine: %v\n", err)
						case context.DeadlineExceeded:
							logger.Printf("ctx is attached with a deadline is exceeded: %v\n", err)
						case rpctypes.ErrEmptyKey:
							logger.Printf("client-side error: %v\n", err)
						default:
							logger.Printf("bad cluster endpoints, which are not etcd servers: %v\n", err)
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
	s.SetKeys(keys)
	logger.Println("Saving generated keys in a file")
	err = os.WriteFile(constants.DEFAULT_KEY_FILE, []byte(strings.Join(keys, "\n")), 0644)
	if err != err {
		logger.Printf("Error saving keys: %v\n", err)
		exit(1)
	}
	logger.Printf("Data loaded successfully")
	s.SendBenchmarkStatus("Synthetic data generated and loaded successfully")
}

func runBenchmarkKV(s *grpcserver.BenchmarkServiceServer) {
	config := s.GetConfig()

	readPercent, writePercent, err := runner.GetRWPercentages(config.WorkloadType)

	if err != nil {
		logger.Printf("Invalid workload type %s, %v", config.WorkloadType, err)
		exit(1)
	}

	runConfig := &runner.BenchmarkRunConfig{
		BenchctlConfig:   *config,
		ReadPercent:      readPercent,
		WritePercent:     writePercent,
		Keys:             s.GetKeys(),
		MetricsBatchSize: constants.DEFAULT_METRICS_BATCH_SIZE,
	}

	bench, err := runner.NewBenchmarkRunnerKV(runConfig, logger)
	if err != nil {
		s.SendBenchmarkStatus("Failed to create benchmark runner")
		logger.Printf("Failed to create benchmark runner: %v", err)
		exit(1)
	}
	defer bench.Close()

	if err := bench.Run(s); err != nil {
		s.SendBenchmarkStatus("Benchmark failed")
		logger.Printf("Benchmark failed: %v", err)
		exit(1)
	}

	log.Printf("Benchmark completed. Overall results:")
	s.SendBenchmarkStatus("Benchmark completed. Overall results:")
	for _, result := range bench.GetResults() {
		resultStr := fmt.Sprintf("Step with #Clients: %d, P99 Latency: %v, #Operations: %d, #Errors: %d", result.NumClients, result.P99Latency, result.Operations, result.Errors)
		logger.Println(resultStr)
		s.SendBenchmarkStatus(resultStr)
	}
}

func runBenchmarkLockService(s *grpcserver.BenchmarkServiceServer) {
	config := s.GetConfig()

	runConfig := &runner.BenchmarkRunConfig{
		BenchctlConfig:   *config,
		Keys:             s.GetKeys(),
		MetricsBatchSize: constants.DEFAULT_METRICS_BATCH_SIZE,
	}

	bench, err := runner.NewBenchmarkRunnerLock(runConfig, logger)
	if err != nil {
		s.SendBenchmarkStatus("Failed to create benchmark runner")
		logger.Printf("Failed to create benchmark runner: %v", err)
		exit(1)
	}
	defer bench.Close()

	if err := bench.Run(s); err != nil {
		s.SendBenchmarkStatus("Benchmark failed")
		logger.Printf("Benchmark failed: %v", err)
		exit(1)
	}

	log.Printf("Benchmark completed. Overall results:")
	s.SendBenchmarkStatus("Benchmark completed. Overall results:")
	for _, result := range bench.GetResults() {
		resultStr := fmt.Sprintf("Step with #Clients: %d, P99 Latency: %v, #Operations: %d, #Errors: %d", result.NumClients, result.P99Latency, result.Operations, result.Errors)
		logger.Println(resultStr)
		s.SendBenchmarkStatus(resultStr)
	}
}

func waitUntilReady(s *grpcserver.BenchmarkServiceServer, readyChan chan struct{}) {
	logger.Println("Waiting for config to start running benchmarks ...")
	// Wait for config and keys
	for {
		if s.IsReady() {
			logger.Println("Ready to run benchmarks")
			break
		} else {
			time.Sleep(250 * time.Millisecond)
		}
	}
	close(readyChan)
}
