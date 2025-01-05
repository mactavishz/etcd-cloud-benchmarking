package runner

import (
	"encoding/csv"
	"os"
	"strconv"
	"sync"
	"time"
)

// StepResult holds metrics for each load step
type RequestMetric struct {
	Timestamp  time.Time     // Timestamp of the request
	Key        string        // Key being accessed
	Operation  string        // "read" or "write"
	Latency    time.Duration // Latency in milliseconds
	Success    bool          // Whether the operation succeeded
	StatusCode int           // etcd response status code
	StatusText string        // etcd response status text
	NumClients int           // Number of clients at current step
	ClientID   int           // ID of the client that made the request
	RunPhase   string        // Phase of the run
}

// LockMetric extends RequestMetric for lock-specific operations
type LockMetric struct {
	RequestMetric
	LockName        string        // Name of the lock being operated on
	AquireLatency   time.Duration // Latency of the acquire operation
	ReleaseLatency  time.Duration // Latency of the release operation
	ContentionLevel int           // Number of clients contending for the lock
}

type Metric interface {
	ToCSVRow() []string // Converts the metric to a slice of strings for CSV writing
	ToCSVHeader() []string
}

// MetricsExporter handles the export of raw metrics to CSV
type MetricsExporter struct {
	file      *os.File
	batchSize int
	metrics   []Metric
	mu        sync.Mutex
}

func (m RequestMetric) ToCSVRow() []string {
	return []string{
		strconv.FormatInt(m.Timestamp.UnixNano(), 10),
		m.Key,
		m.Operation,
		strconv.FormatInt(m.Latency.Milliseconds(), 10),
		strconv.FormatBool(m.Success),
		strconv.Itoa(m.StatusCode),
		m.StatusText,
		strconv.Itoa(m.NumClients),
		strconv.Itoa(m.ClientID),
		m.RunPhase,
	}
}

func (m RequestMetric) ToCSVHeader() []string {
	return []string{
		"unix_timestamp_nano",
		"key",
		"operation",
		"latency_ms",
		"success",
		"status_code",
		"status_text",
		"num_clients",
		"client_id",
		"run_phase",
	}
}

func (m LockMetric) ToCSVHeader() []string {
	return append(m.RequestMetric.ToCSVHeader(), "lock_name", "aquire_latency_ms", "release_latency_ms", "contention_level")
}

func (m LockMetric) ToCSVRow() []string {
	return append(m.RequestMetric.ToCSVRow(), m.LockName, strconv.FormatInt(m.AquireLatency.Milliseconds(), 10), strconv.FormatInt(m.ReleaseLatency.Milliseconds(), 10), strconv.Itoa(m.ContentionLevel))
}

func NewMetricsExporter(filename string, batchSize int, header []string) (*MetricsExporter, error) {
	file, err := os.Create(filename)
	if err != nil {
		return nil, err
	}

	// Write CSV header
	writer := csv.NewWriter(file)
	err = writer.Write(header)
	if err != nil {
		file.Close()
		return nil, err
	}
	writer.Flush()

	return &MetricsExporter{
		file:      file,
		batchSize: batchSize,
		metrics:   make([]Metric, 0, batchSize),
	}, nil
}

func (e *MetricsExporter) AddMetric(metric Metric) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.metrics = append(e.metrics, metric)

	if len(e.metrics) >= e.batchSize {
		return e.flush()
	}
	return nil
}

func (e *MetricsExporter) flush() error {
	writer := csv.NewWriter(e.file)
	for _, metric := range e.metrics {
		err := writer.Write(metric.ToCSVRow())
		if err != nil {
			return err
		}
	}
	writer.Flush()
	e.metrics = e.metrics[:0]
	return writer.Error()
}

func (e *MetricsExporter) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(e.metrics) > 0 {
		if err := e.flush(); err != nil {
			return err
		}
	}
	return e.file.Close()
}
