package achem

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestNewEnvironment(t *testing.T) {
	schema := NewSchema("test")
	env := NewEnvironment(schema)

	if env == nil {
		t.Fatal("NewEnvironment returned nil")
	}
	if env.schema != schema {
		t.Error("Environment schema mismatch")
	}
	if env.mols == nil {
		t.Error("Expected non-nil molecules map")
	}
	if len(env.mols) != 0 {
		t.Errorf("Expected empty molecules map, got %d", len(env.mols))
	}
	if env.time != 0 {
		t.Errorf("Expected initial time 0, got %d", env.time)
	}
	if env.rand == nil {
		t.Error("Expected non-nil random generator")
	}
}

func TestEnvironment_now(t *testing.T) {
	schema := NewSchema("test")
	env := NewEnvironment(schema)

	if env.now() != 0 {
		t.Errorf("Expected initial time 0, got %d", env.now())
	}

	env.time = 42
	if env.now() != 42 {
		t.Errorf("Expected time 42, got %d", env.now())
	}
}

func TestEnvironment_Insert(t *testing.T) {
	schema := NewSchema("test")
	env := NewEnvironment(schema)

	// Insert molecule without ID
	m := NewMolecule("TestSpecies", map[string]any{"key": "value"}, 0)
	m.ID = "" // Clear ID to test auto-generation
	env.Insert(m)

	molecules := env.AllMolecules()
	if len(molecules) != 1 {
		t.Fatalf("Expected 1 molecule, got %d", len(molecules))
	}

	inserted := molecules[0]
	if inserted.ID == "" {
		t.Error("Expected auto-generated ID")
	}
	if inserted.Species != "TestSpecies" {
		t.Errorf("Expected species 'TestSpecies', got '%s'", inserted.Species)
	}
	// CreatedAt and LastTouchedAt should be set to env.now() (which is 0 initially)
	if inserted.CreatedAt != 0 {
		t.Errorf("Expected CreatedAt to be 0 (env.now()), got %d", inserted.CreatedAt)
	}
	if inserted.LastTouchedAt != 0 {
		t.Errorf("Expected LastTouchedAt to be 0 (env.now()), got %d", inserted.LastTouchedAt)
	}
}

func TestEnvironment_Insert_WithID(t *testing.T) {
	schema := NewSchema("test")
	env := NewEnvironment(schema)

	m := Molecule{
		ID:      "custom-id",
		Species: "TestSpecies",
		Payload: map[string]any{},
	}
	env.Insert(m)

	molecules := env.AllMolecules()
	if len(molecules) != 1 {
		t.Fatalf("Expected 1 molecule, got %d", len(molecules))
	}
	if molecules[0].ID != "custom-id" {
		t.Errorf("Expected ID 'custom-id', got '%s'", molecules[0].ID)
	}
}

func TestEnvironment_Insert_Multiple(t *testing.T) {
	schema := NewSchema("test")
	env := NewEnvironment(schema)

	for i := 0; i < 10; i++ {
		m := NewMolecule("TestSpecies", map[string]any{"index": i}, 0)
		env.Insert(m)
	}

	molecules := env.AllMolecules()
	if len(molecules) != 10 {
		t.Errorf("Expected 10 molecules, got %d", len(molecules))
	}
}

func TestEnvironment_AllMolecules(t *testing.T) {
	schema := NewSchema("test")
	env := NewEnvironment(schema)

	// Empty environment
	molecules := env.AllMolecules()
	if len(molecules) != 0 {
		t.Errorf("Expected 0 molecules, got %d", len(molecules))
	}

	// Add some molecules
	env.Insert(NewMolecule("A", nil, 0))
	env.Insert(NewMolecule("B", nil, 0))
	env.Insert(NewMolecule("C", nil, 0))

	molecules = env.AllMolecules()
	if len(molecules) != 3 {
		t.Errorf("Expected 3 molecules, got %d", len(molecules))
	}
}

func TestEnvironment_Step_IncrementsTime(t *testing.T) {
	schema := NewSchema("test")
	env := NewEnvironment(schema)

	initialTime := env.now()
	env.Step()
	if env.now() != initialTime+1 {
		t.Errorf("Expected time to increment by 1, got %d (was %d)", env.now(), initialTime)
	}

	env.Step()
	if env.now() != initialTime+2 {
		t.Errorf("Expected time to increment by 2, got %d", env.now())
	}
}

func TestEnvironment_Step_WithReaction(t *testing.T) {
	schema := NewSchema("test")
	
	// Create a reaction that always matches and always fires
	consumed := false
	r := &mockReaction{
		id:   "test-reaction",
		name: "Test Reaction",
		rate: 1.0, // Always fire
		inputPattern: func(m Molecule) bool {
			return m.Species == "Input"
		},
		apply: func(m Molecule, env EnvView, ctx ReactionContext) ReactionEffect {
			consumed = true
			return ReactionEffect{
				ConsumedIDs: []MoleculeID{m.ID},
				NewMolecules: []Molecule{
					NewMolecule("Output", map[string]any{"from": "Input"}, ctx.EnvTime),
				},
			}
		},
	}

	schema = schema.WithReactions(r)
	env := NewEnvironment(schema)

	// Insert input molecule
	env.Insert(NewMolecule("Input", nil, 0))

	// Step should trigger reaction
	env.Step()

	molecules := env.AllMolecules()
	if !consumed {
		t.Error("Expected reaction to be applied")
	}

	// Should have output molecule, not input
	foundOutput := false
	for _, m := range molecules {
		if m.Species == "Output" {
			foundOutput = true
		}
		if m.Species == "Input" {
			t.Error("Expected input molecule to be consumed")
		}
	}
	if !foundOutput {
		t.Error("Expected output molecule to be created")
	}
}

func TestEnvironment_Step_UpdateMolecule(t *testing.T) {
	schema := NewSchema("test")
	
	updated := false
	r := &mockReaction{
		id:   "update-reaction",
		rate: 1.0,
		inputPattern: func(m Molecule) bool {
			return m.Species == "ToUpdate"
		},
		apply: func(m Molecule, env EnvView, ctx ReactionContext) ReactionEffect {
			updated = true
			m.Energy = 5.0
			m.LastTouchedAt = ctx.EnvTime
			return ReactionEffect{
				ConsumedIDs: []MoleculeID{},
				Changes: []MoleculeChange{
					{ID: m.ID, Updated: &m},
				},
			}
		},
	}

	schema = schema.WithReactions(r)
	env := NewEnvironment(schema)

	m := NewMolecule("ToUpdate", nil, 0)
	originalID := m.ID
	env.Insert(m)

	env.Step()

	molecules := env.AllMolecules()
	if len(molecules) != 1 {
		t.Fatalf("Expected 1 molecule, got %d", len(molecules))
	}

	if !updated {
		t.Error("Expected reaction to update molecule")
	}

	if molecules[0].ID != originalID {
		t.Error("Expected molecule ID to remain the same")
	}
	if molecules[0].Energy != 5.0 {
		t.Errorf("Expected energy 5.0, got %f", molecules[0].Energy)
	}
}

func TestEnvironment_Step_ReactionRate(t *testing.T) {
	schema := NewSchema("test")
	
	// Reaction with rate 0.0 should never fire
	fired := false
	r := &mockReaction{
		id:   "never-fire",
		rate: 0.0,
		inputPattern: func(m Molecule) bool {
			return true // Match everything
		},
		apply: func(m Molecule, env EnvView, ctx ReactionContext) ReactionEffect {
			fired = true
			return ReactionEffect{}
		},
	}

	schema = schema.WithReactions(r)
	env := NewEnvironment(schema)
	env.Insert(NewMolecule("Test", nil, 0))

	// Run many steps
	for i := 0; i < 100; i++ {
		env.Step()
	}

	if fired {
		t.Error("Expected reaction with rate 0.0 to never fire")
	}
}

func TestEnvironment_Step_MultipleReactions(t *testing.T) {
	schema := NewSchema("test")
	
	// Create two reactions that both match the same molecule
	r1Fired := false
	r2Fired := false
	
	r1 := &mockReaction{
		id:   "r1",
		rate: 1.0, // Always fire
		inputPattern: func(m Molecule) bool {
			return m.Species == "Input"
		},
		apply: func(m Molecule, env EnvView, ctx ReactionContext) ReactionEffect {
			r1Fired = true
			return ReactionEffect{
				ConsumedIDs: []MoleculeID{}, // Don't consume, let r2 also process
			}
		},
	}
	
	r2 := &mockReaction{
		id:   "r2",
		rate: 1.0, // Always fire
		inputPattern: func(m Molecule) bool {
			return m.Species == "Input"
		},
		apply: func(m Molecule, env EnvView, ctx ReactionContext) ReactionEffect {
			r2Fired = true
			return ReactionEffect{
				ConsumedIDs: []MoleculeID{m.ID}, // Consume after both reactions
			}
		},
	}

	schema = schema.WithReactions(r1, r2)
	env := NewEnvironment(schema)
	env.Insert(NewMolecule("Input", nil, 0))

	env.Step()

	if !r1Fired {
		t.Error("Expected r1 to fire")
	}
	if !r2Fired {
		t.Error("Expected r2 to fire")
	}
}

func TestEnvironment_Step_NewMoleculeAutoID(t *testing.T) {
	schema := NewSchema("test")
	
	r := &mockReaction{
		id:   "create-new",
		rate: 1.0,
		inputPattern: func(m Molecule) bool {
			return m.Species == "Input"
		},
		apply: func(m Molecule, env EnvView, ctx ReactionContext) ReactionEffect {
			// Create new molecule without ID
			newMol := Molecule{
				Species: "Output",
				Payload: map[string]any{},
			}
			return ReactionEffect{
				ConsumedIDs:  []MoleculeID{m.ID},
				NewMolecules: []Molecule{newMol},
			}
		},
	}

	schema = schema.WithReactions(r)
	env := NewEnvironment(schema)
	env.Insert(NewMolecule("Input", nil, 0))

	env.Step()

	molecules := env.AllMolecules()
	if len(molecules) != 1 {
		t.Fatalf("Expected 1 molecule (output), got %d", len(molecules))
	}
	if molecules[0].Species != "Output" {
		t.Errorf("Expected Output molecule, got %s", molecules[0].Species)
	}
	if molecules[0].ID == "" {
		t.Error("Expected auto-generated ID for new molecule")
	}
	if molecules[0].CreatedAt == 0 {
		t.Error("Expected CreatedAt to be set for new molecule")
	}
}

func TestEnvView_MoleculesBySpecies(t *testing.T) {
	molecules := []Molecule{
		NewMolecule("A", nil, 0),
		NewMolecule("B", nil, 0),
		NewMolecule("A", nil, 0),
		NewMolecule("C", nil, 0),
	}

	// build per-species index
	bySpecies := make(map[SpeciesName][]Molecule)
	for _, m := range molecules {
		bySpecies[m.Species] = append(bySpecies[m.Species], m)
	}

	view := envView{
		molecules: molecules,
		bySpecies: bySpecies,
	}

	as := view.MoleculesBySpecies("A")
	if len(as) != 2 {
		t.Errorf("Expected 2 molecules of species A, got %d", len(as))
	}

	bs := view.MoleculesBySpecies("B")
	if len(bs) != 1 {
		t.Errorf("Expected 1 molecule of species B, got %d", len(bs))
	}

	ds := view.MoleculesBySpecies("D")
	if len(ds) != 0 {
		t.Errorf("Expected 0 molecules of species D, got %d", len(ds))
	}
}

func TestEnvView_Find(t *testing.T) {
	view := envView{
		molecules: []Molecule{
			NewMolecule("A", map[string]any{"value": 1}, 0),
			NewMolecule("B", map[string]any{"value": 2}, 0),
			NewMolecule("A", map[string]any{"value": 3}, 0),
		},
	}

	// Find by species
	result := view.Find(func(m Molecule) bool {
		return m.Species == "A"
	})
	if len(result) != 2 {
		t.Errorf("Expected 2 molecules with species A, got %d", len(result))
	}

	// Find by payload
	result = view.Find(func(m Molecule) bool {
		if val, ok := m.Payload["value"].(int); ok {
			return val > 1
		}
		return false
	})
	if len(result) != 2 {
		t.Errorf("Expected 2 molecules with value > 1, got %d", len(result))
	}

	// Find none
	result = view.Find(func(m Molecule) bool {
		return false
	})
	if len(result) != 0 {
		t.Errorf("Expected 0 molecules, got %d", len(result))
	}
}

func TestEnvironment_Run_StartsTicker(t *testing.T) {
	schema := NewSchema("test")
	env := NewEnvironment(schema)

	// Insert a molecule to track changes
	env.Insert(NewMolecule("Test", nil, 0))
	initialTime := env.now()

	// Start running with a short interval
	env.Run(50 * time.Millisecond)

	// Wait a bit to let it tick a few times
	time.Sleep(200 * time.Millisecond)

	// Stop it
	env.Stop()

	// Give it a moment to stop
	time.Sleep(100 * time.Millisecond)

	// Time should have advanced
	finalTime := env.now()
	if finalTime <= initialTime {
		t.Errorf("Expected time to advance, initial: %d, final: %d", initialTime, finalTime)
	}
}

func TestEnvironment_Run_CanBeStopped(t *testing.T) {
	schema := NewSchema("test")
	env := NewEnvironment(schema)

	initialTime := env.now()

	// Start running
	env.Run(10 * time.Millisecond)

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Stop it
	env.Stop()

	// Wait for it to actually stop
	time.Sleep(100 * time.Millisecond)

	// Record time after stop
	timeAfterStop := env.now()

	// Wait a bit more - time should not advance after stop
	time.Sleep(100 * time.Millisecond)
	timeAfterWait := env.now()

	if timeAfterWait != timeAfterStop {
		t.Errorf("Expected time to stop advancing after Stop(), was %d, now %d", timeAfterStop, timeAfterWait)
	}

	// But it should have advanced while running
	if timeAfterStop <= initialTime {
		t.Errorf("Expected time to advance while running, initial: %d, after stop: %d", initialTime, timeAfterStop)
	}
}

func TestEnvironment_Run_CanBeRestarted(t *testing.T) {
	schema := NewSchema("test")
	env := NewEnvironment(schema)

	initialTime := env.now()

	// Start running
	env.Run(50 * time.Millisecond)
	time.Sleep(100 * time.Millisecond)

	// Stop it
	env.Stop()
	time.Sleep(50 * time.Millisecond)
	timeAfterFirstStop := env.now()

	// Restart it
	env.Run(50 * time.Millisecond)
	time.Sleep(100 * time.Millisecond)

	// Stop again
	env.Stop()
	time.Sleep(50 * time.Millisecond)
	timeAfterSecondStop := env.now()

	// Time should have advanced in both runs
	if timeAfterFirstStop <= initialTime {
		t.Errorf("Expected time to advance in first run, initial: %d, after first stop: %d", initialTime, timeAfterFirstStop)
	}
	if timeAfterSecondStop <= timeAfterFirstStop {
		t.Errorf("Expected time to advance in second run, after first stop: %d, after second stop: %d", timeAfterFirstStop, timeAfterSecondStop)
	}
}

func TestEnvironment_Run_DoesNotBlock(t *testing.T) {
	schema := NewSchema("test")
	env := NewEnvironment(schema)

	// Run should return immediately (not block)
	start := time.Now()
	env.Run(100 * time.Millisecond)
	duration := time.Since(start)

	// Should return almost immediately (within 10ms)
	if duration > 10*time.Millisecond {
		t.Errorf("Run() should not block, took %v", duration)
	}

	// Clean up
	env.Stop()
	time.Sleep(50 * time.Millisecond)
}

func TestEnvironment_Run_MultipleCallsIgnored(t *testing.T) {
	schema := NewSchema("test")
	env := NewEnvironment(schema)

	initialTime := env.now()

	// Call Run multiple times
	env.Run(50 * time.Millisecond)
	env.Run(50 * time.Millisecond)
	env.Run(50 * time.Millisecond)

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Stop once should be enough
	env.Stop()
	time.Sleep(50 * time.Millisecond)

	// Time should have advanced (only one ticker should be running)
	finalTime := env.now()
	if finalTime <= initialTime {
		t.Errorf("Expected time to advance, initial: %d, final: %d", initialTime, finalTime)
	}
}

func TestEnvironment_Stop_WhenNotRunning(t *testing.T) {
	schema := NewSchema("test")
	env := NewEnvironment(schema)

	// Stop when not running should not panic
	env.Stop()
	env.Stop() // Multiple stops should be safe
}

func TestEnvironment_Run_ConcurrentAccess(t *testing.T) {
	schema := NewSchema("test")
	env := NewEnvironment(schema)

	// Insert some molecules
	for i := range 10 {
		env.Insert(NewMolecule("Test", map[string]any{"id": i}, 0))
	}

	var wg sync.WaitGroup

	// Start running
	env.Run(10 * time.Millisecond)

	// Concurrently access molecules while running
	for range 5 {
		wg.Go(func() {
			for range 10 {
				_ = env.AllMolecules()
				time.Sleep(1 * time.Millisecond)
			}
		})
	}

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Stop
	env.Stop()

	// Wait for goroutines
	wg.Wait()

	// Should not have panicked or deadlocked
	molecules := env.AllMolecules()
	_ = molecules // Verify we can access molecules without error
}

func TestEnvironment_Run_StepIsCalled(t *testing.T) {
	schema := NewSchema("test")
	
	// Create a reaction that tracks if it was called
	reactionCalled := false
	var mu sync.Mutex
	
	r := &mockReaction{
		id:   "track-reaction",
		rate: 1.0, // Always fire
		inputPattern: func(m Molecule) bool {
			return m.Species == "Input"
		},
		apply: func(m Molecule, env EnvView, ctx ReactionContext) ReactionEffect {
			mu.Lock()
			reactionCalled = true
			mu.Unlock()
			return ReactionEffect{
				ConsumedIDs: []MoleculeID{m.ID},
			}
		},
	}

	schema = schema.WithReactions(r)
	env := NewEnvironment(schema)

	// Insert input molecule
	env.Insert(NewMolecule("Input", nil, 0))

	// Start running
	env.Run(50 * time.Millisecond)

	// Wait for reaction to be called
	time.Sleep(200 * time.Millisecond)

	// Stop
	env.Stop()
	time.Sleep(50 * time.Millisecond)

	// Check if reaction was called
	mu.Lock()
	called := reactionCalled
	mu.Unlock()

	if !called {
		t.Error("Expected reaction to be called during Run()")
	}
}

func TestEnvironment_SnapshotPath(t *testing.T) {
	schema := NewSchema("test")
	env := NewEnvironment(schema)
	env.SetEnvironmentID("test-env")

	tmpDir := t.TempDir()
	env.SetSnapshotDir(tmpDir)

	expectedPath := filepath.Join(tmpDir, "test-env.snapshot.json")
	actualPath := env.SnapshotPath()

	if actualPath != expectedPath {
		t.Errorf("Expected snapshot path %s, got %s", expectedPath, actualPath)
	}
}

func TestEnvironment_CreateSnapshot(t *testing.T) {
	schema := NewSchema("test")
	env := NewEnvironment(schema)
	env.SetEnvironmentID("test-env")

	// Insert some molecules
	m1 := NewMolecule("Species1", map[string]any{"key": "value1"}, 0)
	m2 := NewMolecule("Species2", map[string]any{"key": "value2"}, 0)
	env.Insert(m1)
	env.Insert(m2)

	// Manually set time for testing
	env.mu.Lock()
	env.time = 42
	env.mu.Unlock()

	// Create snapshot
	snapshot, err := env.createSnapshot()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify snapshot content
	if snapshot.EnvironmentID != "test-env" {
		t.Errorf("Expected EnvironmentID 'test-env', got '%s'", snapshot.EnvironmentID)
	}
	if snapshot.Time != 42 {
		t.Errorf("Expected Time 42, got %d", snapshot.Time)
	}
	if len(snapshot.Molecules) != 2 {
		t.Fatalf("Expected 2 molecules, got %d", len(snapshot.Molecules))
	}

	// Verify molecules are present
	moleculeIDs := make(map[MoleculeID]bool)
	for _, m := range snapshot.Molecules {
		moleculeIDs[m.ID] = true
	}
	if !moleculeIDs[m1.ID] || !moleculeIDs[m2.ID] {
		t.Error("Expected both molecules to be in snapshot")
	}
}

func TestEnvironment_SaveSnapshot_CreatesFile(t *testing.T) {
	schema := NewSchema("test")
	env := NewEnvironment(schema)
	env.SetEnvironmentID("test-env")

	tmpDir := t.TempDir()
	env.SetSnapshotDir(tmpDir)

	// Insert a molecule
	m := NewMolecule("TestSpecies", map[string]any{"test": "data"}, 0)
	env.Insert(m)

	// Manually set time
	env.mu.Lock()
	env.time = 100
	env.mu.Unlock()

	// Save snapshot
	err := env.SaveSnapshot()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify file exists
	expectedPath := filepath.Join(tmpDir, "test-env.snapshot.json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatalf("Expected snapshot file to exist at %s", expectedPath)
	}

	// Verify temp file doesn't exist
	tempPath := expectedPath + ".tmp"
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Errorf("Expected temp file to be removed, but it exists at %s", tempPath)
	}
}

func TestEnvironment_SaveSnapshot_MatchesExpectedContent(t *testing.T) {
	schema := NewSchema("test")
	env := NewEnvironment(schema)
	env.SetEnvironmentID("test-env")

	tmpDir := t.TempDir()
	env.SetSnapshotDir(tmpDir)

	// Insert molecules
	m1 := NewMolecule("Species1", map[string]any{"key1": "value1"}, 0)
	m2 := NewMolecule("Species2", map[string]any{"key2": 42}, 0)
	env.Insert(m1)
	env.Insert(m2)

	// Manually set time
	env.mu.Lock()
	env.time = 123
	env.mu.Unlock()

	// Save snapshot
	err := env.SaveSnapshot()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Read and decode snapshot
	path := filepath.Join(tmpDir, "test-env.snapshot.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read snapshot file: %v", err)
	}

	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("Failed to decode snapshot JSON: %v", err)
	}

	// Verify snapshot content
	if snapshot.EnvironmentID != "test-env" {
		t.Errorf("Expected EnvironmentID 'test-env', got '%s'", snapshot.EnvironmentID)
	}
	if snapshot.Time != 123 {
		t.Errorf("Expected Time 123, got %d", snapshot.Time)
	}
	if len(snapshot.Molecules) != 2 {
		t.Fatalf("Expected 2 molecules, got %d", len(snapshot.Molecules))
	}

	// Verify molecules match
	moleculeMap := make(map[MoleculeID]Molecule)
	for _, m := range snapshot.Molecules {
		moleculeMap[m.ID] = m
	}

	if mol1, ok := moleculeMap[m1.ID]; !ok {
		t.Error("Expected m1 to be in snapshot")
	} else {
		if mol1.Species != m1.Species {
			t.Errorf("Expected m1 species '%s', got '%s'", m1.Species, mol1.Species)
		}
		if mol1.Payload["key1"] != "value1" {
			t.Errorf("Expected m1 payload key1='value1', got '%v'", mol1.Payload["key1"])
		}
	}

	if mol2, ok := moleculeMap[m2.ID]; !ok {
		t.Error("Expected m2 to be in snapshot")
	} else {
		if mol2.Species != m2.Species {
			t.Errorf("Expected m2 species '%s', got '%s'", m2.Species, mol2.Species)
		}
		// JSON unmarshals numbers as float64, so compare accordingly
		if val, ok := mol2.Payload["key2"].(float64); !ok || val != 42 {
			t.Errorf("Expected m2 payload key2=42, got '%v'", mol2.Payload["key2"])
		}
	}
}

func TestEnvironment_SaveSnapshot_AtomicWrite(t *testing.T) {
	schema := NewSchema("test")
	env := NewEnvironment(schema)
	env.SetEnvironmentID("test-env")

	tmpDir := t.TempDir()
	env.SetSnapshotDir(tmpDir)

	// Insert a molecule
	m := NewMolecule("TestSpecies", map[string]any{"test": "data"}, 0)
	env.Insert(m)

	// Manually set time
	env.mu.Lock()
	env.time = 200
	env.mu.Unlock()

	// Save snapshot
	err := env.SaveSnapshot()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify temp file is removed (atomic write succeeded)
	expectedPath := filepath.Join(tmpDir, "test-env.snapshot.json")
	tempPath := expectedPath + ".tmp"
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Errorf("Expected temp file to be removed after atomic write, but it exists at %s", tempPath)
	}

	// Verify final file exists and is valid
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatalf("Expected final snapshot file to exist at %s", expectedPath)
	}

	// Verify file is valid JSON
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read snapshot file: %v", err)
	}

	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("Snapshot file is not valid JSON: %v", err)
	}

	if snapshot.Time != 200 {
		t.Errorf("Expected snapshot time 200, got %d", snapshot.Time)
	}
}

func TestEnvironment_Step_TriggersSnapshot(t *testing.T) {
	schema := NewSchema("test")
	env := NewEnvironment(schema)
	env.SetEnvironmentID("test-env")

	tmpDir := t.TempDir()
	env.SetSnapshotDir(tmpDir)
	env.SetSnapshotEveryNTicks(5) // Snapshot every 5 ticks

	// Insert a molecule
	m := NewMolecule("TestSpecies", map[string]any{"test": "data"}, 0)
	env.Insert(m)

	// Set time to 4, so next step will be 5 (which triggers snapshot)
	env.mu.Lock()
	env.time = 4
	env.mu.Unlock()

	// Step to time 5 (should trigger snapshot)
	env.Step()

	// Wait a bit for async snapshot to complete
	time.Sleep(200 * time.Millisecond)

	// Verify snapshot file was created
	expectedPath := filepath.Join(tmpDir, "test-env.snapshot.json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatalf("Expected snapshot file to be created at time 5, but it doesn't exist")
	}

	// Verify snapshot content
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read snapshot file: %v", err)
	}

	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("Failed to decode snapshot: %v", err)
	}

	if snapshot.Time != 5 {
		t.Errorf("Expected snapshot time 5, got %d", snapshot.Time)
	}
}

func TestEnvironment_SaveSnapshot_NoDirectory(t *testing.T) {
	schema := NewSchema("test")
	env := NewEnvironment(schema)
	env.SetEnvironmentID("test-env")

	// Set snapshot dir to empty (disabled)
	env.SetSnapshotDir("")

	// Insert a molecule
	m := NewMolecule("TestSpecies", nil, 0)
	env.Insert(m)

	// Save snapshot should succeed but do nothing
	err := env.SaveSnapshot()
	if err != nil {
		t.Errorf("Expected no error when snapshot dir is empty, got %v", err)
	}
}

func TestEnvironment_LoadSnapshot_EmptyEnvironment(t *testing.T) {
	// Create schema with TestSpecies
	schema := NewSchema("test")
	schema = schema.WithSpecies(Species{Name: "TestSpecies"})

	env := NewEnvironment(schema)
	env.SetEnvironmentID("test-env")

	tmpDir := t.TempDir()
	env.SetSnapshotDir(tmpDir)

	// Create a snapshot with some molecules
	m1 := NewMolecule("TestSpecies", map[string]any{"key": "value"}, 0)
	m2 := NewMolecule("TestSpecies", map[string]any{"key2": 42}, 0)
	env.Insert(m1)
	env.Insert(m2)

	// Set time
	env.mu.Lock()
	env.time = 100
	env.mu.Unlock()

	// Save snapshot
	if err := env.SaveSnapshot(); err != nil {
		t.Fatalf("Failed to save snapshot: %v", err)
	}

	// Create a new empty environment
	newEnv := NewEnvironment(schema)
	newEnv.SetEnvironmentID("test-env")
	newEnv.SetSnapshotDir(tmpDir)

	// Load snapshot
	if err := newEnv.LoadSnapshot(); err != nil {
		t.Fatalf("Failed to load snapshot: %v", err)
	}

	// Verify environment was restored
	if newEnv.time != 100 {
		t.Errorf("Expected time 100, got %d", newEnv.time)
	}

	molecules := newEnv.AllMolecules()
	if len(molecules) != 2 {
		t.Fatalf("Expected 2 molecules, got %d", len(molecules))
	}

	// Verify molecules match
	moleculeMap := make(map[MoleculeID]Molecule)
	for _, m := range molecules {
		moleculeMap[m.ID] = m
	}

	if _, ok := moleculeMap[m1.ID]; !ok {
		t.Error("Expected m1 to be restored")
	}
	if _, ok := moleculeMap[m2.ID]; !ok {
		t.Error("Expected m2 to be restored")
	}
}

func TestEnvironment_LoadSnapshot_WithMolecules(t *testing.T) {
	schema := NewSchema("test")
	schema = schema.WithSpecies(Species{Name: "Species1"}, Species{Name: "Species2"})

	env := NewEnvironment(schema)
	env.SetEnvironmentID("test-env")

	tmpDir := t.TempDir()
	env.SetSnapshotDir(tmpDir)

	// Create molecules of different species
	m1 := NewMolecule("Species1", map[string]any{"field1": "value1"}, 0)
	m2 := NewMolecule("Species2", map[string]any{"field2": 123}, 0)
	m3 := NewMolecule("Species1", map[string]any{"field3": true}, 0)

	env.Insert(m1)
	env.Insert(m2)
	env.Insert(m3)

	// Set time
	env.mu.Lock()
	env.time = 250
	env.mu.Unlock()

	// Save snapshot
	if err := env.SaveSnapshot(); err != nil {
		t.Fatalf("Failed to save snapshot: %v", err)
	}

	// Create new environment and load snapshot
	newEnv := NewEnvironment(schema)
	newEnv.SetEnvironmentID("test-env")
	newEnv.SetSnapshotDir(tmpDir)

	if err := newEnv.LoadSnapshot(); err != nil {
		t.Fatalf("Failed to load snapshot: %v", err)
	}

	// Verify all state was restored
	if newEnv.time != 250 {
		t.Errorf("Expected time 250, got %d", newEnv.time)
	}

	molecules := newEnv.AllMolecules()
	if len(molecules) != 3 {
		t.Fatalf("Expected 3 molecules, got %d", len(molecules))
	}

	// Verify molecule details
	moleculeMap := make(map[MoleculeID]Molecule)
	for _, m := range molecules {
		moleculeMap[m.ID] = m
	}

	// Check m1
	if mol1, ok := moleculeMap[m1.ID]; !ok {
		t.Error("Expected m1 to be restored")
	} else {
		if mol1.Species != "Species1" {
			t.Errorf("Expected m1 species 'Species1', got '%s'", mol1.Species)
		}
		if mol1.Payload["field1"] != "value1" {
			t.Errorf("Expected m1 field1='value1', got '%v'", mol1.Payload["field1"])
		}
	}

	// Check m2
	if mol2, ok := moleculeMap[m2.ID]; !ok {
		t.Error("Expected m2 to be restored")
	} else {
		if mol2.Species != "Species2" {
			t.Errorf("Expected m2 species 'Species2', got '%s'", mol2.Species)
		}
		// JSON unmarshals numbers as float64
		if val, ok := mol2.Payload["field2"].(float64); !ok || val != 123 {
			t.Errorf("Expected m2 field2=123, got '%v'", mol2.Payload["field2"])
		}
	}

	// Check m3
	if mol3, ok := moleculeMap[m3.ID]; !ok {
		t.Error("Expected m3 to be restored")
	} else {
		if mol3.Species != "Species1" {
			t.Errorf("Expected m3 species 'Species1', got '%s'", mol3.Species)
		}
		if mol3.Payload["field3"] != true {
			t.Errorf("Expected m3 field3=true, got '%v'", mol3.Payload["field3"])
		}
	}
}

func TestEnvironment_LoadSnapshot_EnvIDMismatch(t *testing.T) {
	schema := NewSchema("test")
	schema = schema.WithSpecies(Species{Name: "TestSpecies"})

	tmpDir := t.TempDir()

	// Create a snapshot file manually with wrong envID
	// The file will be at the path for "different-env" but contain "test-env" as the envID
	m := NewMolecule("TestSpecies", nil, 0)
	snapshot := Snapshot{
		EnvironmentID: "test-env", // Wrong envID in snapshot
		Time:          50,
		Molecules:     []Molecule{m},
	}

	data, err := EncodeSnapshotJSON(snapshot)
	if err != nil {
		t.Fatalf("Failed to encode snapshot: %v", err)
	}

	// Write to path that "different-env" would use
	path := filepath.Join(tmpDir, "different-env.snapshot.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("Failed to write snapshot: %v", err)
	}

	// Try to load with different env ID
	newEnv := NewEnvironment(schema)
	newEnv.SetEnvironmentID("different-env")
	newEnv.SetSnapshotDir(tmpDir)

	err = newEnv.LoadSnapshot()
	if err == nil {
		t.Fatal("Expected error when loading snapshot with mismatched env ID, got nil")
	}

	// Verify error mentions environment ID mismatch
	if err.Error() == "" {
		t.Error("Expected non-empty error message")
	}
}

func TestEnvironment_LoadSnapshot_UnknownSpecies(t *testing.T) {
	// Create schema with only Species1
	schema1 := NewSchema("test1")
	schema1 = schema1.WithSpecies(Species{Name: "Species1"})

	env := NewEnvironment(schema1)
	env.SetEnvironmentID("test-env")

	tmpDir := t.TempDir()
	env.SetSnapshotDir(tmpDir)

	// Create snapshot with Species1 and Species2
	m1 := NewMolecule("Species1", nil, 0)
	m2 := NewMolecule("Species2", nil, 0) // Species2 not in schema
	env.Insert(m1)
	env.Insert(m2)

	env.mu.Lock()
	env.time = 75
	env.mu.Unlock()

	// Manually create snapshot with invalid species (bypassing validation)
	snapshot := Snapshot{
		EnvironmentID: "test-env",
		Time:          75,
		Molecules:      []Molecule{m1, m2},
	}

	data, err := EncodeSnapshotJSON(snapshot)
	if err != nil {
		t.Fatalf("Failed to encode snapshot: %v", err)
	}

	path := filepath.Join(tmpDir, "test-env.snapshot.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("Failed to write snapshot: %v", err)
	}

	// Try to load with schema that doesn't have Species2
	newEnv := NewEnvironment(schema1)
	newEnv.SetEnvironmentID("test-env")
	newEnv.SetSnapshotDir(tmpDir)

	err = newEnv.LoadSnapshot()
	if err == nil {
		t.Fatal("Expected error when loading snapshot with unknown species, got nil")
	}

	// Verify error mentions species validation
	if err.Error() == "" {
		t.Error("Expected non-empty error message")
	}
}

func TestEnvironment_LoadSnapshot_NoFile(t *testing.T) {
	schema := NewSchema("test")
	env := NewEnvironment(schema)
	env.SetEnvironmentID("test-env")

	tmpDir := t.TempDir()
	env.SetSnapshotDir(tmpDir)

	// Try to load snapshot when file doesn't exist (should be no-op)
	err := env.LoadSnapshot()
	if err != nil {
		t.Errorf("Expected no error when snapshot file doesn't exist, got %v", err)
	}

	// Verify environment is still empty
	if env.time != 0 {
		t.Errorf("Expected time 0, got %d", env.time)
	}
	if len(env.AllMolecules()) != 0 {
		t.Errorf("Expected 0 molecules, got %d", len(env.AllMolecules()))
	}
}

func TestEnvironment_LoadSnapshot_NoSnapshotDir(t *testing.T) {
	schema := NewSchema("test")
	env := NewEnvironment(schema)
	env.SetEnvironmentID("test-env")

	// Don't set snapshot dir (empty)
	env.SetSnapshotDir("")

	// Try to load snapshot (should be no-op)
	err := env.LoadSnapshot()
	if err != nil {
		t.Errorf("Expected no error when snapshot dir is not set, got %v", err)
	}
}

