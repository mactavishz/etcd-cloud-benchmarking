package runner

import (
	"errors"
	"math/rand"
	"sync"
	"time"

	logger "git.tu-berlin.de/mactavishz/csb-project-ws2425/client/logger"
	benchCfg "git.tu-berlin.de/mactavishz/csb-project-ws2425/control/config"
	"git.tu-berlin.de/mactavishz/csb-project-ws2425/control/constants"
	generator "git.tu-berlin.de/mactavishz/csb-project-ws2425/data-generator"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

// BenchmarkRunConfig holds all configuration parameters
type BenchmarkRunConfig struct {
	benchCfg.BenchctlConfig

	// Workload parameters
	ReadPercent  int
	WritePercent int

	// Keys to operate on
	Keys []string

	// Metrics parameters
	MetricsBatchSize int
}

func GetRWPercentages(w string) (int, int, error) {
	//workload read, write distribution
	var readPercent, writePercent int
	switch w {
	case constants.WORKLOAD_TYPE_READ_HEAVY:
		readPercent, writePercent = 95, 5
	case constants.WORKLOAD_TYPE_UPDATE_HEAVY:
		readPercent, writePercent = 50, 50
	case constants.WORKLOAD_TYPE_READ_ONLY:
		readPercent, writePercent = 100, 0
	default:
		return 0, 0, errors.New("unknown workload type")
	}
	return readPercent, writePercent, nil
}

type StepResult struct {
	NumClients int
	StartTime  time.Time
	EndTime    time.Time
	Latencies  []time.Duration
	Operations int64
	Errors     int64
	P99Latency time.Duration
}

// BenchmarkRunner manages the benchmark execution
type BenchmarkRunnerKV struct {
	config          *BenchmarkRunConfig
	clients         []*clientv3.Client
	results         []*StepResult
	metricsExporter *MetricsExporter
	mut             sync.Mutex
	rand            *rand.Rand
	generator       *generator.Generator
	logger          *logger.Logger
}

// BenchmarkRunnerLock manages the lock service benchmark
type BenchmarkRunnerLock struct {
	config          *BenchmarkRunConfig
	clients         []*clientv3.Client
	sessions        []*concurrency.Session
	results         []*StepResult
	metricsExporter *MetricsExporter
	mut             sync.Mutex
	rand            *rand.Rand
	generator       *generator.Generator

	// Lock-specific configurations
	lockNames       []string // List of available lock names
	contentionLevel int      // Number of clients competing for same lock
	logger          *logger.Logger
}
