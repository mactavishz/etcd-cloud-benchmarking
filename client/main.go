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

func main() {
	port := flag.Int("p", constants.DEFAULT_GRPC_SERVER_PORT, "The GRPC server port")
	flag.Parse()

	// wg := &sync.WaitGroup{}
	readyChan := make(chan struct{})
	sigChan := make(chan os.Signal, 1)
	termChan := make(chan struct{})
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	gServer := grpc.NewServer()
	benchmarkServiceServer := grpcserver.NewBenchmarkServiceServer(gServer, termChan)
	pb.RegisterBenchmarkServiceServer(gServer, benchmarkServiceServer)
	go waitUntilReady(benchmarkServiceServer, readyChan)

	go func() {
		log.Printf("Benchmark client's GRPC Server starting on port %d", *port)
		if err := gServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
		log.Println("GRPC Server stopped")
	}()

	go func() {
		<-readyChan
		benchCfg := benchmarkServiceServer.GetConfig()
		if benchCfg.Scenario == constants.SCENARIO_KV_STORE {
			runBenchmarkKV(benchmarkServiceServer)
		} else {
			runBenchmarkLockService(benchmarkServiceServer)
		}
		err = benchmarkServiceServer.SendCTRLMessage(&pb.CTRLMessage{
			Payload: &pb.CTRLMessage_BenchmarkFinished{},
		})
		if err != nil {
			log.Printf("Failed to send benchmark finished message: %v", err)
		}
	}()

	for {
		select {
		case <-sigChan:
			fmt.Println()
			log.Println("Manually shutting down server ...")
			gServer.Stop()
			return
		case <-termChan:
			log.Println("Gracefully shutting down server ...")
			gServer.GracefulStop()
			return
		}
	}
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

	log.Printf("Benchmark completed. Overall results:")
	for _, result := range bench.GetResults() {
		log.Printf("Clients: %d, P99 Latency: %v, Operations: %d, Errors: %d",
			result.NumClients,
			result.P99Latency,
			result.Operations,
			result.Errors,
		)
	}
}

func runBenchmarkLockService(s *grpcserver.BenchmarkServiceServer) {
}

func waitUntilReady(s *grpcserver.BenchmarkServiceServer, readyChan chan struct{}) {
	log.Println("Waiting for config and keys to start running benchmarks ...")
	// Wait for config and keys
	for {
		if s.IsReady() {
			log.Println("Ready to run benchmarks")
			break
		} else {
			time.Sleep(250 * time.Millisecond)
		}
	}
	close(readyChan)
}
