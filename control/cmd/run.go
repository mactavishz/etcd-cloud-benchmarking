package cmd

import (
	"context"
	pb "csb/api/benchmarkpb"
	grpcclient "csb/control/grpc"
	"io"
	"log"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var RunCmd = &cobra.Command{
	Use:   "run [flags] <client_addr>",
	Short: "Run benchmarks",
	Long:  "Run benchmarks against the database, sends control message to benchmark client to start the benchmark. The <client_addr> is the ip address along with port of the benchmark client",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		log.SetPrefix("[RUN] ")
		err := runBenchmark(args[0], GConfig.GetKeyFilePath())
		if err != nil {
			log.Fatalf("Failed to run benchmark: %v", err)
		}
	},
}

var keysReceived int32
var configReceived bool

func init() {
}

func runBenchmark(clientAddr string, keysFile string) error {
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithConnectParams(grpc.ConnectParams{
			MinConnectTimeout: 15 * time.Second,
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

	// send loaded keys to the client
	go func() {
		if err := benchmarkServiceClient.SendKeys(ctx, keysFile); err != nil {
			log.Fatalf("Failed to send keys: %v", err)
		}
	}()

	// send config file to the client
	go func() {
		if err := benchmarkServiceClient.SendConfigFile(ctx, GConfig.GetConfigFilePath()); err != nil {
			log.Fatalf("Failed to send config file: %v", err)
		}
	}()

	for {
		res, err := stream.Recv()

		if err == io.EOF {
			// End of stream
			return nil
		}

		if err != nil {
			log.Fatalf("Error receiving from server: %v", err)
		}

		// Handle server responses
		switch payload := res.Payload.(type) {
		case *pb.CTRLMessage_KeyBatchResponse:
			keysReceived = payload.KeyBatchResponse.TotalKeysReceived
			log.Printf("%d keys sent", keysReceived)
		case *pb.CTRLMessage_ConfigFileResponse:
			configReceived = payload.ConfigFileResponse.Success
			log.Printf("Config file received: %v", configReceived)
		default:
			log.Printf("Unknown message type from server")
		}
	}
}
