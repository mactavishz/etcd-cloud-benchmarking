package main

import (
	"flag"
	"fmt"
	"net"
	"sync"
	"time"

	pb "csb/api/benchmarkpb"
	grpcserver "csb/client/grpc"
	runner "csb/client/runner"
	constants "csb/control/constants"
	"log"

	"google.golang.org/grpc"
)

func main() {
	port := flag.Int("p", constants.DEFAULT_GRPC_SERVER_PORT, "The grpc server port")
	flag.Parse()

	wg := &sync.WaitGroup{}
	readyChan := make(chan struct{})

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	gServer := grpc.NewServer()
	benchmarkServiceServer := grpcserver.NewBenchmarkServiceServer()
	pb.RegisterBenchmarkServiceServer(gServer, benchmarkServiceServer)
	go waitForConfigAndKeys(benchmarkServiceServer, readyChan)

	log.Printf("Benchmark client starting on port %d", *port)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := gServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	<-readyChan
	log.Printf("Benchmark client is ready")
	runBenchmarkKV(benchmarkServiceServer)
	wg.Wait()
}

func runBenchmarkKV(s *grpcserver.BenchmarkServiceServer) {
	config := s.GetConfig()

	readPercent, writePercent, err := runner.GetRWPercentages(config.WorkloadType)

	if err != nil {
		log.Fatalf("Invalid workload type %s, %v", config.WorkloadType, err)
	}

	runConfig := &runner.BenchmarkRunConfig{
		BenchctlConfig:   *config,
		ReadPercent:      readPercent,
		WritePercent:     writePercent,
		Keys:             s.GetKeys(),
		MetricsBatchSize: constants.DEFAULT_METRICS_BATCH_SIZE,
	}

	bench, err := runner.NewBenchmarkRunnerKV(runConfig)
	if err != nil {
		log.Fatalf("Failed to create benchmark runner: %v", err)
	}
	defer bench.Close()

	if err := bench.Run(); err != nil {
		log.Fatalf("Benchmark failed: %v", err)
	}

	log.Printf("Benchmark completed. Results:")
	for _, result := range bench.GetResults() {
		log.Printf("Clients: %d, P99 Latency: %v, Operations: %d, Errors: %d",
			result.NumClients,
			result.P99Latency,
			result.Operations,
			result.Errors,
		)
	}
}

func waitForConfigAndKeys(s *grpcserver.BenchmarkServiceServer, readyChan chan struct{}) {
	// Wait for config and keys
	for {
		if s.IsReady() {
			break
		} else {
			time.Sleep(500 * time.Millisecond)
		}
	}
	close(readyChan)
}
