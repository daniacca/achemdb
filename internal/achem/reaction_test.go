package achem

import (
	"math/rand"
	"testing"
)

func TestReactionContext(t *testing.T) {
	// Test creating a ReactionContext
	ctx := ReactionContext{
		EnvTime: 100,
		Random:  rand.Float64,
	}
	
	if ctx.EnvTime != 100 {
		t.Errorf("Expected EnvTime 100, got %d", ctx.EnvTime)
	}
	
	if ctx.Random == nil {
		t.Error("Expected non-nil Random function")
	}
	
	// Test Random function
	val := ctx.Random()
	if val < 0 || val >= 1 {
		t.Errorf("Expected random value between 0 and 1, got %f", val)
	}
}

func TestMoleculeChange(t *testing.T) {
	mol := NewMolecule("Test", map[string]any{"value": 1}, 0)
	updated := NewMolecule("Test", map[string]any{"value": 2}, 0)
	
	// Test MoleculeChange with update
	change := MoleculeChange{
		ID:      mol.ID,
		Updated: &updated,
	}
	
	if change.ID != mol.ID {
		t.Errorf("Expected ID %s, got %s", mol.ID, change.ID)
	}
	
	if change.Updated == nil {
		t.Error("Expected non-nil Updated molecule")
	}
	
	if change.Updated.Payload["value"] != 2 {
		t.Error("Expected updated value to be 2")
	}
	
	// Test MoleculeChange for deletion (Updated is nil)
	deleteChange := MoleculeChange{
		ID:      mol.ID,
		Updated: nil,
	}
	
	if deleteChange.Updated != nil {
		t.Error("Expected nil Updated for deletion")
	}
}

func TestReactionEffect(t *testing.T) {
	mol1 := NewMolecule("Test1", nil, 0)
	mol2 := NewMolecule("Test2", nil, 0)
	mol3 := NewMolecule("Test3", nil, 0)
	
	updated := NewMolecule("Test1", map[string]any{"value": 1}, 0)
	
	// Test empty ReactionEffect
	effect := ReactionEffect{}
	
	// Empty slices can be nil in Go, which is valid
	// We just verify they work correctly when used
	if len(effect.ConsumedIDs) != 0 {
		t.Errorf("Expected empty ConsumedIDs, got %d", len(effect.ConsumedIDs))
	}
	
	if len(effect.Changes) != 0 {
		t.Errorf("Expected empty Changes, got %d", len(effect.Changes))
	}
	
	if len(effect.NewMolecules) != 0 {
		t.Errorf("Expected empty NewMolecules, got %d", len(effect.NewMolecules))
	}
	
	if len(effect.AdditionalOps) != 0 {
		t.Errorf("Expected empty AdditionalOps, got %d", len(effect.AdditionalOps))
	}
	
	// Test ReactionEffect with consumed molecules
	effect = ReactionEffect{
		ConsumedIDs: []MoleculeID{mol1.ID, mol2.ID},
	}
	
	if len(effect.ConsumedIDs) != 2 {
		t.Errorf("Expected 2 consumed IDs, got %d", len(effect.ConsumedIDs))
	}
	if effect.ConsumedIDs[0] != mol1.ID {
		t.Errorf("Expected first consumed ID %s, got %s", mol1.ID, effect.ConsumedIDs[0])
	}
	
	// Test ReactionEffect with changes
	effect = ReactionEffect{
		Changes: []MoleculeChange{
			{ID: mol1.ID, Updated: &updated},
		},
	}
	
	if len(effect.Changes) != 1 {
		t.Errorf("Expected 1 change, got %d", len(effect.Changes))
	}
	if effect.Changes[0].ID != mol1.ID {
		t.Errorf("Expected change ID %s, got %s", mol1.ID, effect.Changes[0].ID)
	}
	
	// Test ReactionEffect with new molecules
	effect = ReactionEffect{
		NewMolecules: []Molecule{mol1, mol2, mol3},
	}
	
	if len(effect.NewMolecules) != 3 {
		t.Errorf("Expected 3 new molecules, got %d", len(effect.NewMolecules))
	}
	
	// Test ReactionEffect with additional operations
	effect = ReactionEffect{
		AdditionalOps: []Operation{
			{},
			{},
		},
	}
	
	if len(effect.AdditionalOps) != 2 {
		t.Errorf("Expected 2 additional operations, got %d", len(effect.AdditionalOps))
	}
	
	// Test complete ReactionEffect
	effect = ReactionEffect{
		ConsumedIDs: []MoleculeID{mol1.ID},
		Changes: []MoleculeChange{
			{ID: mol2.ID, Updated: &updated},
		},
		NewMolecules: []Molecule{mol3},
		AdditionalOps: []Operation{{}},
	}
	
	if len(effect.ConsumedIDs) != 1 {
		t.Errorf("Expected 1 consumed ID, got %d", len(effect.ConsumedIDs))
	}
	if len(effect.Changes) != 1 {
		t.Errorf("Expected 1 change, got %d", len(effect.Changes))
	}
	if len(effect.NewMolecules) != 1 {
		t.Errorf("Expected 1 new molecule, got %d", len(effect.NewMolecules))
	}
	if len(effect.AdditionalOps) != 1 {
		t.Errorf("Expected 1 additional operation, got %d", len(effect.AdditionalOps))
	}
}

func TestOperation(t *testing.T) {
	// Test Operation struct (placeholder for future operations)
	op := Operation{}
	
	// Operation is currently empty, so we just test it can be created
	if op == (Operation{}) {
		// This is expected - Operation is a placeholder
	}
	
	// Test Operation in a slice
	ops := []Operation{{}, {}}
	if len(ops) != 2 {
		t.Errorf("Expected 2 operations, got %d", len(ops))
	}
}

