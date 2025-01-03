package config

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"csb/control/constants"

	validator "github.com/go-playground/validator/v10"
)

type BenchctlConfig struct {
	Seed           int64    `json:"seed" validate:"required,gt=0"`
	NumKeys        int      `json:"num_keys" validate:"required,gt=0"`
	KeySize        int      `json:"key_size" validate:"required,gt=0,valid_key_size"`
	ValueSize      int      `json:"value_size" validate:"required,gt=0"`
	Endpoints      []string `json:"endpoints" validate:"required,dive,valid_endpoint"`
	WarmupDuration Duration `json:"warmup_duration" validate:"required"`
	StepDuration   Duration `json:"step_duration" validate:"required"`
	TotalDuration  Duration `json:"total_duration" validate:"required"`
	InitialClients int      `json:"initial_clients" validate:"required,gt=0"`
	ClientStepSize int      `json:"client_step_size" validate:"required,gt=0"`
	MaxClients     int      `json:"max_clients" validate:"required,gt=0"`
	MaxWaitTime    Duration `json:"max_wait_time" validate:"required"`
	WorkloadType   string   `json:"workload_type" validate:"required,valid_workload_type"`
	Scenario       string   `json:"scenario" validate:"required,valid_scenario"`
	// SLA parameters
	SLALatency    Duration `json:"sla_latency" validate:"required"`
	SLAPercentile float64  `json:"sla_percentile" validate:"required,gt=0"`
	// Metrics parameters
	MetricsFile string `json:"metrics_file" validate:"required,filepath"`
}

// Custom validation tags
const (
	workloadTypeTag = "valid_workload_type"
	endpointTag     = "valid_endpoint"
	keySizeTag      = "valid_key_size"
	scenarioTag     = "valid_scenario"
)

// RegisterCustomValidators registers all custom validators for BenchctlConfig
func RegisterCustomValidators(v *validator.Validate) error {
	// Register workload type validator
	if err := v.RegisterValidation(workloadTypeTag, validateWorkloadType); err != nil {
		return fmt.Errorf("failed to register workload type validator: %w", err)
	}

	// Register duration format validator
	if err := v.RegisterValidation(scenarioTag, validateScenarioType); err != nil {
		return fmt.Errorf("failed to register scenario validator: %w", err)
	}

	// Register endpoint validator
	if err := v.RegisterValidation(endpointTag, validateEndpoint); err != nil {
		return fmt.Errorf("failed to register endpoint validator: %w", err)
	}

	if err := v.RegisterValidation(keySizeTag, validateKeySize); err != nil {
		return fmt.Errorf("failed to register key size validator: %w", err)
	}

	return nil
}

func validateKeySize(fl validator.FieldLevel) bool {
	keySize := fl.Field().Int()
	return keySize >= int64(constants.MIN_KEY_SIZE)
}

// validateWorkloadType ensures the workload type is one of the allowed values
func validateWorkloadType(fl validator.FieldLevel) bool {
	workloadType := fl.Field().String()
	validTypes := map[string]bool{
		"read-heavy":   true,
		"update-heavy": true,
		"read-only":    true,
	}

	return validTypes[workloadType]
}

func validateScenarioType(fl validator.FieldLevel) bool {
	scenarioType := fl.Field().String()
	validTypes := map[string]bool{
		"kv-store":     true,
		"lock-service": true,
	}
	return validTypes[scenarioType]
}

// validateEndpoint ensures the endpoint string is in the correct format
func validateEndpoint(fl validator.FieldLevel) bool {
	endpoint := fl.Field().String()

	// Strip protocol if present
	if strings.HasPrefix(endpoint, "http://") {
		endpoint = endpoint[7:]
	} else if strings.HasPrefix(endpoint, "https://") {
		endpoint = endpoint[8:]
	}

	// Split host and port
	host, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		return false
	}

	// Validate port
	portNum, err := strconv.Atoi(port)
	if err != nil || portNum < 1 || portNum > 65535 {
		return false
	}

	// Validate IP address
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	return true
}

func GetDefaultConfig() *BenchctlConfig {
	return &BenchctlConfig{
		Seed:           constants.DEFAULT_SEED,
		NumKeys:        constants.DEFAULT_NUM_KEYS,
		Endpoints:      []string{},
		KeySize:        16,
		ValueSize:      128,
		WarmupDuration: Duration(5 * time.Minute),
		StepDuration:   Duration(1 * time.Minute),
		TotalDuration:  Duration(30 * time.Minute),
		InitialClients: 5,
		ClientStepSize: 5,
		MaxClients:     100,
		MaxWaitTime:    Duration(500 * time.Millisecond),
		WorkloadType:   constants.WORKLOAD_TYPE_READ_HEAVY,
		Scenario:       constants.SCENARIO_KV_STORE,
		SLALatency:     Duration(100 * time.Millisecond),
		SLAPercentile:  0.99,
		MetricsFile:    "metrics.csv",
	}
}

func ValidateConfig(config *BenchctlConfig) error {
	v := validator.New()
	if err := RegisterCustomValidators(v); err != nil {
		return fmt.Errorf("failed to register custom validators: %w", err)
	}

	return v.Struct(config)
}

func ReadConfig(path string) (*BenchctlConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	benchctlConfig := &BenchctlConfig{}
	err = json.Unmarshal(data, benchctlConfig)
	if err != nil {
		return nil, err
	}
	err = ValidateConfig(benchctlConfig)
	if err != nil {
		return nil, err
	}
	return benchctlConfig, nil
}

func (cfg *BenchctlConfig) WriteConfig(path string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return err
	}
	return nil
}
