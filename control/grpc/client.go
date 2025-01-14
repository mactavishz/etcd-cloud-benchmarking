package grpcclient

import (
	"context"
	pb "csb/api/benchmarkpb"
	"io"
	"os"

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

func (c *BenchmarkServiceClient) SendConfigFile(ctx context.Context, configFile string) error {
	// Open the config file
	file, err := os.Open(configFile)
	if err != nil {
		return err
	}
	defer file.Close()
	// Read all config file at once
	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	request := &pb.CTRLMessage{
		Payload: &pb.CTRLMessage_ConfigFile{
			ConfigFile: &pb.ConfigFile{
				Content: data,
			},
		},
	}
	err = c.stream.Send(request)
	return err
}
