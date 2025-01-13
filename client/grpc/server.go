package grpcserver

import (
	pb "csb/api/benchmarkpb"
	runner "csb/client/runner"
	config "csb/control/config"
	"encoding/json"
	"errors"
	"io"
	"sync"

	"google.golang.org/grpc"
)

type BenchmarkServiceServer struct {
	pb.UnimplementedBenchmarkServiceServer
	keys            []string
	receivedKeys    int32
	allKeysReceived bool
	keysMu          sync.RWMutex
	ctlConfig       *config.BenchctlConfig
	grpcServer      *grpc.Server
	termChan        chan struct{}
	currStream      pb.BenchmarkService_CTRLStreamServer
	streamMu        sync.Mutex
	logger          *runner.Logger
}

func NewBenchmarkServiceServer(grpcserver *grpc.Server, logger *runner.Logger, termChan chan struct{}) *BenchmarkServiceServer {
	return &BenchmarkServiceServer{
		keys:       make([]string, 0),
		grpcServer: grpcserver,
		termChan:   termChan,
		logger:     logger,
	}
}

func (s *BenchmarkServiceServer) IsReady() bool {
	return s.allKeysReceived && s.ctlConfig != nil
}

func (s *BenchmarkServiceServer) GetConfig() *config.BenchctlConfig {
	return s.ctlConfig
}

func (s *BenchmarkServiceServer) GetKeys() []string {
	s.keysMu.RLock()
	defer s.keysMu.RUnlock()
	return append([]string{}, s.keys...)
}

func (s *BenchmarkServiceServer) SendCTRLMessage(msg *pb.CTRLMessage) error {
	s.streamMu.Lock()
	defer s.streamMu.Unlock()
	if s.currStream == nil {
		return errors.New("no active stream")
	}
	return s.currStream.Send(msg)
}

func (s *BenchmarkServiceServer) CTRLStream(stream pb.BenchmarkService_CTRLStreamServer) error {
	// Store stream for server-initiated messages
	s.streamMu.Lock()
	s.currStream = stream
	s.streamMu.Unlock()

	// Clean up stream when done
	defer func() {
		s.streamMu.Lock()
		s.currStream = nil
		s.streamMu.Unlock()
	}()

	for {
		select {
		case <-s.termChan:
			return nil
		default:

			req, err := stream.Recv()

			if err == io.EOF {
				// client closes the stream
				return nil
			}

			if err != nil {
				s.logger.Printf("Error receiving grpc message from client: %v", err)
				return err
			}

			switch payload := req.Payload.(type) {
			case *pb.CTRLMessage_KeyBatch:
				s.receiveKeys(payload.KeyBatch.GetKeys(), payload.KeyBatch.GetIsLastBatch())
				s.logger.Printf("Received batch of %d keys. Total keys: %d", len(payload.KeyBatch.Keys), s.receivedKeys)
				response := &pb.CTRLMessage{
					Payload: &pb.CTRLMessage_KeyBatchResponse{
						KeyBatchResponse: &pb.KeyBatchResponse{
							TotalKeysReceived: s.receivedKeys,
						},
					},
				}
				err = stream.Send(response)
			case *pb.CTRLMessage_ConfigFile:
				bytes := payload.ConfigFile.GetContent()
				err = json.Unmarshal(bytes, &s.ctlConfig)
				if err != nil {
					s.logger.Printf("Error unmarshalling config file: %v", err)
					return err
				}
				configPretty, _ := json.MarshalIndent(s.ctlConfig, "", "  ")
				s.logger.Printf("Received config file:\n %s", string(configPretty))
				response := &pb.CTRLMessage{
					Payload: &pb.CTRLMessage_ConfigFileResponse{
						ConfigFileResponse: &pb.ConfigFileResponse{
							Success: true,
						},
					},
				}
				err = stream.Send(response)
			case *pb.CTRLMessage_Shutdown:
				s.logger.Printf("Received shutdown message from client")
				close(s.termChan)
			}

			if err != nil {
				s.logger.Printf("Error sending grpc message to client: %v", err)
				return err
			}
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
