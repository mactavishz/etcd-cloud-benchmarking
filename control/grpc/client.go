package grpcclient

import (
	"context"
	pb "csb/api/benchmarkpb"
	"io"
	"os"
	"strings"

	"google.golang.org/grpc"
)

const (
	BATCH_SIZE = 1000
)

type BenchmarkServiceClient struct {
	client pb.BenchmarkServiceClient
	stream pb.BenchmarkService_CTRLStreamClient
}

func NewBenchmarkServiceClient(conn *grpc.ClientConn, ctx context.Context) (*BenchmarkServiceClient, error) {
	client := pb.NewBenchmarkServiceClient(conn)
	stream, err := client.CTRLStream(ctx)
	if err != nil {
		return nil, err
	}
	return &BenchmarkServiceClient{
		client: client,
		stream: stream,
	}, nil
}

func (c *BenchmarkServiceClient) GetStream() pb.BenchmarkService_CTRLStreamClient {
	return c.stream
}

func (c *BenchmarkServiceClient) SendKeys(ctx context.Context, keysFile string) error {
	// Open the keys file
	file, err := os.Open(keysFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// Read all keys at once
	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	// Split keys into lines
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	totalKeys := len(lines)

	// Send keys in batches
	for start := 0; start < totalKeys; start += BATCH_SIZE {
		end := start + BATCH_SIZE
		if end > totalKeys {
			end = totalKeys
		}

		batch := lines[start:end]
		isLastBatch := end == totalKeys

		request := &pb.CTRLMessage{
			Payload: &pb.CTRLMessage_KeyBatch{
				KeyBatch: &pb.KeyBatch{
					Keys:        batch,
					IsLastBatch: isLastBatch,
				},
			},
		}

		err := c.stream.Send(request)
		if err != nil {
			return err
		}
	}
	return nil

}
