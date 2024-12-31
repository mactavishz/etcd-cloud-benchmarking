// Description: This package provides a schema for the synthetic key-value pairs and the logic to generate them.
// We assume that all textual information is in English, so we only handle ASCII characters.
package generator

import (
	"fmt"
	"math/rand"
	"sort"

	"csb/control/constants"

	"github.com/google/uuid"
)

// Minimum key size calculation:
// slash (1) + domain (3) + slash (1) + region (3) + slash (1) + shard (3)  = 12 bytes
// See module csb/control/constants/constants.go for the values of MIN_KEY_SIZE

// Domain prefixes represent different business domains
var domains = []string{
	"usr", // user-related data
	"ord", // order-related data
	"prd", // product-related data
	"inv", // inventory-related data
	"sys", // system-related data
	"app", // application-related data
	"etc", // config or other data
	"var", // variable data
}

// Region codes for geographical distribution
var regions = []string{
	"na1", // North America 1
	"na2", // North America 2
	"eu1", // Europe 1
	"eu2", // Europe 2
	"ap1", // Asia Pacific 1
	"ap2", // Asia Pacific 2
	"afr", // Africa
	"mea", // Middle East
}

type Generator struct {
	rg *rand.Rand
}

func NewGenerator(rg *rand.Rand) *Generator {
	return &Generator{
		rg: rg,
	}
}

// GenerateKey creates an etcd key following K8s format
func (g *Generator) GenerateKey(targetSize int) (string, error) {
	if targetSize < constants.MIN_KEY_SIZE {
		return "", fmt.Errorf("target size %d is less than minimum required size %d", targetSize, constants.MIN_KEY_SIZE)
	}

	domain := domains[g.rg.Intn(len(domains))]
	region := regions[g.rg.Intn(len(regions))]
	shard := fmt.Sprintf("%03d", g.rg.Intn(1000))

	// Calculate required padding size
	// prefix = slash + domain + slash + region + slash + shard = 12 bytes
	prefixLen := 1 + len(domain) + 1 + len(region) + 1 + len(shard)
	paddingSize := targetSize - prefixLen

	if paddingSize == 0 {
		return fmt.Sprintf("/%s/%s/%s", domain, region, shard), nil
	}

	// Generate padding using hex characters
	const charPool = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	padding := make([]byte, paddingSize)
	for i := range padding {
		padding[i] = charPool[g.rg.Intn(len(charPool))]
	}

	return fmt.Sprintf("/%s/%s/%s%s", domain, region, shard, string(padding)), nil
}

// GenerateValue creates a synthetic value for the resource
func (g *Generator) GenerateValue(targetBytes int) ([]byte, error) {
	result := make([]byte, targetBytes)
	_, err := g.rg.Read(result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (g *Generator) GenerateData(count int, keySize int, valueSize int) (map[string][]byte, error) {
	rand := g.rg
	uuid.SetRand(rand)
	data := make(map[string][]byte)

	// Generate all entries first in a deterministic order
	type entry struct {
		key   string
		value []byte
	}

	entries := make([]entry, 0, count)

	for i := 0; i < count; i++ {
		key, err := g.GenerateKey(keySize)

		if err != nil {
			return nil, err
		}

		// Check for duplicates before generating value
		if _, exists := data[key]; exists {
			fmt.Printf("Duplicate key generated: %s, will try again\n", key)
			i--
			continue
		}

		// Generate value of a given size in bytes
		value, err := g.GenerateValue(valueSize)
		if err != nil {
			fmt.Printf("Error generating value for %s: %v, will try again\n", key, err)
			// try again
			i--
			continue
		}

		entries = append(entries, entry{key: key, value: value})
	}

	// Sort entries by key to ensure deterministic ordering
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].key < entries[j].key
	})

	// Populate the map in sorted order
	for _, e := range entries {
		data[e.key] = e.value
	}

	return data, nil
}
