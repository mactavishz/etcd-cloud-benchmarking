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
	Use:   "run [flags]",
	Short: "Run benchmarks",
	Long:  "Run benchmarks against the database, sends control message to benchmark client to start the benchmark",
	// Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		log.SetPrefix("[RUN] ")
		err := runBenchmark("localhost:50051", GConfig.GetKeyFilePath())
		if err != nil {
			log.Fatalf("Failed to run benchmark: %v", err)
		}
	},
}

func init() {
}

func runBenchmark(clientAddr string, keysFile string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Connect to each client and send keys
	conn, err := grpc.NewClient(clientAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()

	benchmarkServiceClient, err := grpcclient.NewBenchmarkServiceClient(conn, ctx)
	if err != nil {
		return err
	}

	stream := benchmarkServiceClient.GetStream()
	var keysReceived int32

	go func() {
		if err := benchmarkServiceClient.SendKeys(ctx, keysFile); err != nil {
			log.Fatalf("Failed to send keys: %v", err)
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
		default:
			log.Printf("Unknown message type from server")
		}
	}
}
