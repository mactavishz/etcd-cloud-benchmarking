package config

import (
	"math/rand"
	"os"
	"testing"
	"time"

	"csb/control/constants"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *BenchctlConfig
		isErr  bool
	}{
		{
			name:   "valid default config",
			config: GetDefaultConfig(),
			isErr:  false,
		},
		{
			name: "valid config with multiple endpoints and protocols",
			config: &BenchctlConfig{
				Seed: constants.DEFAULT_SEED,
				Endpoints: []string{
					"192.168.1.1:8080",
					"http://10.0.0.1:9090",
					"https://172.16.0.1:443",
				},
				KeySize:        16,
				ValueSize:      128,
				NumKeys:        1000,
				WarmupDuration: Duration(10 * time.Minute),
				StepDuration:   Duration(2 * time.Minute),
				TotalDuration:  Duration(1 * time.Hour),
				InitialClients: 10,
				ClientStepSize: 10,
				MaxClients:     200,
				MaxWaitTime:    Duration(1 * time.Second),
				WorkloadType:   constants.WORKLOAD_TYPE_READ_HEAVY,
				Scenario:       constants.SCENARIO_KV_STORE,
				MetricsFile:    "test_metrics.csv",
			},
			isErr: false,
		},
		{
			name: "invalid seed (zero)",
			config: func() *BenchctlConfig {
				cfg := GetDefaultConfig()
				cfg.Seed = 0
				cfg.Endpoints = []string{"192.168.1.1:8080"}
				return cfg
			}(),
			isErr: true,
		},
		{
			name: "invalid endpoints",
			config: func() *BenchctlConfig {
				cfg := GetDefaultConfig()
				cfg.Endpoints = []string{
					"invalid:8080",           // invalid IP
					"192.168.1.1",            // missing port
					"192.168.1.1:99999",      // invalid port
					"ftp://192.168.1.1:8080", // invalid protocol
					"192.168.1.1:abc",        // non-numeric port
				}
				return cfg
			}(),
			isErr: true,
		},
		{
			name: "invalid client numbers",
			config: func() *BenchctlConfig {
				cfg := GetDefaultConfig()
				cfg.Endpoints = []string{"192.168.1.1:8080"}
				cfg.InitialClients = 0  // should be > 0
				cfg.ClientStepSize = -1 // should be > 0
				cfg.MaxClients = 0      // should be > 0
				return cfg
			}(),
			isErr: true,
		},
		{
			name: "invalid workload type",
			config: func() *BenchctlConfig {
				cfg := GetDefaultConfig()
				cfg.Endpoints = []string{"192.168.1.1:8080"}
				cfg.WorkloadType = "write-heavy" // not in allowed values
				return cfg
			}(),
			isErr: true,
		},
		{
			name: "mismatched workload type and scenario kv-store",
			config: func() *BenchctlConfig {
				cfg := GetDefaultConfig()
				cfg.Endpoints = []string{"192.168.1.1:8080"}
				cfg.Scenario = constants.SCENARIO_KV_STORE
				cfg.WorkloadType = "lock-only" // invalid: doesn't match scenario
				return cfg
			}(),
			isErr: true,
		},
		{
			name: "mismatched workload type and scenario lock-service",
			config: func() *BenchctlConfig {
				cfg := GetDefaultConfig()
				cfg.Endpoints = []string{"192.168.1.1:8080"}
				cfg.Scenario = constants.SCENARIO_LOCK_SERVICE
				cfg.WorkloadType = "read-only" // invalid: doesn't match scenario
				return cfg
			}(),
			isErr: true,
		},
		{
			name: "invalid key size",
			config: func() *BenchctlConfig {
				cfg := GetDefaultConfig()
				cfg.KeySize = rand.Intn(constants.MIN_KEY_SIZE) - 1
				return cfg
			}(),
			isErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)
			if (err != nil) != tt.isErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.isErr)
			}
		})
	}
}

func TestReadConfig(t *testing.T) {
	// Test reading valid config
	validConfig := GetDefaultConfig()
	validConfig.Endpoints = []string{"192.168.1.1:8080"}

	tempFile := t.TempDir() + "/valid_config.json"
	err := validConfig.WriteConfig(tempFile)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err = ReadConfig(tempFile)
	if err != nil {
		t.Errorf("ReadConfig() error = %v", err)
	}

	// Test reading non-existent file
	_, err = ReadConfig("non_existent_file.json")
	if err == nil {
		t.Error("ReadConfig() expected error for non-existent file")
	}

	// Test reading invalid JSON
	invalidJSONFile := t.TempDir() + "/invalid.json"
	err = os.WriteFile(invalidJSONFile, []byte("{invalid json}"), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid JSON file: %v", err)
	}

	_, err = ReadConfig(invalidJSONFile)
	if err == nil {
		t.Error("ReadConfig() expected error for invalid JSON")
	}
}
