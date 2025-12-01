package achem

import (
	"fmt"
	"sync"
)

// EnvironmentID is a unique identifier for an environment
type EnvironmentID string

// EnvironmentManager manages multiple environments, each isolated from others
type EnvironmentManager struct {
	mu          sync.RWMutex
	environments map[EnvironmentID]*Environment
}

// NewEnvironmentManager creates a new environment manager
func NewEnvironmentManager() *EnvironmentManager {
	return &EnvironmentManager{
		environments: make(map[EnvironmentID]*Environment),
	}
}

// CreateEnvironment creates a new environment with the given ID and schema
// Returns an error if an environment with that ID already exists
func (em *EnvironmentManager) CreateEnvironment(id EnvironmentID, schema *Schema) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	if _, exists := em.environments[id]; exists {
		return fmt.Errorf("environment with id %s already exists", id)
	}

	env := NewEnvironment(schema)
	em.environments[id] = env
	return nil
}

// GetEnvironment retrieves an environment by ID
// Returns the environment and a boolean indicating if it was found
func (em *EnvironmentManager) GetEnvironment(id EnvironmentID) (*Environment, bool) {
	em.mu.RLock()
	defer em.mu.RUnlock()

	env, exists := em.environments[id]
	return env, exists
}

// DeleteEnvironment removes an environment by ID
// Returns an error if the environment doesn't exist
func (em *EnvironmentManager) DeleteEnvironment(id EnvironmentID) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	if _, exists := em.environments[id]; !exists {
		return fmt.Errorf("environment with id %s does not exist", id)
	}

	// Stop the environment if it's running
	env := em.environments[id]
	env.Stop()

	delete(em.environments, id)
	return nil
}

// ListEnvironments returns a list of all environment IDs
func (em *EnvironmentManager) ListEnvironments() []EnvironmentID {
	em.mu.RLock()
	defer em.mu.RUnlock()

	ids := make([]EnvironmentID, 0, len(em.environments))
	for id := range em.environments {
		ids = append(ids, id)
	}
	return ids
}

// UpdateEnvironmentSchema updates the schema of an existing environment
// This will replace the schema but keep all existing molecules
func (em *EnvironmentManager) UpdateEnvironmentSchema(id EnvironmentID, schema *Schema) error {
	em.mu.RLock()
	env, exists := em.environments[id]
	em.mu.RUnlock()

	if !exists {
		return fmt.Errorf("environment with id %s does not exist", id)
	}

	// Update the schema (we're in the same package, so we can access the field directly)
	env.mu.Lock()
	env.schema = schema
	env.mu.Unlock()

	return nil
}

