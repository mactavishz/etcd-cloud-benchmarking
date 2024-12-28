package main

import (
	"flag"
	"fmt"
	"net"

	// "go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	// clientv3 "go.etcd.io/etcd/client/v3"
	pb "csb/api/benchmarkpb"
	grpcserver "csb/client/grpc"
	"log"

	"google.golang.org/grpc"
)

func main() {
	port := flag.Int("port", 50051, "The server port")
	flag.Parse()

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	gServer := grpc.NewServer()
	benchmarkServiceServer := grpcserver.NewBenchmarkServiceServer()
	pb.RegisterBenchmarkServiceServer(gServer, benchmarkServiceServer)

	log.Printf("Benchmark client starting on port %d", *port)
	if err := gServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
