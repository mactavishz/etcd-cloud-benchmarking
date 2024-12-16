// Description: This package provides a schema for the synthetic K8s data generator.
// We assume that all textual information is in English, so we only handle ASCII characters.
package generator

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strings"

	"github.com/google/uuid"
)

var DEFAULT_RESOURCE_TYPES = []string{"pods", "services", "configmaps", "secrets", "deployments"}
var DEFAULT_NAMESPACES = []string{"default", "kube-system", "monitoring", "application"}
var DEFAULT_RV int64 = 1

type Generator struct {
	resourceTypes []string
	namespaces    []string
	rv            int64 // Resource version counter
	rg            *rand.Rand
}

// Represents common metadata for K8s resources
type ResourceMetadata struct {
	APIVersion string          `json:"apiVersion"`
	Kind       string          `json:"kind"`
	Metadata   ObjectMetadata  `json:"metadata"`
	Spec       json.RawMessage `json:"spec"`
	Status     json.RawMessage `json:"status"`
}

// Represents K8s object metadata
type ObjectMetadata struct {
	Name            string            `json:"name"`
	Namespace       string            `json:"namespace,omitempty"`
	Labels          map[string]string `json:"labels,omitempty"`
	ResourceVersion string            `json:"resourceVersion"`
}

func NewGenerator(rg *rand.Rand) *Generator {
	return &Generator{
		resourceTypes: DEFAULT_RESOURCE_TYPES[:],
		namespaces:    DEFAULT_NAMESPACES[:],
		rv:            DEFAULT_RV,
		rg:            rg,
	}
}

func (g *Generator) GetRV() int64 {
	return g.rv
}

func (g *Generator) GetResourceTypes() []string {
	return g.resourceTypes[:]
}

func (g *Generator) GetNamespaces() []string {
	return g.namespaces[:]
}

// GenerateKey creates an etcd key following K8s format
func (g *Generator) GenerateKey(resourceType, namespace, name string) string {
	if namespace == "" {
		return fmt.Sprintf("/registry/%s/%s", resourceType, name)
	}
	return fmt.Sprintf("/registry/%s/%s/%s", resourceType, namespace, name)
}

// GenerateValue creates a synthetic value for the resource
func (g *Generator) GenerateValue(resourceType, namespace, name string) ([]byte, error) {
	g.rv++ // Increment resource version
	rand := g.rg

	resource := ResourceMetadata{
		APIVersion: "v1",
		Kind:       strings.TrimSuffix(strings.ToUpper(string(resourceType[0]))+string(resourceType[1:]), "s"),
		Metadata: ObjectMetadata{
			Name:            name,
			Namespace:       namespace,
			Labels:          g.generateLabels(),
			ResourceVersion: fmt.Sprintf("%d", g.rv),
		},
	}

	// Add dummy spec and status data
	spec := map[string]interface{}{
		"replicas": rand.Intn(5) + 1,
		"selector": map[string]interface{}{
			"matchLabels": resource.Metadata.Labels,
		},
	}

	status := map[string]interface{}{
		"availableReplicas": spec["replicas"],
		"readyReplicas":     spec["replicas"],
		"updatedReplicas":   spec["replicas"],
	}

	specBytes, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	resource.Spec = specBytes

	statusBytes, err := json.Marshal(status)
	if err != nil {
		return nil, err
	}
	resource.Status = statusBytes

	return json.Marshal(resource)
}

func (g *Generator) generateLabels() map[string]string {
	rand := g.rg
	labels := make(map[string]string)
	environments := []string{"dev", "staging", "prod", "test"}
	teams := []string{"frontend", "backend", "data", "platform", "devops", "ml"}

	labels["environment"] = environments[rand.Intn(len(environments))]
	labels["team"] = teams[rand.Intn(len(teams))]

	return labels
}

func (g *Generator) generateName(resourceType string) string {
	rand := g.rg
	nouns := []string{"nginx", "redis", "postgres", "website", "nodejs", "mysql", "kafka", "zookeeper", "rabbitmq"}
	regions := []string{"us-west", "us-east", "eu-west", "eu-east", "asia", "africa", "meast"}
	return fmt.Sprintf("%s-%s-%s%d-%s",
		resourceType[:3],
		nouns[rand.Intn(len(nouns))],
		regions[rand.Intn(len(regions))],
		rand.Intn(10),
		uuid.New().String())
}

func (g *Generator) GenerateData(count int) map[string][]byte {
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
		resourceType := g.resourceTypes[rand.Intn(len(g.resourceTypes))]
		namespace := g.namespaces[rand.Intn(len(g.namespaces))]
		name := g.generateName(resourceType)

		key := g.GenerateKey(resourceType, namespace, name)

		// Check for duplicates before generating value
		if _, exists := data[key]; exists {
			fmt.Printf("Duplicate key generated: %s, will try again\n", key)
			i--
			continue
		}

		value, err := g.GenerateValue(resourceType, namespace, name)
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

	return data
}
