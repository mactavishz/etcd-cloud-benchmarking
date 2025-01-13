package runner

import (
	"context"
	generator "csb/data-generator"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func NewBenchmarkRunnerKV(config *BenchmarkRunConfig, logger *Logger) (*BenchmarkRunnerKV, error) {
	clients := make([]*clientv3.Client, config.InitialClients)

	// Create multiple client connections
	for i := 0; i < config.InitialClients; i++ {
		cli, err := clientv3.New(clientv3.Config{
			Endpoints:   config.Endpoints,
			DialTimeout: 5 * time.Second,
			Logger:      zap.NewNop(),
		})
		if err != nil {
			// Clean up any clients already created
			for j := 0; j < i; j++ {
				clients[j].Close()
			}
			return nil, fmt.Errorf("failed to create etcd client %d: %w", i, err)
		}
		clients[i] = cli
	}
	rg := rand.New(rand.NewSource(config.Seed))
	metricsExporter, err := NewMetricsExporter(config.MetricsFile, config.MetricsBatchSize, (&RequestMetric{}).ToCSVHeader())
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics exporter: %w", err)
	}
	return &BenchmarkRunnerKV{
		config:          config,
		clients:         clients,
		results:         make([]*StepResult, 0),
		rand:            rg,
		generator:       generator.NewGenerator(rg),
		metricsExporter: metricsExporter,
		logger:          logger,
	}, nil
}

func (r *BenchmarkRunnerKV) Close() error {
	var lastErr error
	for i, cli := range r.clients {
		if err := cli.Close(); err != nil {
			lastErr = fmt.Errorf("failed to close client %d: %w", i, err)
		}
	}
	return lastErr
}

func (r *BenchmarkRunnerKV) GetResults() []*StepResult {
	return r.results
}

func (r *BenchmarkRunnerKV) addClients(numNewClients int) error {
	for i := 0; i < numNewClients; i++ {
		cli, err := clientv3.New(clientv3.Config{
			Endpoints:   r.config.Endpoints,
			DialTimeout: 5 * time.Second,
			Logger:      zap.NewNop(),
		})
		if err != nil {
			return fmt.Errorf("failed to create new client: %w", err)
		}
		r.clients = append(r.clients, cli)
	}
	return nil
}

func (r *BenchmarkRunnerKV) runLoadStep(ctx context.Context, numClients int, isWarmup bool) (*StepResult, error) {
	runPhase := "main"
	if isWarmup {
		runPhase = "warmup"
	}
	result := &StepResult{
		NumClients: numClients,
		StartTime:  time.Now(),
		Latencies:  make([]time.Duration, 0),
	}

	var wg sync.WaitGroup
	latencyChan := make(chan time.Duration, numClients*int(time.Duration(r.config.StepDuration).Seconds()))

	// Start a separate goroutine to collect latencies
	go func() {
		for latency := range latencyChan {
			result.Latencies = append(result.Latencies, latency)
		}
	}()

	// Start client goroutines
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			// per goroutine random generator
			rg := r.generator.NewRand(r.config.Seed, clientID)
			// Get the assigned client from the pool
			client := r.clients[clientID%len(r.clients)]

			for {
				select {
				case <-ctx.Done():
					return
				default:
					// Determine operation type based on workload distribution
					isRead := rg.Float64()*100 < float64(r.config.ReadPercent)
					// Select random key from available keys
					key := r.config.Keys[rg.Intn(len(r.config.Keys))]
					newVal, _ := r.generator.GenerateValue(r.config.ValueSize, rg)
					requestTimeout := time.Duration(r.config.MaxWaitTime)
					timeoutCtx, cancel := context.WithTimeout(ctx, requestTimeout)
					defer cancel()

					var err error
					var statusCode int
					var statusText string = ""
					operation := "read"

					start := time.Now()
					if isRead {
						_, err = client.Get(timeoutCtx, key)
					} else {
						operation = "write"
						_, err = client.Put(timeoutCtx, key, string(newVal))
					}
					latency := time.Since(start)
					latencyChan <- latency

					if err != nil {
						statusCode, statusText = GetErrInfo(err)
					}

					go func() {
						if err != nil {
							atomic.AddInt64(&result.Errors, 1)
						}
						atomic.AddInt64(&result.Operations, 1)
					}()

					go func() {
						// Record raw metric
						metric := &RequestMetric{
							Timestamp:  time.Now(),
							Key:        key,
							Operation:  operation,
							Latency:    latency,
							Success:    err == nil,
							StatusCode: statusCode,
							StatusText: statusText,
							NumClients: numClients,
							ClientID:   clientID,
							RunPhase:   runPhase,
						}

						// Add metric to exporter
						if r.metricsExporter != nil {
							if err := r.metricsExporter.AddMetric(metric); err != nil {
								r.logger.Printf("Failed to export metric: %v", err)
							}
						}
					}()
				}
			}
		}(i)
	}

	wg.Wait()
	close(latencyChan)
	result.EndTime = time.Now()

	// Calculate P99 latency
	r.calculateP99Latency(result)

	return result, nil
}

func (r *BenchmarkRunnerKV) calculateP99Latency(result *StepResult) {
	if len(result.Latencies) == 0 {
		return
	}

	// Sort latencies
	sorted := make([]time.Duration, len(result.Latencies))
	copy(sorted, result.Latencies)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	// Calculate P99 index
	index := int(math.Ceil(float64(len(sorted))*0.99)) - 1
	result.P99Latency = sorted[index]
}

func (r *BenchmarkRunnerKV) Run() error {
	// Warm-up period
	r.logger.Printf("Starting warm-up step (%v)...", r.config.WarmupDuration)
	warmupCtx, warmupCancel := context.WithTimeout(context.Background(), time.Duration(r.config.WarmupDuration))
	defer warmupCancel()
	warmupResult, err := r.runLoadStep(warmupCtx, r.config.InitialClients, true)
	if err != nil {
		return fmt.Errorf("warm-up failed: %w", err)
	}
	r.logger.Printf("Warm-up step completed with %d clients (P99: %dms), #Ops: %d, #Errors: %d", r.config.InitialClients, warmupResult.P99Latency.Milliseconds(), warmupResult.Operations, warmupResult.Errors)

	// Main benchmark loop
	curNumClients := r.config.InitialClients
	remainingTime := time.Duration(r.config.TotalDuration)
	saturated := false

	for remainingTime > 0 {
		r.logger.Printf("Starting step with %d clients", curNumClients)
		var acutalDuration time.Duration
		if remainingTime < time.Duration(r.config.StepDuration) {
			acutalDuration = remainingTime
		} else {
			acutalDuration = time.Duration(r.config.StepDuration)
		}
		stepCtx, stepCancel := context.WithTimeout(context.Background(), acutalDuration)
		result, err := r.runLoadStep(stepCtx, curNumClients, false)
		defer stepCancel()

		if err != nil {
			return fmt.Errorf("step failed with %d clients: %w", curNumClients, err)
		}

		r.mut.Lock()
		r.results = append(r.results, result)
		r.mut.Unlock()

		// Check if SLA is violated
		if !saturated && result.P99Latency > time.Duration(r.config.SLALatency) {
			r.logger.Printf("Throughput is saturated, SLA violated with %d clients (P99: %dms)", curNumClients, result.P99Latency.Milliseconds())
			saturated = true
		}

		r.logger.Printf("Step completed with %d clients (P99: %dms), #Ops: %d, #Errors: %d", curNumClients, result.P99Latency.Milliseconds(), result.Operations, result.Errors)

		if !saturated {
			curNumClients += r.config.ClientStepSize
			err = r.addClients(r.config.ClientStepSize)
			if err != nil {
				return err
			}
		}
		remainingTime -= time.Duration(r.config.StepDuration)
	}

	if r.metricsExporter != nil {
		if err := r.metricsExporter.Close(); err != nil {
			r.logger.Printf("Failed to close metrics exporter: %v", err)
		}
	}

	r.logger.Printf("All benchmark steps are completed")
	return nil
}
