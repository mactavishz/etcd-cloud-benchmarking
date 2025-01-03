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

// MetricsExporter handles the export of raw metrics to CSV
type MetricsExporter struct {
	file      *os.File
	batchSize int
	metrics   []RequestMetric
	mu        sync.Mutex
}

func NewMetricsExporter(filename string, batchSize int) (*MetricsExporter, error) {
	file, err := os.Create(filename)
	if err != nil {
		return nil, err
	}

	// Write CSV header
	writer := csv.NewWriter(file)
	err = writer.Write([]string{
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
	})
	if err != nil {
		file.Close()
		return nil, err
	}
	writer.Flush()

	return &MetricsExporter{
		file:      file,
		batchSize: batchSize,
		metrics:   make([]RequestMetric, 0, batchSize),
	}, nil
}

func (e *MetricsExporter) AddMetric(metric RequestMetric) error {
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
		err := writer.Write([]string{
			strconv.FormatInt(metric.Timestamp.UnixNano(), 10),
			metric.Key,
			metric.Operation,
			strconv.FormatInt(metric.Latency.Milliseconds(), 10),
			strconv.FormatBool(metric.Success),
			strconv.Itoa(metric.StatusCode),
			metric.StatusText,
			strconv.Itoa(metric.NumClients),
			strconv.Itoa(metric.ClientID),
			metric.RunPhase,
		})
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
