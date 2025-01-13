package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "csb/api/benchmarkpb"
	grpcserver "csb/client/grpc"
	runner "csb/client/runner"
	constants "csb/control/constants"
	"log"

	"google.golang.org/grpc"
)

var logger *runner.Logger

func init() {
	var err error
	logger, err = runner.NewLogger("run.log")
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
		if benchCfg.Scenario == constants.SCENARIO_KV_STORE {
			logger.Println("Running KV store benchmark ...")
			runBenchmarkKV(benchmarkServiceServer)
		} else {
			logger.Println("Running Lock service benchmark ...")
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
		logger.Printf("Failed to create benchmark runner: %v", err)
		exit(1)
	}
	defer bench.Close()

	if err := bench.Run(); err != nil {
		logger.Printf("Benchmark failed: %v", err)
		exit(1)
	}

	log.Printf("Benchmark completed. Overall results:")
	for _, result := range bench.GetResults() {
		logger.Printf("Clients: %d, P99 Latency: %v, Operations: %d, Errors: %d",
			result.NumClients,
			result.P99Latency,
			result.Operations,
			result.Errors,
		)
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
		logger.Printf("Failed to create benchmark runner: %v", err)
		exit(1)
	}
	defer bench.Close()

	if err := bench.Run(); err != nil {
		logger.Printf("Benchmark failed: %v", err)
		exit(1)
	}

	logger.Printf("Benchmark completed. Overall results:")
	for _, result := range bench.GetResults() {
		logger.Printf("Clients: %d, P99 Latency: %v, Operations: %d, Errors: %d",
			result.NumClients,
			result.P99Latency,
			result.Operations,
			result.Errors,
		)
	}
}

func waitUntilReady(s *grpcserver.BenchmarkServiceServer, readyChan chan struct{}) {
	logger.Println("Waiting for config and keys to start running benchmarks ...")
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
