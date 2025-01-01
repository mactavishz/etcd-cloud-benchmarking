// Description: This package provides a schema for the synthetic key-value pairs and the logic to generate them.
// We assume that all textual information is in English, so we only handle ASCII characters.
package generator

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"

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
	"na3", // North America 3
	"na4", // North America 4
	"na5", // North America 5
	"na6", // North America 6
	"na7", // North America 7
	"na8", // North America 8
	"na9", // North America 9
	"eu1", // Europe 1
	"eu2", // Europe 2
	"eu3", // Europe 3
	"eu4", // Europe 4
	"eu5", // Europe 5
	"eu6", // Europe 6
	"eu7", // Europe 7
	"eu8", // Europe 8
	"eu9", // Europe 9
	"ap1", // Asia Pacific 1
	"ap2", // Asia Pacific 2
	"ap3", // Asia Pacific 3
	"ap4", // Asia Pacific 4
	"ap5", // Asia Pacific 5
	"ap6", // Asia Pacific 6
	"ap7", // Asia Pacific 7
	"ap8", // Asia Pacific 8
	"ap9", // Asia Pacific 9
	"afr", // Africa
	"mea", // Middle East
}

type Generator struct {
	rg *rand.Rand
	// Track used combinations for each domain/region
	usedCombinations map[string]map[string]map[int]map[string]bool
}

func NewGenerator(rg *rand.Rand) *Generator {
	return &Generator{
		rg:               rg,
		usedCombinations: make(map[string]map[string]map[int]map[string]bool),
	}
}

func (g *Generator) NewRand(seed int64, id int) *rand.Rand {
	// Create unique but deterministic seed for each goroutine
	uniqueSeed := seed + int64(id)
	return rand.New(rand.NewSource(uniqueSeed))
}

// generateUniquePadding creates a unique padding for a given prefix
func (g *Generator) generateUniquePadding(paddingSize int, prefix string) string {
	const charPool = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for {
		padding := make([]byte, paddingSize)
		for i := range padding {
			padding[i] = charPool[g.rg.Intn(len(charPool))]
		}
		if !g.isUsedPadding(prefix, string(padding)) {
			return string(padding)
		}
	}
}

// isUsedPadding checks if a padding has been used for a given prefix
func (g *Generator) isUsedPadding(prefix string, padding string) bool {
	parts := strings.Split(strings.Trim(prefix, "/"), "/")
	domain, region := parts[0], parts[1]
	shard := parts[2]
	shardNum := 0
	fmt.Sscanf(shard, "%d", &shardNum)

	if g.usedCombinations[domain] == nil {
		g.usedCombinations[domain] = make(map[string]map[int]map[string]bool)
	}
	if g.usedCombinations[domain][region] == nil {
		g.usedCombinations[domain][region] = make(map[int]map[string]bool)
	}
	if g.usedCombinations[domain][region][shardNum] == nil {
		g.usedCombinations[domain][region][shardNum] = make(map[string]bool)
	}

	if g.usedCombinations[domain][region][shardNum][padding] {
		return true
	}

	g.usedCombinations[domain][region][shardNum][padding] = true
	return false
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

	prefix := fmt.Sprintf("/%s/%s/%s", domain, region, shard)
	if paddingSize == 0 {
		if !g.isUsedPadding(prefix, "") {
			return prefix, nil
		}
		// If this combination is used, we need to try another one
		return g.GenerateKey(targetSize)
	}

	padding := g.generateUniquePadding(paddingSize, prefix)
	return prefix + padding, nil
}

// GenerateValue creates a synthetic value for the resource
func (g *Generator) GenerateValue(targetBytes int, rg *rand.Rand) ([]byte, error) {
	if rg == nil {
		rg = g.rg
	}
	result := make([]byte, targetBytes)
	_, err := rg.Read(result)
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

		// Generate value of a given size in bytes
		value, err := g.GenerateValue(valueSize, nil)
		if err != nil {
			return nil, err
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
