package config

import (
	"encoding/json"
	"os"

	validator "github.com/go-playground/validator/v10"
)

const DEFAULT_SEED int64 = 0x207B096061CDA310

type BenchctlConfig struct {
	Seed      int64    `json:"seed" validate:"required"`
	Endpoints []string `json:"endpoints" validate:"required"`
}

func GetDefaultConfig() *BenchctlConfig {
	return &BenchctlConfig{
		Seed:      DEFAULT_SEED,
		Endpoints: []string{},
	}
}

func ReadConfig(path string) (*BenchctlConfig, error) {
	validate := validator.New()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	benchctlConfig := &BenchctlConfig{}
	err = json.Unmarshal(data, benchctlConfig)
	if err != nil {
		return nil, err
	}
	err = validate.Struct(benchctlConfig)
	if err != nil {
		return nil, err
	}
	return benchctlConfig, nil
}

func (cfg *BenchctlConfig) WriteConfig(path string) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return err
	}
	return nil
}
