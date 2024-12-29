package main

import (
	"flag"
	"fmt"
	"net"
	"sync"
	"time"

	pb "csb/api/benchmarkpb"
	grpcserver "csb/client/grpc"
	"log"

	"google.golang.org/grpc"
)

const DEFAULT_PORT = 50051

func main() {
	port := flag.Int("p", DEFAULT_PORT, "The grpc server port")
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
	wg.Wait()
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
