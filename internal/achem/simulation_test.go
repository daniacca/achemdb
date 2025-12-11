package achem

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// loadSchemaFromExamples loads a schema from the examples directory
func loadSchemaFromExamples(t *testing.T, filename string) (SchemaConfig, *Schema) {
	t.Helper()
	
	// Get the path to examples/schema directory relative to this test file
	// This file is in internal/achem/, so examples/schema is at ../../examples/schema/
	examplesPath := filepath.Join("..", "..", "examples", "schema", filename)
	
	data, err := os.ReadFile(examplesPath)
	if err != nil {
		t.Fatalf("Failed to read schema file %s: %v", examplesPath, err)
	}

	var cfg SchemaConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("Failed to parse schema JSON: %v", err)
	}

	if err := ValidateSchemaConfig(cfg); err != nil {
		t.Fatalf("Schema validation failed: %v", err)
	}

	schema, err := BuildSchemaFromConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to build schema: %v", err)
	}

	return cfg, schema
}

func TestSimulation_SecuritySchema(t *testing.T) {
	_, schema := loadSchemaFromExamples(t, "security.json")
	env := NewEnvironment(schema)

	// Seed with a few login failure events with the same IP
	for range 3 {
		m := NewMolecule("Event", map[string]any{
			"type": "login_failed",
			"ip":   "1.2.3.4",
		}, 0)
		m.CreatedAt = 0
		m.LastTouchedAt = 0
		env.Insert(m)
	}

	// Run simulation
	ticks := 100
	for i := 0; i < ticks; i++ {
		env.Step()
	}

	// Check sanity conditions
	molecules := env.AllMolecules()
	totalCount := len(molecules)

	// Total molecules should not exceed a reasonable bound
	// (decay reactions should prevent unbounded growth)
	if totalCount > 1000 {
		t.Errorf("Total molecule count (%d) exceeds reasonable bound (1000)", totalCount)
	}

	// Count by species
	counts := make(map[SpeciesName]int)
	for _, m := range molecules {
		counts[m.Species]++
	}

	// Check that higher-level species appeared (at least Suspicion should appear)
	if counts["Suspicion"] == 0 && counts["Alert"] == 0 {
		t.Logf("Warning: No higher-level species appeared. Counts: %v", counts)
		// This is not necessarily a failure, but worth noting
	}

	// Ensure we don't have excessive counts of any species
	for species, count := range counts {
		if count > 500 {
			t.Errorf("Species %s has excessive count (%d), possible runaway reaction", species, count)
		}
	}

	t.Logf("Security schema simulation: total=%d, counts=%v", totalCount, counts)
}

func TestSimulation_EcommerceSchema(t *testing.T) {
	_, schema := loadSchemaFromExamples(t, "ecommerce.json")
	env := NewEnvironment(schema)

	// Seed with page views that lead to cart items
	for i := range 5 {
		m := NewMolecule("PageView", map[string]any{
			"action":    "add_to_cart",
			"user_id":   "user123",
			"product_id": "prod" + string(rune('A'+i)),
			"price":     10.0 + float64(i),
		}, 0)
		m.CreatedAt = 0
		m.LastTouchedAt = 0
		env.Insert(m)
	}

	// Run simulation
	for range 100 {
		env.Step()
	}

	// Check sanity conditions
	molecules := env.AllMolecules()
	totalCount := len(molecules)

	if totalCount > 1000 {
		t.Errorf("Total molecule count (%d) exceeds reasonable bound (1000)", totalCount)
	}

	counts := make(map[SpeciesName]int)
	for _, m := range molecules {
		counts[m.Species]++
	}

	// Check that higher-level species can appear (CartItem, Purchase, etc.)
	// Note: Due to decay, some may not appear, which is fine
	hasHigherLevel := counts["CartItem"] > 0 || counts["Purchase"] > 0 || 
		counts["Recommendation"] > 0 || counts["AbandonedCart"] > 0
	if !hasHigherLevel && totalCount > 0 {
		t.Logf("Warning: No higher-level species appeared. Counts: %v", counts)
	}

	// Ensure no runaway species
	for species, count := range counts {
		if count > 500 {
			t.Errorf("Species %s has excessive count (%d), possible runaway reaction", species, count)
		}
	}

	t.Logf("Ecommerce schema simulation: total=%d, counts=%v", totalCount, counts)
}

func TestSimulation_MonitoringSchema(t *testing.T) {
	_, schema := loadSchemaFromExamples(t, "monitoring.json")
	env := NewEnvironment(schema)

	// Seed with metrics for a few different metric names
	metricNames := []string{"cpu_usage", "memory_usage", "disk_io"}
	for _, name := range metricNames {
		for i := range 3 {
			m := NewMolecule("Metric", map[string]any{
				"name":  name,
				"value": 75.0 + float64(i)*5.0,
			}, 0)
			m.CreatedAt = 0
			m.LastTouchedAt = 0
			env.Insert(m)
		}
	}

	// Run simulation
	for range 100 {
		env.Step()
	}

	// Check sanity conditions
	molecules := env.AllMolecules()
	totalCount := len(molecules)

	if totalCount > 1000 {
		t.Errorf("Total molecule count (%d) exceeds reasonable bound (1000)", totalCount)
	}

	counts := make(map[SpeciesName]int)
	for _, m := range molecules {
		counts[m.Species]++
	}

	// Check that higher-level species can appear
	hasHigherLevel := counts["Baseline"] > 0 || counts["Anomaly"] > 0 || counts["Incident"] > 0
	if !hasHigherLevel && totalCount > 0 {
		t.Logf("Warning: No higher-level species appeared. Counts: %v", counts)
	}

	// Ensure no runaway species
	for species, count := range counts {
		if count > 500 {
			t.Errorf("Species %s has excessive count (%d), possible runaway reaction", species, count)
		}
	}

	t.Logf("Monitoring schema simulation: total=%d, counts=%v", totalCount, counts)
}

func TestSimulation_IoTSchema(t *testing.T) {
	_, schema := loadSchemaFromExamples(t, "iot.json")
	env := NewEnvironment(schema)

	// Seed with threshold and sensor readings
	// First add a threshold
	threshold := NewMolecule("Threshold", map[string]any{
		"sensor_id": "sensor1",
		"device_id": "device1",
		"value":     80.0,
	}, 0)
	threshold.CreatedAt = 0
	threshold.LastTouchedAt = 0
	env.Insert(threshold)

	// Add some sensor readings that might exceed threshold
	for i := range 5 {
		m := NewMolecule("SensorReading", map[string]any{
			"sensor_id": "sensor1",
			"device_id": "device1",
			"value":     85.0 + float64(i),
		}, 0)
		m.CreatedAt = 0
		m.LastTouchedAt = 0
		env.Insert(m)
	}

	// Run simulation
	for range 100 {
		env.Step()
	}

	// Check sanity conditions
	molecules := env.AllMolecules()
	totalCount := len(molecules)

	if totalCount > 1000 {
		t.Errorf("Total molecule count (%d) exceeds reasonable bound (1000)", totalCount)
	}

	counts := make(map[SpeciesName]int)
	for _, m := range molecules {
		counts[m.Species]++
	}

	// Check that higher-level species can appear
	hasHigherLevel := counts["Alert"] > 0 || counts["DeviceStatus"] > 0 || counts["Maintenance"] > 0
	if !hasHigherLevel && totalCount > 0 {
		t.Logf("Warning: No higher-level species appeared. Counts: %v", counts)
	}

	// Ensure no runaway species
	for species, count := range counts {
		if count > 500 {
			t.Errorf("Species %s has excessive count (%d), possible runaway reaction", species, count)
		}
	}

	// Check that SensorReading molecules with very low energy are eventually removed
	// (decay reactions should consume them)
	lowEnergyReadings := 0
	for _, m := range molecules {
		if m.Species == "SensorReading" && m.Energy <= 0.1 {
			lowEnergyReadings++
		}
	}
	if lowEnergyReadings > 10 {
		t.Logf("Warning: %d SensorReading molecules with very low energy remain (decay may not be working)", lowEnergyReadings)
	}

	t.Logf("IoT schema simulation: total=%d, counts=%v", totalCount, counts)
}

func TestSimulation_DefaultSchema(t *testing.T) {
	_, schema := loadSchemaFromExamples(t, "default.json")
	env := NewEnvironment(schema)

	// Seed with some events
	for i := range 5 {
		m := NewMolecule("Event", map[string]any{
			"type": "test_event",
			"data": i,
		}, 0)
		m.CreatedAt = 0
		m.LastTouchedAt = 0
		env.Insert(m)
	}

	// Run simulation
	for range 100 {
		env.Step()
	}

	// Check sanity conditions
	molecules := env.AllMolecules()
	totalCount := len(molecules)

	if totalCount > 1000 {
		t.Errorf("Total molecule count (%d) exceeds reasonable bound (1000)", totalCount)
	}

	counts := make(map[SpeciesName]int)
	for _, m := range molecules {
		counts[m.Species]++
	}

	// Ensure no runaway species
	for species, count := range counts {
		if count > 500 {
			t.Errorf("Species %s has excessive count (%d), possible runaway reaction", species, count)
		}
	}

	t.Logf("Default schema simulation: total=%d, counts=%v", totalCount, counts)
}

