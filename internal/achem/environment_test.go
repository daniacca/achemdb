package achem

import (
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
	view := envView{
		molecules: []Molecule{
			NewMolecule("A", nil, 0),
			NewMolecule("B", nil, 0),
			NewMolecule("A", nil, 0),
			NewMolecule("C", nil, 0),
		},
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

