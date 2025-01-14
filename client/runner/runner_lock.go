package runner

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	grpcserver "csb/client/grpc"
	lg "csb/client/logger"
	"csb/control/constants"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"go.uber.org/zap"
)

func NewBenchmarkRunnerLock(config *BenchmarkRunConfig, logger *lg.Logger) (*BenchmarkRunnerLock, error) {
	clients := make([]*clientv3.Client, config.InitialClients)
	sessions := make([]*concurrency.Session, config.InitialClients)

	// Create client connections and sessions
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

		// Create session for distributed locking
		session, err := concurrency.NewSession(cli)
		if err != nil {
			// Clean up clients and sessions
			for j := 0; j < i; j++ {
				sessions[j].Close()
				clients[j].Close()
			}
			return nil, fmt.Errorf("failed to create session %d: %w", i, err)
		}
		sessions[i] = session
	}

	rg := rand.New(rand.NewSource(config.Seed))
	metricsExporter, err := NewMetricsExporter(config.MetricsFile, config.MetricsBatchSize, (&LockMetric{}).ToCSVHeader())
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics exporter: %w", err)
	}

	// Generate lock names
	lockNames := make([]string, config.NumKeys)
	for i := 0; i < config.NumKeys; i++ {
		// since all key starts with / we can use /lock[key]
		lockNames[i] = fmt.Sprintf("/lock%s", config.Keys[i])
	}

	return &BenchmarkRunnerLock{
		config:          config,
		clients:         clients,
		sessions:        sessions,
		results:         make([]*StepResult, 0),
		metricsExporter: metricsExporter,
		rand:            rg,
		lockNames:       lockNames,
		logger:          logger,
	}, nil
}

func (r *BenchmarkRunnerLock) Close() error {
	var lastErr error
	for i := range r.clients {
		if err := r.sessions[i].Close(); err != nil {
			lastErr = fmt.Errorf("failed to close session %d: %w", i, err)
		}
		if err := r.clients[i].Close(); err != nil {
			lastErr = fmt.Errorf("failed to close client %d: %w", i, err)
		}
	}
	return lastErr
}

func (r *BenchmarkRunnerLock) addClients(numNewClients int) error {
	for i := 0; i < numNewClients; i++ {
		cli, err := clientv3.New(clientv3.Config{
			Endpoints:   r.config.Endpoints,
			DialTimeout: 5 * time.Second,
			Logger:      zap.NewNop(),
		})
		if err != nil {
			return fmt.Errorf("failed to create new client: %w", err)
		}

		session, err := concurrency.NewSession(cli)
		if err != nil {
			cli.Close()
			return fmt.Errorf("failed to create new session: %w", err)
		}

		r.clients = append(r.clients, cli)
		r.sessions = append(r.sessions, session)
	}
	return nil
}

// Quick acquire-release cycles without any KV operations
func (r *BenchmarkRunnerLock) runLockOnlyWorkload(mutex *concurrency.Mutex, clientID int, lockName string, runPhase string, latencyChan chan time.Duration) error {
	var (
		acquireLatency, releaseLatency time.Duration
		success                        bool = false
		err                            error
		statusCode, lockOpStatusCode   int
		statusText                     string = ""
		lockOpStatusText               string = ""
	)

	tryLockCtx, tryLockCtxCancel := GetTimeoutCtx(time.Duration(r.config.MaxWaitTime))
	defer tryLockCtxCancel()
	start := time.Now()
	if err = mutex.TryLock(tryLockCtx); err == nil {
		acquireLatency = time.Since(start)
		latencyChan <- acquireLatency
		// Release immediately
		unLockCtx, unLockCtxCancel := GetTimeoutCtx(time.Duration(r.config.MaxWaitTime))
		defer unLockCtxCancel()
		releaseStart := time.Now()
		err = mutex.Unlock(unLockCtx)
		releaseLatency = time.Since(releaseStart)
		if err != nil {
			lockOpStatusCode, lockOpStatusText = GetErrInfo(err)
			r.logger.Printf("Failed to release the lock: %v", err)
		}
		latencyChan <- releaseLatency
		success = true
	} else if err == concurrency.ErrLocked {
		r.logger.Printf("Failed to acquire the lock %s, which is held by other session: %v", mutex.Key(), err)
	} else if err == concurrency.ErrSessionExpired {
		r.logger.Printf("Failed to acquire the lock, session expired: %v", err)
	}

	if err != nil && lockOpStatusCode == 0 {
		lockOpStatusCode, lockOpStatusText = GetErrInfo(err)
	}

	go func() {
		metric := &LockMetric{
			RequestMetric: &RequestMetric{
				Timestamp:  time.Now(),
				Key:        "",
				Operation:  "lock",
				Latency:    acquireLatency + releaseLatency,
				Success:    success,
				ClientID:   clientID,
				NumClients: len(r.clients),
				RunPhase:   runPhase,
				StatusCode: statusCode,
				StatusText: statusText,
			},
			LockName:         lockName,
			AquireLatency:    acquireLatency,
			ReleaseLatency:   releaseLatency,
			LockOpStatusCode: lockOpStatusCode,
			LockOpStatusText: lockOpStatusText,
			ContentionLevel:  r.contentionLevel,
		}

		// Add metric to exporter
		if r.metricsExporter != nil {
			if err := r.metricsExporter.AddMetric(metric); err != nil {
				r.logger.Printf("Failed to export metric: %v", err)
			}
		}
	}()

	return err
}

// Mixed workload with lock acquisition, write, lock release operations
func (r *BenchmarkRunnerLock) runLockMixedWorkload(mutex *concurrency.Mutex, rg *rand.Rand, key string, clientID int, lockName string, runPhase string, latencyChan chan time.Duration) error {
	var (
		acquireLatency, kvLatency, releaseLatency time.Duration
		success                                   bool = false
		err                                       error
		statusCode, lockOpStatusCode              int
		statusText                                string = ""
		lockOpStatusText                          string = ""
		isRead                                    bool   = r.config.WorkloadType == constants.WORKLOAD_TYPE_LOCK_MIXED_READ
	)

	tryLockCtx, tryLockCtxCancel := GetTimeoutCtx(time.Duration(r.config.MaxWaitTime))
	defer tryLockCtxCancel()
	start := time.Now()
	if err = mutex.TryLock(tryLockCtx); err == nil {
		acquireLatency = time.Since(start)
		success = true
		latencyChan <- acquireLatency

		// Perform KV operation -- write
		client := r.clients[clientID%len(r.clients)]
		newVal, _ := r.generator.GenerateValue(r.config.ValueSize, rg)

		kvCtx, kvCtxCancel := GetTimeoutCtx(time.Duration(r.config.MaxWaitTime))
		defer kvCtxCancel()
		kvStart := time.Now()
		if isRead {
			_, err = client.Get(kvCtx, key)
		} else {
			_, err = client.Put(kvCtx, key, string(newVal))
		}
		kvLatency = time.Since(kvStart)
		latencyChan <- kvLatency
		if err != nil {
			statusCode, statusText = GetErrInfo(err)
			success = false
		}

		unLockCtx, unLockCtxCancel := GetTimeoutCtx(time.Duration(r.config.MaxWaitTime))
		defer unLockCtxCancel()
		releaseStart := time.Now()
		err = mutex.Unlock(unLockCtx)
		releaseLatency = time.Since(releaseStart)
		latencyChan <- releaseLatency
		if err != nil {
			success = false
			r.logger.Printf("Failed to release the lock: %v", err)
			lockOpStatusCode, lockOpStatusText = GetErrInfo(err)
		}
	} else if err == concurrency.ErrLocked {
		r.logger.Printf("Failed to acquire the lock, lock is held by other session, : %v", err)
	} else if err == concurrency.ErrSessionExpired {
		r.logger.Printf("Failed to acquire the lock, session expired: %v", err)
	}

	if err != nil && lockOpStatusCode == 0 {
		lockOpStatusCode, lockOpStatusText = GetErrInfo(err)
	}

	go func() {
		var operationStr string
		if isRead {
			operationStr = "lock-r"
		} else {
			operationStr = "lock-w"
		}
		// Record metrics for all operations
		metric := &LockMetric{
			RequestMetric: &RequestMetric{
				Timestamp:  time.Now(),
				Key:        key,
				Operation:  operationStr,
				Latency:    acquireLatency + kvLatency + releaseLatency,
				Success:    success,
				RunPhase:   runPhase,
				StatusCode: statusCode,
				StatusText: statusText,
			},
			LockName:         lockName,
			AquireLatency:    acquireLatency,
			ReleaseLatency:   releaseLatency,
			LockOpStatusCode: lockOpStatusCode,
			LockOpStatusText: lockOpStatusText,
			ContentionLevel:  r.contentionLevel,
		}

		// Add metric to exporter
		if r.metricsExporter != nil {
			if err := r.metricsExporter.AddMetric(metric); err != nil {
				r.logger.Printf("Failed to export metric: %v", err)
			}
		}
	}()

	return err
}

func (r *BenchmarkRunnerLock) runLoadStep(ctx context.Context, numClients int, isWarmup bool) (*StepResult, error) {
	runPhase := "main"
	if isWarmup {
		runPhase = "warmup"
	}

	result := &StepResult{
		NumClients: numClients,
		StartTime:  time.Now(),
		Latencies:  make([]time.Duration, 0),
	}

	if r.config.WorkloadType == constants.WORKLOAD_TYPE_LOCK_CONTENTION {
		r.contentionLevel = numClients / 2
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

			rg := r.generator.NewRand(r.config.Seed, clientID)
			session := r.sessions[clientID%len(r.sessions)]

			for {
				select {
				case <-ctx.Done():
					return
				default:
					// Select lock name based on workload type
					var lockName string
					var key string
					if r.config.WorkloadType == constants.WORKLOAD_TYPE_LOCK_CONTENTION {
						// Determine a random starting index for the section
						startIndex := rg.Intn(len(r.lockNames) - r.contentionLevel + 1) // Ensure the range fits the slice
						// Use a small subset of locks for higher contention
						lockName = r.lockNames[startIndex+rg.Intn(r.contentionLevel)]
					} else {
						// Randonly pick a lockname from all available names
						lockName = r.lockNames[rg.Intn(len(r.lockNames))]
					}

					key = lockName[5:] // Remove "/lock" prefix
					mutex := concurrency.NewMutex(session, lockName)

					var err error

					switch r.config.WorkloadType {
					case constants.WORKLOAD_TYPE_LOCK_ONLY:
						err = r.runLockOnlyWorkload(mutex, clientID, lockName, runPhase, latencyChan)
					case constants.WORKLOAD_TYPE_LOCK_MIXED_READ, constants.WORKLOAD_TYPE_LOCK_MIXED_WRITE:
						err = r.runLockMixedWorkload(mutex, rg, key, clientID, lockName, runPhase, latencyChan)
					case constants.WORKLOAD_TYPE_LOCK_CONTENTION:
						err = r.runLockOnlyWorkload(mutex, clientID, lockName, runPhase, latencyChan)
					}

					if err != nil {
						atomic.AddInt64(&result.Errors, 1)
					}
					atomic.AddInt64(&result.Operations, 1)
				}
			}
		}(i)
	}

	wg.Wait()
	close(latencyChan)
	result.EndTime = time.Now()

	r.calculateP99Latency(result)
	return result, nil
}

func (r *BenchmarkRunnerLock) calculateP99Latency(result *StepResult) {
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

func (r *BenchmarkRunnerLock) Run(s *grpcserver.BenchmarkServiceServer) error {
	// Implementation follows same pattern as BenchmarkRunnerKV
	// Warm-up period
	reportStr := fmt.Sprintf("Starting warm-up step (%v)...", r.config.WarmupDuration)
	r.logger.Println(reportStr)
	s.SendBenchmarkStatus(reportStr)
	warmupCtx, warmupCancel := context.WithTimeout(context.Background(), time.Duration(r.config.WarmupDuration))
	defer warmupCancel()

	warmupResult, err := r.runLoadStep(warmupCtx, r.config.InitialClients, true)
	if err != nil {
		s.SendBenchmarkStatus("Warm-up failed")
		return fmt.Errorf("warm-up failed: %w", err)
	}

	reportStr = fmt.Sprintf("Warm-up step completed with %d clients (P99: %dms), #Ops: %d, #Errors: %d", r.config.InitialClients, warmupResult.P99Latency.Milliseconds(), warmupResult.Operations, warmupResult.Errors)
	r.logger.Println(reportStr)
	s.SendBenchmarkStatus(reportStr)

	// Main benchmark loop
	curNumClients := r.config.InitialClients
	remainingTime := time.Duration(r.config.TotalDuration)
	maxClientsReached := false

	for remainingTime > 0 {
		reportStr = fmt.Sprintf("Starting step with %d clients", curNumClients)
		r.logger.Println(reportStr)
		s.SendBenchmarkStatus(reportStr)
		var actualDuration time.Duration
		if remainingTime < time.Duration(r.config.StepDuration) {
			actualDuration = remainingTime
		} else {
			actualDuration = time.Duration(r.config.StepDuration)
		}

		stepCtx, stepCancel := context.WithTimeout(context.Background(), actualDuration)
		result, err := r.runLoadStep(stepCtx, curNumClients, false)
		stepCancel()

		if err != nil {
			s.SendBenchmarkStatus("Step failed")
			return fmt.Errorf("step failed with %d clients: %w", curNumClients, err)
		}

		r.mut.Lock()
		r.results = append(r.results, result)
		r.mut.Unlock()

		reportStr = fmt.Sprintf("Step completed with %d clients (P99: %dms), #Ops: %d, #Errors: %d", curNumClients, result.P99Latency.Milliseconds(), result.Operations, result.Errors)
		r.logger.Println(reportStr)
		s.SendBenchmarkStatus(reportStr)

		if curNumClients >= r.config.MaxClients {
			if !maxClientsReached {
				maxClientsReached = true
				r.logger.Printf("Reached maximum number of clients")
			}
		} else {
			acutalIncClients := r.config.ClientStepSize
			if curNumClients+r.config.ClientStepSize > r.config.MaxClients {
				acutalIncClients = r.config.MaxClients - curNumClients
			}
			curNumClients += acutalIncClients
			if err = r.addClients(acutalIncClients); err != nil {
				return err
			}
		}

		remainingTime -= time.Duration(r.config.StepDuration)
	}

	if r.metricsExporter != nil {
		if err := r.metricsExporter.Close(); err != nil {
			s.SendBenchmarkStatus("Failed to close metrics exporter")
			r.logger.Printf("Failed to close metrics exporter: %v", err)
		}
	}

	s.SendBenchmarkStatus("All benchmark steps are completed")
	r.logger.Printf("All benchmark steps are completed")
	return nil
}

func (r *BenchmarkRunnerLock) GetResults() []*StepResult {
	return r.results
}
