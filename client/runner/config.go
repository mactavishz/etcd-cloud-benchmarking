package runner

import (
	benchCfg "csb/control/config"
	generator "csb/data-generator"
	"errors"
	"math/rand"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
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

// WorkloadType represents predefined workload distributions
type WorkloadType string

const (
	ReadHeavy   WorkloadType = "read-heavy"   // 95% reads, 5% writes
	UpdateHeavy WorkloadType = "update-heavy" // 50% reads, 50% writes
	ReadOnly    WorkloadType = "read-only"    // 100% reads
)

func GetRWPercentages(w string) (int, int, error) {
	// Configure workload distribution
	var readPercent, writePercent int
	switch WorkloadType(w) {
	case ReadHeavy:
		readPercent, writePercent = 95, 5
	case UpdateHeavy:
		readPercent, writePercent = 50, 50
	case ReadOnly:
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
}
