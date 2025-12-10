package main

import (
	"encoding/json"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/daniacca/achemdb/internal/achem"
)

func TestServer_HandleSaveSnapshot(t *testing.T) {
	logger := NewLogger("info")
	srv := NewServer(logger)
	tmpDir := t.TempDir()
	srv.SetSnapshotDir(tmpDir)

	// Create environment with schema
	schema := achem.NewSchema("test")
	schema = schema.WithSpecies(achem.Species{Name: "TestSpecies"})
	
	envID := achem.EnvironmentID("test-env")
	if err := srv.manager.CreateEnvironment(envID, schema); err != nil {
		t.Fatalf("Failed to create environment: %v", err)
	}

	// Get environment and set snapshot dir
	env, exists := srv.manager.GetEnvironment(envID)
	if !exists {
		t.Fatal("Environment not found")
	}
	env.SetSnapshotDir(tmpDir)

	// Insert some molecules
	m1 := achem.NewMolecule("TestSpecies", map[string]any{"key1": "value1"}, 0)
	m2 := achem.NewMolecule("TestSpecies", map[string]any{"key2": 42}, 0)
	env.Insert(m1)
	env.Insert(m2)

	// Step a few times to increment time
	for i := 0; i < 5; i++ {
		env.Step()
	}

	// Create request
	req := httptest.NewRequest(http.MethodPost, "/env/test-env/snapshot", nil)
	w := httptest.NewRecorder()

	// Call handler
	srv.handleSaveSnapshot(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Parse response
	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", response["status"])
	}

	if response["path"] == "" {
		t.Error("Expected non-empty path in response")
	}

	// Verify snapshot file exists
	expectedPath := filepath.Join(tmpDir, "test-env.snapshot.json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatalf("Expected snapshot file to exist at %s", expectedPath)
	}

	// Verify snapshot content
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read snapshot file: %v", err)
	}

	var snapshot achem.Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("Failed to decode snapshot: %v", err)
	}

	if snapshot.EnvironmentID != envID {
		t.Errorf("Expected EnvironmentID %s, got %s", envID, snapshot.EnvironmentID)
	}

	if snapshot.Time < 5 {
		t.Errorf("Expected Time >= 5, got %d", snapshot.Time)
	}

	if len(snapshot.Molecules) != 2 {
		t.Errorf("Expected 2 molecules, got %d", len(snapshot.Molecules))
	}
}

func TestServer_HandleGetSnapshot(t *testing.T) {
	logger := NewLogger("info")
	srv := NewServer(logger)
	tmpDir := t.TempDir()
	srv.SetSnapshotDir(tmpDir)

	// Create environment with schema
	schema := achem.NewSchema("test")
	schema = schema.WithSpecies(achem.Species{Name: "TestSpecies"})
	
	envID := achem.EnvironmentID("test-env")
	if err := srv.manager.CreateEnvironment(envID, schema); err != nil {
		t.Fatalf("Failed to create environment: %v", err)
	}

	// Get environment and set snapshot dir
	env, exists := srv.manager.GetEnvironment(envID)
	if !exists {
		t.Fatal("Environment not found")
	}
	env.SetSnapshotDir(tmpDir)

	// Insert molecules and create snapshot
	m1 := achem.NewMolecule("TestSpecies", map[string]any{"key1": "value1"}, 0)
	m2 := achem.NewMolecule("TestSpecies", map[string]any{"key2": 42}, 0)
	env.Insert(m1)
	env.Insert(m2)

	// Step a few times to increment time
	for i := 0; i < 10; i++ {
		env.Step()
	}

	// Save snapshot
	if err := env.SaveSnapshot(); err != nil {
		t.Fatalf("Failed to save snapshot: %v", err)
	}

	// Create GET request
	req := httptest.NewRequest(http.MethodGet, "/env/test-env/snapshot", nil)
	w := httptest.NewRecorder()

	// Call handler
	srv.handleGetSnapshot(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify Content-Type
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", w.Header().Get("Content-Type"))
	}

	// Parse response as snapshot
	var snapshot achem.Snapshot
	if err := json.Unmarshal(w.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("Failed to decode snapshot JSON: %v", err)
	}

	// Verify snapshot structure
	if snapshot.EnvironmentID != envID {
		t.Errorf("Expected EnvironmentID %s, got %s", envID, snapshot.EnvironmentID)
	}

	if snapshot.Time < 10 {
		t.Errorf("Expected Time >= 10, got %d", snapshot.Time)
	}

	if len(snapshot.Molecules) != 2 {
		t.Errorf("Expected 2 molecules, got %d", len(snapshot.Molecules))
	}

	// Verify molecules
	moleculeMap := make(map[achem.MoleculeID]achem.Molecule)
	for _, m := range snapshot.Molecules {
		moleculeMap[m.ID] = m
	}

	if _, ok := moleculeMap[m1.ID]; !ok {
		t.Error("Expected m1 to be in snapshot")
	}

	if _, ok := moleculeMap[m2.ID]; !ok {
		t.Error("Expected m2 to be in snapshot")
	}
}

func TestServer_HandleGetSnapshot_NotFound(t *testing.T) {
	logger := NewLogger("info")
	srv := NewServer(logger)
	tmpDir := t.TempDir()
	srv.SetSnapshotDir(tmpDir)

	// Create environment
	schema := achem.NewSchema("test")
	envID := achem.EnvironmentID("test-env")
	if err := srv.manager.CreateEnvironment(envID, schema); err != nil {
		t.Fatalf("Failed to create environment: %v", err)
	}

	// Create GET request for non-existent snapshot
	req := httptest.NewRequest(http.MethodGet, "/env/test-env/snapshot", nil)
	w := httptest.NewRecorder()

	// Call handler
	srv.handleGetSnapshot(w, req)

	// Check response
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestServer_HandleSaveSnapshot_NoSnapshotDir(t *testing.T) {
	logger := NewLogger("info")
	srv := NewServer(logger)
	// Don't set snapshot directory

	// Create environment
	schema := achem.NewSchema("test")
	envID := achem.EnvironmentID("test-env")
	if err := srv.manager.CreateEnvironment(envID, schema); err != nil {
		t.Fatalf("Failed to create environment: %v", err)
	}

	// Create request
	req := httptest.NewRequest(http.MethodPost, "/env/test-env/snapshot", nil)
	w := httptest.NewRecorder()

	// Call handler
	srv.handleSaveSnapshot(w, req)

	// Check response
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLoadServerConfig_Defaults(t *testing.T) {
	// Save original env vars
	origAddr := os.Getenv("ACHEMDB_ADDR")
	origEnvID := os.Getenv("ACHEMDB_ENV_ID")
	origSchemaFile := os.Getenv("ACHEMDB_SCHEMA_FILE")

	// Clean up env vars
	os.Unsetenv("ACHEMDB_ADDR")
	os.Unsetenv("ACHEMDB_ENV_ID")
	os.Unsetenv("ACHEMDB_SCHEMA_FILE")

	// Reset flag state
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{"achemdb-server"}

	// Restore env vars after test
	defer func() {
		if origAddr != "" {
			os.Setenv("ACHEMDB_ADDR", origAddr)
		}
		if origEnvID != "" {
			os.Setenv("ACHEMDB_ENV_ID", origEnvID)
		}
		if origSchemaFile != "" {
			os.Setenv("ACHEMDB_SCHEMA_FILE", origSchemaFile)
		}
	}()

	cfg := loadServerConfig()

	if cfg.Addr != ":8080" {
		t.Errorf("Expected Addr to be ':8080', got '%s'", cfg.Addr)
	}
	if cfg.DefaultEnvID != "default" {
		t.Errorf("Expected DefaultEnvID to be 'default', got '%s'", cfg.DefaultEnvID)
	}
	if cfg.SchemaFile != "" {
		t.Errorf("Expected SchemaFile to be empty, got '%s'", cfg.SchemaFile)
	}
}

func TestLoadServerConfig_EnvVars(t *testing.T) {
	// Save original env vars
	origAddr := os.Getenv("ACHEMDB_ADDR")
	origEnvID := os.Getenv("ACHEMDB_ENV_ID")
	origSchemaFile := os.Getenv("ACHEMDB_SCHEMA_FILE")

	// Set test env vars
	os.Setenv("ACHEMDB_ADDR", ":9090")
	os.Setenv("ACHEMDB_ENV_ID", "test-env")
	os.Setenv("ACHEMDB_SCHEMA_FILE", "/path/to/schema.json")

	// Reset flag state
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{"achemdb-server"}

	// Restore env vars after test
	defer func() {
		if origAddr != "" {
			os.Setenv("ACHEMDB_ADDR", origAddr)
		} else {
			os.Unsetenv("ACHEMDB_ADDR")
		}
		if origEnvID != "" {
			os.Setenv("ACHEMDB_ENV_ID", origEnvID)
		} else {
			os.Unsetenv("ACHEMDB_ENV_ID")
		}
		if origSchemaFile != "" {
			os.Setenv("ACHEMDB_SCHEMA_FILE", origSchemaFile)
		} else {
			os.Unsetenv("ACHEMDB_SCHEMA_FILE")
		}
	}()

	cfg := loadServerConfig()

	if cfg.Addr != ":9090" {
		t.Errorf("Expected Addr to be ':9090', got '%s'", cfg.Addr)
	}
	if cfg.DefaultEnvID != "test-env" {
		t.Errorf("Expected DefaultEnvID to be 'test-env', got '%s'", cfg.DefaultEnvID)
	}
	if cfg.SchemaFile != "/path/to/schema.json" {
		t.Errorf("Expected SchemaFile to be '/path/to/schema.json', got '%s'", cfg.SchemaFile)
	}
}

func TestLoadServerConfig_FlagsOverrideEnvVars(t *testing.T) {
	// Save original env vars
	origAddr := os.Getenv("ACHEMDB_ADDR")
	origEnvID := os.Getenv("ACHEMDB_ENV_ID")
	origSchemaFile := os.Getenv("ACHEMDB_SCHEMA_FILE")

	// Set env vars
	os.Setenv("ACHEMDB_ADDR", ":9090")
	os.Setenv("ACHEMDB_ENV_ID", "env-env")
	os.Setenv("ACHEMDB_SCHEMA_FILE", "/env/schema.json")

	// Reset flag state and set flags
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{"achemdb-server", "-addr", ":7070", "-env-id", "flag-env", "-schema-file", "/flag/schema.json"}

	// Restore env vars after test
	defer func() {
		if origAddr != "" {
			os.Setenv("ACHEMDB_ADDR", origAddr)
		} else {
			os.Unsetenv("ACHEMDB_ADDR")
		}
		if origEnvID != "" {
			os.Setenv("ACHEMDB_ENV_ID", origEnvID)
		} else {
			os.Unsetenv("ACHEMDB_ENV_ID")
		}
		if origSchemaFile != "" {
			os.Setenv("ACHEMDB_SCHEMA_FILE", origSchemaFile)
		} else {
			os.Unsetenv("ACHEMDB_SCHEMA_FILE")
		}
	}()

	cfg := loadServerConfig()

	if cfg.Addr != ":7070" {
		t.Errorf("Expected Addr to be ':7070' (from flag), got '%s'", cfg.Addr)
	}
	if cfg.DefaultEnvID != "flag-env" {
		t.Errorf("Expected DefaultEnvID to be 'flag-env' (from flag), got '%s'", cfg.DefaultEnvID)
	}
	if cfg.SchemaFile != "/flag/schema.json" {
		t.Errorf("Expected SchemaFile to be '/flag/schema.json' (from flag), got '%s'", cfg.SchemaFile)
	}
}

func TestLoadInitialSchemaFromFile_ValidSchema(t *testing.T) {
	// Create a temporary JSON file with a valid schema
	tmpFile := filepath.Join(t.TempDir(), "schema.json")
	
	validSchema := achem.SchemaConfig{
		Name: "test-schema",
		Species: []achem.SpeciesConfig{
			{Name: "TestSpecies", Description: "A test species"},
		},
		Reactions: []achem.ReactionConfig{
			{
				ID:   "test-reaction",
				Name: "Test Reaction",
				Input: achem.InputConfig{
					Species: "TestSpecies",
				},
				Rate: 1.0,
				Effects: []achem.EffectConfig{},
			},
		},
	}

	data, err := json.Marshal(validSchema)
	if err != nil {
		t.Fatalf("Failed to marshal schema: %v", err)
	}

	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		t.Fatalf("Failed to write schema file: %v", err)
	}

	cfg, schema, err := loadInitialSchemaFromFile(tmpFile)
	if err != nil {
		t.Fatalf("Expected no error loading valid schema, got: %v", err)
	}

	if cfg.Name != "test-schema" {
		t.Errorf("Expected schema name 'test-schema', got '%s'", cfg.Name)
	}

	if schema == nil {
		t.Fatal("Expected non-nil schema")
	}

	if schema.Name != "test-schema" {
		t.Errorf("Expected built schema name 'test-schema', got '%s'", schema.Name)
	}
}

func TestLoadInitialSchemaFromFile_MissingFile(t *testing.T) {
	_, _, err := loadInitialSchemaFromFile("/nonexistent/file.json")
	if err == nil {
		t.Error("Expected error when loading missing file")
	}
}

func TestLoadInitialSchemaFromFile_InvalidJSON(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "invalid.json")
	if err := os.WriteFile(tmpFile, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("Failed to write invalid JSON file: %v", err)
	}

	_, _, err := loadInitialSchemaFromFile(tmpFile)
	if err == nil {
		t.Error("Expected error when loading invalid JSON")
	}
}

func TestLoadInitialSchemaFromFile_InvalidSchema(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "invalid-schema.json")
	
	// Create a schema with invalid configuration (missing required fields)
	invalidSchema := map[string]interface{}{
		"name": "test",
		"reactions": []map[string]interface{}{
			{
				"id": "test-reaction",
				// Missing required fields like input, rate, effects
			},
		},
	}

	data, err := json.Marshal(invalidSchema)
	if err != nil {
		t.Fatalf("Failed to marshal invalid schema: %v", err)
	}

	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		t.Fatalf("Failed to write invalid schema file: %v", err)
	}

	_, _, err = loadInitialSchemaFromFile(tmpFile)
	if err == nil {
		t.Error("Expected error when loading invalid schema config")
	}
}

func TestLoadServerConfig_SnapshotDefaults(t *testing.T) {
	// Save original env vars
	origSnapshotDir := os.Getenv("ACHEMDB_SNAPSHOT_DIR")
	origSnapshotTicks := os.Getenv("ACHEMDB_SNAPSHOT_EVERY_TICKS")
	origLogLevel := os.Getenv("ACHEMDB_LOG_LEVEL")

	// Clean up env vars
	os.Unsetenv("ACHEMDB_SNAPSHOT_DIR")
	os.Unsetenv("ACHEMDB_SNAPSHOT_EVERY_TICKS")
	os.Unsetenv("ACHEMDB_LOG_LEVEL")

	// Reset flag state
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{"achemdb-server"}

	// Restore env vars after test
	defer func() {
		if origSnapshotDir != "" {
			os.Setenv("ACHEMDB_SNAPSHOT_DIR", origSnapshotDir)
		}
		if origSnapshotTicks != "" {
			os.Setenv("ACHEMDB_SNAPSHOT_EVERY_TICKS", origSnapshotTicks)
		}
		if origLogLevel != "" {
			os.Setenv("ACHEMDB_LOG_LEVEL", origLogLevel)
		}
	}()

	cfg := loadServerConfig()

	if cfg.SnapshotDir != "./data" {
		t.Errorf("Expected SnapshotDir to be './data', got '%s'", cfg.SnapshotDir)
	}
	if cfg.SnapshotEveryTicks != 1000 {
		t.Errorf("Expected SnapshotEveryTicks to be 1000, got %d", cfg.SnapshotEveryTicks)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("Expected LogLevel to be 'info', got '%s'", cfg.LogLevel)
	}
}

func TestLoadServerConfig_SnapshotEnvVars(t *testing.T) {
	// Save original env vars
	origSnapshotDir := os.Getenv("ACHEMDB_SNAPSHOT_DIR")
	origSnapshotTicks := os.Getenv("ACHEMDB_SNAPSHOT_EVERY_TICKS")
	origLogLevel := os.Getenv("ACHEMDB_LOG_LEVEL")

	// Set test env vars
	os.Setenv("ACHEMDB_SNAPSHOT_DIR", "/custom/snapshots")
	os.Setenv("ACHEMDB_SNAPSHOT_EVERY_TICKS", "500")
	os.Setenv("ACHEMDB_LOG_LEVEL", "debug")

	// Reset flag state
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{"achemdb-server"}

	// Restore env vars after test
	defer func() {
		if origSnapshotDir != "" {
			os.Setenv("ACHEMDB_SNAPSHOT_DIR", origSnapshotDir)
		} else {
			os.Unsetenv("ACHEMDB_SNAPSHOT_DIR")
		}
		if origSnapshotTicks != "" {
			os.Setenv("ACHEMDB_SNAPSHOT_EVERY_TICKS", origSnapshotTicks)
		} else {
			os.Unsetenv("ACHEMDB_SNAPSHOT_EVERY_TICKS")
		}
		if origLogLevel != "" {
			os.Setenv("ACHEMDB_LOG_LEVEL", origLogLevel)
		} else {
			os.Unsetenv("ACHEMDB_LOG_LEVEL")
		}
	}()

	cfg := loadServerConfig()

	if cfg.SnapshotDir != "/custom/snapshots" {
		t.Errorf("Expected SnapshotDir to be '/custom/snapshots', got '%s'", cfg.SnapshotDir)
	}
	if cfg.SnapshotEveryTicks != 500 {
		t.Errorf("Expected SnapshotEveryTicks to be 500, got %d", cfg.SnapshotEveryTicks)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("Expected LogLevel to be 'debug', got '%s'", cfg.LogLevel)
	}
}

func TestLoadServerConfig_SnapshotFlagsOverrideEnvVars(t *testing.T) {
	// Save original env vars
	origSnapshotDir := os.Getenv("ACHEMDB_SNAPSHOT_DIR")
	origSnapshotTicks := os.Getenv("ACHEMDB_SNAPSHOT_EVERY_TICKS")
	origLogLevel := os.Getenv("ACHEMDB_LOG_LEVEL")

	// Set env vars
	os.Setenv("ACHEMDB_SNAPSHOT_DIR", "/env/snapshots")
	os.Setenv("ACHEMDB_SNAPSHOT_EVERY_TICKS", "200")
	os.Setenv("ACHEMDB_LOG_LEVEL", "warn")

	// Reset flag state and set flags
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{"achemdb-server", "-snapshot-dir", "/flag/snapshots", "-snapshot-every-ticks", "300", "-log-level", "error"}

	// Restore env vars after test
	defer func() {
		if origSnapshotDir != "" {
			os.Setenv("ACHEMDB_SNAPSHOT_DIR", origSnapshotDir)
		} else {
			os.Unsetenv("ACHEMDB_SNAPSHOT_DIR")
		}
		if origSnapshotTicks != "" {
			os.Setenv("ACHEMDB_SNAPSHOT_EVERY_TICKS", origSnapshotTicks)
		} else {
			os.Unsetenv("ACHEMDB_SNAPSHOT_EVERY_TICKS")
		}
		if origLogLevel != "" {
			os.Setenv("ACHEMDB_LOG_LEVEL", origLogLevel)
		} else {
			os.Unsetenv("ACHEMDB_LOG_LEVEL")
		}
	}()

	cfg := loadServerConfig()

	if cfg.SnapshotDir != "/flag/snapshots" {
		t.Errorf("Expected SnapshotDir to be '/flag/snapshots' (from flag), got '%s'", cfg.SnapshotDir)
	}
	if cfg.SnapshotEveryTicks != 300 {
		t.Errorf("Expected SnapshotEveryTicks to be 300 (from flag), got %d", cfg.SnapshotEveryTicks)
	}
	if cfg.LogLevel != "error" {
		t.Errorf("Expected LogLevel to be 'error' (from flag), got '%s'", cfg.LogLevel)
	}
}

func TestLoadServerConfig_InvalidSnapshotTicks(t *testing.T) {
	// Save original env var
	origSnapshotTicks := os.Getenv("ACHEMDB_SNAPSHOT_EVERY_TICKS")

	// Set invalid env var
	os.Setenv("ACHEMDB_SNAPSHOT_EVERY_TICKS", "invalid")

	// Reset flag state
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{"achemdb-server"}

	// Restore env var after test
	defer func() {
		if origSnapshotTicks != "" {
			os.Setenv("ACHEMDB_SNAPSHOT_EVERY_TICKS", origSnapshotTicks)
		} else {
			os.Unsetenv("ACHEMDB_SNAPSHOT_EVERY_TICKS")
		}
	}()

	cfg := loadServerConfig()

	// Should fall back to default 1000 when invalid value is provided
	if cfg.SnapshotEveryTicks != 1000 {
		t.Errorf("Expected SnapshotEveryTicks to be 1000 (default) when invalid, got %d", cfg.SnapshotEveryTicks)
	}
}

func TestLogger_Levels(t *testing.T) {
	// Test debug level - should log everything
	var debugOutput []byte
	logger := NewLogger("debug")
	logger.Debugf("debug message")
	logger.Infof("info message")
	logger.Warnf("warn message")
	logger.Errorf("error message")

	// Test info level - should log info, warn, error
	logger = NewLogger("info")
	logger.Debugf("debug message") // Should not appear
	logger.Infof("info message")
	logger.Warnf("warn message")
	logger.Errorf("error message")

	// Test warn level - should log warn, error
	logger = NewLogger("warn")
	logger.Debugf("debug message") // Should not appear
	logger.Infof("info message")   // Should not appear
	logger.Warnf("warn message")
	logger.Errorf("error message")

	// Test error level - should log only error
	logger = NewLogger("error")
	logger.Debugf("debug message") // Should not appear
	logger.Infof("info message")   // Should not appear
	logger.Warnf("warn message")   // Should not appear
	logger.Errorf("error message")

	// Test case-insensitive parsing
	logger = NewLogger("DEBUG")
	if logger.level != LogLevelDebug {
		t.Errorf("Expected DEBUG to parse as LogLevelDebug, got %v", logger.level)
	}

	logger = NewLogger("INFO")
	if logger.level != LogLevelInfo {
		t.Errorf("Expected INFO to parse as LogLevelInfo, got %v", logger.level)
	}

	logger = NewLogger("WARN")
	if logger.level != LogLevelWarn {
		t.Errorf("Expected WARN to parse as LogLevelWarn, got %v", logger.level)
	}

	logger = NewLogger("ERROR")
	if logger.level != LogLevelError {
		t.Errorf("Expected ERROR to parse as LogLevelError, got %v", logger.level)
	}

	// Test invalid level - should default to info
	logger = NewLogger("invalid")
	if logger.level != LogLevelInfo {
		t.Errorf("Expected invalid level to default to LogLevelInfo, got %v", logger.level)
	}

	_ = debugOutput // Suppress unused variable warning
}

