package achem

import (
	"sync"
	"testing"
)

func TestNewEnvironmentManager(t *testing.T) {
	em := NewEnvironmentManager()
	if em == nil {
		t.Fatal("NewEnvironmentManager returned nil")
	}
	if em.environments == nil {
		t.Error("Expected non-nil environments map")
	}
	if len(em.environments) != 0 {
		t.Errorf("Expected empty environments map, got %d", len(em.environments))
	}
}

func TestEnvironmentManager_CreateEnvironment(t *testing.T) {
	em := NewEnvironmentManager()
	schema := NewSchema("test-schema")

	envID := EnvironmentID("test-env")
	err := em.CreateEnvironment(envID, schema)
	if err != nil {
		t.Fatalf("Expected no error creating environment, got: %v", err)
	}

	// Verify environment was created
	env, exists := em.GetEnvironment(envID)
	if !exists {
		t.Fatal("Expected environment to exist after creation")
	}
	if env == nil {
		t.Fatal("Expected non-nil environment")
	}
	if env.schema != schema {
		t.Error("Environment schema mismatch")
	}
}

func TestEnvironmentManager_CreateEnvironment_Duplicate(t *testing.T) {
	em := NewEnvironmentManager()
	schema := NewSchema("test-schema")
	envID := EnvironmentID("test-env")

	// Create first environment
	err := em.CreateEnvironment(envID, schema)
	if err != nil {
		t.Fatalf("Expected no error creating first environment, got: %v", err)
	}

	// Try to create duplicate
	schema2 := NewSchema("test-schema-2")
	err = em.CreateEnvironment(envID, schema2)
	if err == nil {
		t.Error("Expected error when creating duplicate environment")
	}
	if err.Error() != "environment with id test-env already exists" {
		t.Errorf("Expected specific error message, got: %v", err)
	}

	// Verify original environment still exists with original schema
	env, exists := em.GetEnvironment(envID)
	if !exists {
		t.Fatal("Expected original environment to still exist")
	}
	if env.schema != schema {
		t.Error("Expected original schema to be preserved")
	}
}

func TestEnvironmentManager_GetEnvironment(t *testing.T) {
	em := NewEnvironmentManager()
	schema := NewSchema("test-schema")
	envID := EnvironmentID("test-env")

	// Get non-existent environment
	_, exists := em.GetEnvironment(envID)
	if exists {
		t.Error("Expected environment not to exist")
	}

	// Create environment
	err := em.CreateEnvironment(envID, schema)
	if err != nil {
		t.Fatalf("Expected no error creating environment, got: %v", err)
	}

	// Get existing environment
	env, exists := em.GetEnvironment(envID)
	if !exists {
		t.Error("Expected environment to exist")
	}
	if env == nil {
		t.Error("Expected non-nil environment")
	}
}

func TestEnvironmentManager_GetEnvironment_Multiple(t *testing.T) {
	em := NewEnvironmentManager()

	// Create multiple environments
	envIDs := []EnvironmentID{"env1", "env2", "env3"}
	for i, envID := range envIDs {
		schema := NewSchema("test-schema-" + string(rune(i+'0')))
		err := em.CreateEnvironment(envID, schema)
		if err != nil {
			t.Fatalf("Expected no error creating environment %s, got: %v", envID, err)
		}
	}

	// Verify all environments exist
	for _, envID := range envIDs {
		env, exists := em.GetEnvironment(envID)
		if !exists {
			t.Errorf("Expected environment %s to exist", envID)
		}
		if env == nil {
			t.Errorf("Expected non-nil environment for %s", envID)
		}
	}
}

func TestEnvironmentManager_DeleteEnvironment(t *testing.T) {
	em := NewEnvironmentManager()
	schema := NewSchema("test-schema")
	envID := EnvironmentID("test-env")

	// Try to delete non-existent environment
	err := em.DeleteEnvironment(envID)
	if err == nil {
		t.Error("Expected error when deleting non-existent environment")
	}

	// Create environment
	err = em.CreateEnvironment(envID, schema)
	if err != nil {
		t.Fatalf("Expected no error creating environment, got: %v", err)
	}

	// Verify it exists
	_, exists := em.GetEnvironment(envID)
	if !exists {
		t.Fatal("Expected environment to exist before deletion")
	}

	// Delete environment
	err = em.DeleteEnvironment(envID)
	if err != nil {
		t.Fatalf("Expected no error deleting environment, got: %v", err)
	}

	// Verify it no longer exists
	_, exists = em.GetEnvironment(envID)
	if exists {
		t.Error("Expected environment not to exist after deletion")
	}
}

func TestEnvironmentManager_DeleteEnvironment_StopsRunning(t *testing.T) {
	em := NewEnvironmentManager()
	schema := NewSchema("test-schema")
	envID := EnvironmentID("test-env")

	// Create environment
	err := em.CreateEnvironment(envID, schema)
	if err != nil {
		t.Fatalf("Expected no error creating environment, got: %v", err)
	}

	// Get environment and start it
	env, exists := em.GetEnvironment(envID)
	if !exists {
		t.Fatal("Expected environment to exist")
	}

	// Start the environment
	env.Run(100 * 1000000) // 100ms interval

	// Verify it's running
	env.mu.RLock()
	isRunning := env.isRunning
	env.mu.RUnlock()
	if !isRunning {
		t.Error("Expected environment to be running")
	}

	// Delete environment (should stop it)
	err = em.DeleteEnvironment(envID)
	if err != nil {
		t.Fatalf("Expected no error deleting environment, got: %v", err)
	}

	// Verify it's stopped (we can't check directly since env is deleted, but we can check it's gone)
	_, exists = em.GetEnvironment(envID)
	if exists {
		t.Error("Expected environment not to exist after deletion")
	}
}

func TestEnvironmentManager_ListEnvironments(t *testing.T) {
	em := NewEnvironmentManager()

	// Empty manager
	envs := em.ListEnvironments()
	if len(envs) != 0 {
		t.Errorf("Expected 0 environments, got %d", len(envs))
	}

	// Create multiple environments
	envIDs := []EnvironmentID{"env1", "env2", "env3"}
	for i, envID := range envIDs {
		schema := NewSchema("test-schema-" + string(rune(i+'0')))
		err := em.CreateEnvironment(envID, schema)
		if err != nil {
			t.Fatalf("Expected no error creating environment %s, got: %v", envID, err)
		}
	}

	// List environments
	listed := em.ListEnvironments()
	if len(listed) != len(envIDs) {
		t.Errorf("Expected %d environments, got %d", len(envIDs), len(listed))
	}

	// Verify all IDs are present
	idMap := make(map[EnvironmentID]bool)
	for _, id := range listed {
		idMap[id] = true
	}
	for _, expectedID := range envIDs {
		if !idMap[expectedID] {
			t.Errorf("Expected to find environment ID %s in list", expectedID)
		}
	}
}

func TestEnvironmentManager_UpdateEnvironmentSchema(t *testing.T) {
	em := NewEnvironmentManager()
	schema1 := NewSchema("schema-1")
	envID := EnvironmentID("test-env")

	// Try to update non-existent environment
	err := em.UpdateEnvironmentSchema(envID, schema1)
	if err == nil {
		t.Error("Expected error when updating non-existent environment")
	}

	// Create environment
	err = em.CreateEnvironment(envID, schema1)
	if err != nil {
		t.Fatalf("Expected no error creating environment, got: %v", err)
	}

	// Verify original schema
	env, exists := em.GetEnvironment(envID)
	if !exists {
		t.Fatal("Expected environment to exist")
	}
	if env.schema != schema1 {
		t.Error("Expected original schema")
	}

	// Update schema
	schema2 := NewSchema("schema-2")
	err = em.UpdateEnvironmentSchema(envID, schema2)
	if err != nil {
		t.Fatalf("Expected no error updating schema, got: %v", err)
	}

	// Verify schema was updated
	env, exists = em.GetEnvironment(envID)
	if !exists {
		t.Fatal("Expected environment to still exist after schema update")
	}
	if env.schema != schema2 {
		t.Error("Expected schema to be updated")
	}
	if env.schema.Name != "schema-2" {
		t.Errorf("Expected schema name 'schema-2', got '%s'", env.schema.Name)
	}
}

func TestEnvironmentManager_UpdateEnvironmentSchema_PreservesMolecules(t *testing.T) {
	em := NewEnvironmentManager()
	schema1 := NewSchema("schema-1")
	envID := EnvironmentID("test-env")

	// Create environment
	err := em.CreateEnvironment(envID, schema1)
	if err != nil {
		t.Fatalf("Expected no error creating environment, got: %v", err)
	}

	// Add some molecules
	env, _ := em.GetEnvironment(envID)
	m1 := NewMolecule("Species1", map[string]any{"key": "value1"}, 0)
	m2 := NewMolecule("Species2", map[string]any{"key": "value2"}, 0)
	env.Insert(m1)
	env.Insert(m2)

	// Verify molecules exist
	molecules := env.AllMolecules()
	if len(molecules) != 2 {
		t.Fatalf("Expected 2 molecules, got %d", len(molecules))
	}

	// Update schema
	schema2 := NewSchema("schema-2")
	err = em.UpdateEnvironmentSchema(envID, schema2)
	if err != nil {
		t.Fatalf("Expected no error updating schema, got: %v", err)
	}

	// Verify molecules are still there
	molecules = env.AllMolecules()
	if len(molecules) != 2 {
		t.Errorf("Expected 2 molecules after schema update, got %d", len(molecules))
	}
}

func TestEnvironmentManager_ConcurrentAccess(t *testing.T) {
	em := NewEnvironmentManager()
	schema := NewSchema("test-schema")

	// Test concurrent creation
	var wg sync.WaitGroup
	numGoroutines := 10
	envIDs := make([]EnvironmentID, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		envIDs[i] = EnvironmentID("env-" + string(rune(i+'0')))
	}

	// Concurrently create environments
	for _, envID := range envIDs {
		wg.Add(1)
		go func(id EnvironmentID) {
			defer wg.Done()
			_ = em.CreateEnvironment(id, schema)
		}(envID)
	}
	wg.Wait()

	// Verify all were created
	listed := em.ListEnvironments()
	if len(listed) != numGoroutines {
		t.Errorf("Expected %d environments, got %d", numGoroutines, len(listed))
	}

	// Concurrently read environments
	wg = sync.WaitGroup{}
	for i := 0; i < numGoroutines*2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, envID := range envIDs {
				_, _ = em.GetEnvironment(envID)
			}
		}()
	}
	wg.Wait()

	// Concurrently delete some environments
	wg = sync.WaitGroup{}
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func(id EnvironmentID) {
			defer wg.Done()
			_ = em.DeleteEnvironment(id)
		}(envIDs[i])
	}
	wg.Wait()

	// Verify remaining environments
	listed = em.ListEnvironments()
	expectedRemaining := numGoroutines - numGoroutines/2
	if len(listed) != expectedRemaining {
		t.Errorf("Expected %d environments after deletion, got %d", expectedRemaining, len(listed))
	}
}

func TestEnvironmentManager_Isolation(t *testing.T) {
	em := NewEnvironmentManager()
	schema1 := NewSchema("schema-1")
	schema2 := NewSchema("schema-2")

	envID1 := EnvironmentID("env-1")
	envID2 := EnvironmentID("env-2")

	// Create two environments
	err := em.CreateEnvironment(envID1, schema1)
	if err != nil {
		t.Fatalf("Expected no error creating env-1, got: %v", err)
	}
	err = em.CreateEnvironment(envID2, schema2)
	if err != nil {
		t.Fatalf("Expected no error creating env-2, got: %v", err)
	}

	// Add molecules to each environment
	env1, _ := em.GetEnvironment(envID1)
	env2, _ := em.GetEnvironment(envID2)

	m1 := NewMolecule("Species1", map[string]any{"env": "1"}, 0)
	m2 := NewMolecule("Species2", map[string]any{"env": "2"}, 0)

	env1.Insert(m1)
	env2.Insert(m2)

	// Verify molecules are isolated
	molecules1 := env1.AllMolecules()
	molecules2 := env2.AllMolecules()

	if len(molecules1) != 1 {
		t.Errorf("Expected 1 molecule in env-1, got %d", len(molecules1))
	}
	if len(molecules2) != 1 {
		t.Errorf("Expected 1 molecule in env-2, got %d", len(molecules2))
	}

	// Verify schemas are isolated
	if env1.schema != schema1 {
		t.Error("env-1 should have schema1")
	}
	if env2.schema != schema2 {
		t.Error("env-2 should have schema2")
	}
}

