package generator

import (
	"bytes"
	config "csb/control/config"
	"math/rand"
	"reflect"
	"sort"
	"testing"
)

func TestGenerateDataDeterminism(t *testing.T) {
	// Test cases with different counts
	defaultCfg := config.GetDefaultConfig()
	testCases := []struct {
		name      string
		count     int
		seed      int64
		keySize   int
		valueSize int
	}{
		{"Small dataset", 1000, 42, defaultCfg.KeySize, defaultCfg.ValueSize},
		{"Medium dataset", 10000, 42, defaultCfg.KeySize, defaultCfg.ValueSize},
		{"Large dataset", 100000, 42, defaultCfg.KeySize, defaultCfg.ValueSize},
		{"Extra	Large dataset", 1000000, 42, defaultCfg.KeySize, defaultCfg.ValueSize},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// First execution
			rg1 := rand.New(rand.NewSource(tc.seed))
			gen1 := NewGenerator(rg1)
			data1, _ := gen1.GenerateData(tc.count, tc.keySize, tc.valueSize)

			// Second execution
			rg2 := rand.New(rand.NewSource(tc.seed))
			gen2 := NewGenerator(rg2)
			data2, _ := gen2.GenerateData(tc.count, tc.keySize, tc.valueSize)

			// Check if both executions generated the same number of items
			if len(data1) != len(data2) {
				t.Errorf("Different number of items generated: got %d and %d, want same count",
					len(data1), len(data2))
			}

			// Check if both executions generated the same keys
			if !reflect.DeepEqual(getOrderedKeys(data1), getOrderedKeys(data2)) {
				t.Error("Different keys generated between executions")
			}

			// Check if values are identical for each key
			for k := range data1 {
				if !bytes.Equal(data1[k], data2[k]) {
					t.Errorf("Different values generated for key %s", k)
				}
			}

			// Check if the values are the expected size
			for k, v := range data1 {
				if len(v) != tc.valueSize {
					t.Errorf("Wrong size for value of key %s: got %d, want %d",
						k, len(v), tc.valueSize)
				}
			}

			// Verify we got the requested number of items
			if len(data1) != tc.count {
				t.Errorf("Wrong number of items generated: got %d, want %d",
					len(data1), tc.count)
			}
		})
	}
}

// Helper function to get sorted keys from a map
func getOrderedKeys(data map[string][]byte) []string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
