package runner

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"csb/control/constants"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"go.uber.org/zap"
)

// EtcdOp represents different types of operations
type EtcdOp string

const (
	OpLockAcquire EtcdOp = "lock-acquire"
	OpLockRelease EtcdOp = "lock-release"
	OpLeaseRenew  EtcdOp = "lease-renew"
	OpKVWrite     EtcdOp = "write"
	OpKVRead      EtcdOp = "read"
)

func NewBenchmarkRunnerLock(config *BenchmarkRunConfig) (*BenchmarkRunnerLock, error) {
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
	metricsExporter, err := NewMetricsExporter(config.MetricsFile, config.MetricsBatchSize, LockMetric{}.ToCSVHeader())
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics exporter: %w", err)
	}

	// Generate lock names
	lockNames := make([]string, config.NumKeys)
	for i := 0; i < config.NumKeys; i++ {
		// since all key starts with / we can use /lock[key]
		lockNames[i] = fmt.Sprintf("/lock%s", config.Keys[i])
	}

	var contentionLevel int
	if config.WorkloadType == constants.WORKLOAD_TYPE_LOCK_CONTENTION {
		contentionLevel = config.InitialClients / 2
	}

	return &BenchmarkRunnerLock{
		config:          config,
		clients:         clients,
		sessions:        sessions,
		results:         make([]*StepResult, 0),
		metricsExporter: metricsExporter,
		rand:            rg,
		lockNames:       lockNames,
		contentionLevel: contentionLevel,
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
func (r *BenchmarkRunnerLock) runLockOnlyWorkload(mutex *concurrency.Mutex, clientID int, runPhase string, latencyChan chan time.Duration) error {
	var (
		acquireLatency, releaseLatency time.Duration
		success                        bool = false
		err                            error
		statusCode, lockOpStatusCode   int
		statusText                     string = "N/A"
		lockOpStatusText               string = "N/A"
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
			log.Printf("Failed to release the lock: %v", err)
		}
		latencyChan <- releaseLatency
		success = true
	} else if err == concurrency.ErrLocked {
		log.Printf("Failed to acquire the lock, lock is held by other session, : %v", err)
	} else if err == concurrency.ErrSessionExpired {
		log.Printf("Failed to acquire the lock, session expired: %v", err)
	}

	if err != nil && lockOpStatusCode == 0 {
		lockOpStatusCode, lockOpStatusText = GetErrInfo(err)
	}

	go func() {
		metric := LockMetric{
			RequestMetric: RequestMetric{
				Timestamp:  time.Now(),
				Key:        "N/A",
				Operation:  strings.Join([]string{string(OpLockAcquire), string(OpLockRelease)}, "+"),
				Latency:    acquireLatency + releaseLatency,
				Success:    success,
				ClientID:   clientID,
				NumClients: len(r.clients),
				RunPhase:   runPhase,
				StatusCode: statusCode,
				StatusText: statusText,
			},
			LockName:         mutex.Key(),
			AquireLatency:    acquireLatency,
			ReleaseLatency:   releaseLatency,
			LockOpStatusCode: lockOpStatusCode,
			LockOpStatusText: lockOpStatusText,
			ContentionLevel:  r.contentionLevel,
		}

		// Add metric to exporter
		if r.metricsExporter != nil {
			if err := r.metricsExporter.AddMetric(metric); err != nil {
				log.Printf("Failed to export metric: %v", err)
			}
		}
	}()

	return err
}

// Mixed workload with lock acquisition, write, lock release operations
func (r *BenchmarkRunnerLock) runLockMixedWorkload(mutex *concurrency.Mutex, rg *rand.Rand, key string, clientID int, runPhase string, latencyChan chan time.Duration) error {
	var (
		acquireLatency, kvLatency, releaseLatency time.Duration
		success                                   bool = false
		err                                       error
		statusCode, lockOpStatusCode              int
		statusText                                string = "N/A"
		lockOpStatusText                          string = "N/A"
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
		_, err = client.Put(kvCtx, key, string(newVal))
		kvLatency = time.Since(kvStart)
		if err != nil {
			statusCode, statusText = GetErrInfo(err)
			success = false
		}

		unLockCtx, unLockCtxCancel := GetTimeoutCtx(time.Duration(r.config.MaxWaitTime))
		defer unLockCtxCancel()
		releaseStart := time.Now()
		err = mutex.Unlock(unLockCtx)
		releaseLatency = time.Since(releaseStart)
		if err != nil {
			success = false
			log.Printf("Failed to release the lock: %v", err)
			lockOpStatusCode, lockOpStatusText = GetErrInfo(err)
		}
		latencyChan <- kvLatency
	} else if err == concurrency.ErrLocked {
		log.Printf("Failed to acquire the lock, lock is held by other session, : %v", err)
	} else if err == concurrency.ErrSessionExpired {
		log.Printf("Failed to acquire the lock, session expired: %v", err)
	}

	if err != nil && lockOpStatusCode == 0 {
		lockOpStatusCode, lockOpStatusText = GetErrInfo(err)
	}

	go func() {
		// Record metrics for all operations
		metric := LockMetric{
			RequestMetric: RequestMetric{
				Timestamp:  time.Now(),
				Operation:  strings.Join([]string{string(OpLockAcquire), string(OpKVWrite), string(OpLockRelease)}, "+"),
				Latency:    acquireLatency + kvLatency + releaseLatency,
				Success:    success,
				RunPhase:   runPhase,
				StatusCode: statusCode,
				StatusText: statusText,
			},
			LockName:         mutex.Key(),
			AquireLatency:    acquireLatency,
			ReleaseLatency:   releaseLatency,
			LockOpStatusCode: lockOpStatusCode,
			LockOpStatusText: lockOpStatusText,
			ContentionLevel:  r.contentionLevel,
		}

		// Add metric to exporter
		if r.metricsExporter != nil {
			if err := r.metricsExporter.AddMetric(metric); err != nil {
				log.Printf("Failed to export metric: %v", err)
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
						// Use a small subset of locks for higher contention
						lockName = r.lockNames[rg.Intn(r.contentionLevel)]
					} else {
						// Use all available locks
						lockName = r.lockNames[rg.Intn(len(r.lockNames))]
					}

					key = lockName[5:] // Remove "/lock" prefix
					mutex := concurrency.NewMutex(session, lockName)

					var err error

					switch r.config.WorkloadType {
					case constants.WORKLOAD_TYPE_LOCK_ONLY:
						err = r.runLockOnlyWorkload(mutex, clientID, runPhase, latencyChan)
					case constants.WORKLOAD_TYPE_LOCK_MIXED:
						err = r.runLockMixedWorkload(mutex, rg, key, clientID, runPhase, latencyChan)
					case constants.WORKLOAD_TYPE_LOCK_CONTENTION:
						err = r.runLockOnlyWorkload(mutex, clientID, runPhase, latencyChan)
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

func (r *BenchmarkRunnerLock) Run() error {
	// Implementation follows same pattern as BenchmarkRunnerKV
	// Warm-up period
	log.Printf("Starting warm-up step (%v)...", r.config.WarmupDuration)
	warmupCtx, warmupCancel := context.WithTimeout(context.Background(), time.Duration(r.config.WarmupDuration))
	defer warmupCancel()

	warmupResult, err := r.runLoadStep(warmupCtx, r.config.InitialClients, true)
	if err != nil {
		return fmt.Errorf("warm-up failed: %w", err)
	}

	log.Printf("Warm-up step completed with %d clients (P99: %dms), #Ops: %d, #Errors: %d", r.config.InitialClients, warmupResult.P99Latency.Milliseconds(), warmupResult.Operations, warmupResult.Errors)

	// Main benchmark loop
	curNumClients := r.config.InitialClients
	remainingTime := time.Duration(r.config.TotalDuration)
	saturated := false

	for remainingTime > 0 {
		log.Printf("Starting step with %d clients", curNumClients)
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
			return fmt.Errorf("step failed with %d clients: %w", curNumClients, err)
		}

		r.mut.Lock()
		r.results = append(r.results, result)
		r.mut.Unlock()

		if !saturated && result.P99Latency > time.Duration(r.config.SLALatency) {
			log.Printf("Throughput is saturated, SLA violated with %d clients (P99: %dms)",
				curNumClients, result.P99Latency.Milliseconds())
			saturated = true
		}

		log.Printf("Step completed with %d clients (P99: %dms), #Ops: %d, #Errors: %d", curNumClients, result.P99Latency.Milliseconds(), result.Operations, result.Errors)

		if !saturated {
			curNumClients += r.config.ClientStepSize
			if err = r.addClients(r.config.ClientStepSize); err != nil {
				return err
			}
		}

		remainingTime -= time.Duration(r.config.StepDuration)
	}

	if r.metricsExporter != nil {
		if err := r.metricsExporter.Close(); err != nil {
			log.Printf("Failed to close metrics exporter: %v", err)
		}
	}

	log.Printf("All benchmark steps are completed")
	return nil
}

func (r *BenchmarkRunnerLock) GetResults() []*StepResult {
	return r.results
}
