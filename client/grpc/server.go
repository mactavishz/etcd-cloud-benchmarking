package grpcserver

import (
	pb "csb/api/benchmarkpb"
	"io"
	"log"
	"sync"
)

type BenchmarkServiceServer struct {
	pb.UnimplementedBenchmarkServiceServer
	keys            []string
	receivedKeys    int32
	allKeysReceived bool
	keysMu          sync.RWMutex
}

func NewBenchmarkServiceServer() *BenchmarkServiceServer {
	return &BenchmarkServiceServer{
		keys: make([]string, 0),
	}
}

func (s *BenchmarkServiceServer) CTRLStream(stream pb.BenchmarkService_CTRLStreamServer) error {
	for {
		req, err := stream.Recv()

		if err == io.EOF {
			// client closes the stream
			return nil
		}

		if err != nil {
			log.Printf("Error receiving grpc message from client: %v", err)
			return err
		}

		switch payload := req.Payload.(type) {
		case *pb.CTRLMessage_KeyBatch:
			s.receiveKeys(payload.KeyBatch.GetKeys(), payload.KeyBatch.GetIsLastBatch())
			log.Printf("Received batch of %d keys. Total keys: %d", len(payload.KeyBatch.Keys), s.receivedKeys)
			response := &pb.CTRLMessage{
				Payload: &pb.CTRLMessage_KeyBatchResponse{
					KeyBatchResponse: &pb.KeyBatchResponse{
						TotalKeysReceived: s.receivedKeys,
					},
				},
			}
			err = stream.Send(response)
		}

		if err != nil {
			log.Printf("Error sending grpc message to client: %v", err)
			return err
		}
	}
}

func (s *BenchmarkServiceServer) receiveKeys(keys []string, isLastBatch bool) {
	s.keysMu.Lock()
	s.keys = append(s.keys, keys...)
	s.receivedKeys += int32(len(keys))
	s.allKeysReceived = isLastBatch
	s.keysMu.Unlock()
}

// GetKeys returns the stored keys (for benchmark use)
func (s *BenchmarkServiceServer) GetKeys() []string {
	s.keysMu.RLock()
	defer s.keysMu.RUnlock()
	return append([]string{}, s.keys...)
}
