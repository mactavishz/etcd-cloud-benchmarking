package cmd

import (
	"context"
	pb "csb/api/benchmarkpb"
	grpcclient "csb/control/grpc"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

var RunCmd = &cobra.Command{
	Use:   "run [flags] <client_addr>",
	Short: "Run benchmarks",
	Long:  "Run benchmarks against the database, sends control message to benchmark client to start the benchmark. The <client_addr> is the ip address along with port of the benchmark client",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		log.SetPrefix("[RUN] ")
		if GConfig.ctlConfig == nil {
			fmt.Println("Config not found, please run 'benchctl config init' first")
			os.Exit(1)
		}
		err := runBenchmark(args[0])
		if err != nil {
			log.Fatalf("Error from the benchmark run: %v", err)
		}
		log.Println("Benchmark terminated")
	},
}

func runBenchmark(clientAddr string) error {
	termChan := make(chan struct{})
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithConnectParams(grpc.ConnectParams{
			MinConnectTimeout: 30 * time.Second,
		}),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second, // send pings every 30 seconds if there is no activity
			Timeout:             60 * time.Second, // wait 60 seconds for ping responses
			PermitWithoutStream: true,             // allow pings even without active streams
		}),
	}

	// Connect to each client and send keys
	conn, err := grpc.NewClient(clientAddr, dialOpts...)
	if err != nil {
		return err
	}

	defer conn.Close()

	ctx := context.Background()
	benchmarkServiceClient, err := grpcclient.NewBenchmarkServiceClient(conn, ctx)
	if err != nil {
		return err
	}

	stream := benchmarkServiceClient.GetStream()

	// wait for connection to be ready
	log.Println("Waiting for connection to be ready")
	for {
		if conn.GetState() != connectivity.Ready {
			time.Sleep(100 * time.Millisecond)
		} else {
			log.Println("Connection is ready")
			break
		}
	}

	// send config file to the client
	go func() {
		if err := benchmarkServiceClient.SendConfigFile(ctx, GConfig.GetConfigFilePath()); err != nil {
			log.Printf("Failed to send config file: %v", err)
			terminate(stream, termChan)
		}
	}()

	for {
		var err error
		select {
		case <-sigChan:
			terminate(stream, termChan)
		case <-termChan:
			return err
		default:
			res, err := stream.Recv()

			if err == io.EOF {
				// End of stream
				log.Println("GRPC stream closed by server")
				close(termChan)
				return nil
			}

			if err != nil {
				log.Printf("Error receiving from server: %v", err)
				close(termChan)
				return err
			}

			// Handle server responses
			switch payload := res.Payload.(type) {
			case *pb.CTRLMessage_BenchmarkStatus:
				log.Printf("Benchmark status: %v", payload.BenchmarkStatus.Status)
			case *pb.CTRLMessage_ConfigFileResponse:
				configReceived := payload.ConfigFileResponse.Success
				log.Printf("Config file sent: %v", configReceived)
			case *pb.CTRLMessage_BenchmarkFinished:
				log.Println("Benchmark run finished")
				terminate(stream, termChan)
			default:
				log.Printf("Unknown message type from server")
			}
		}
	}
}

func terminate(stream pb.BenchmarkService_CTRLStreamClient, termChan chan struct{}) {
	log.Println("Terminating benchmark")
	err := stream.Send(&pb.CTRLMessage{
		Payload: &pb.CTRLMessage_Shutdown{},
	})
	if err != nil {
		log.Printf("Failed to send shutdown message: %v", err)
	}
	close(termChan)
}
