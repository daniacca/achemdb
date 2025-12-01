package achem

import (
	"testing"
)

// mockReaction is a simple reaction implementation for testing
type mockReaction struct {
	id           string
	name         string
	inputPattern func(Molecule) bool
	rate         float64
	apply        func(Molecule, EnvView, ReactionContext) ReactionEffect
	effectiveRate func(Molecule, EnvView) float64
}

func (m *mockReaction) ID() string   { return m.id }
func (m *mockReaction) Name() string { return m.name }
func (m *mockReaction) Rate() float64 { return m.rate }
func (m *mockReaction) EffectiveRate(mol Molecule, env EnvView) float64 {
	if m.effectiveRate != nil {
		return m.effectiveRate(mol, env)
	}
	return m.rate
}
func (m *mockReaction) InputPattern(mol Molecule) bool {
	if m.inputPattern != nil {
		return m.inputPattern(mol)
	}
	return false
}
func (m *mockReaction) Apply(mol Molecule, env EnvView, ctx ReactionContext) ReactionEffect {
	if m.apply != nil {
		return m.apply(mol, env, ctx)
	}
	return ReactionEffect{}
}

func TestNewSchema(t *testing.T) {
	schema := NewSchema("test-schema")
	if schema == nil {
		t.Fatal("NewSchema returned nil")
	}
	if schema.Name != "test-schema" {
		t.Errorf("Expected name 'test-schema', got '%s'", schema.Name)
	}
	if schema.species == nil {
		t.Error("Expected non-nil species map")
	}
	if len(schema.species) != 0 {
		t.Errorf("Expected empty species map, got %d species", len(schema.species))
	}
	if schema.reactions == nil {
		t.Error("Expected non-nil reactions slice")
	}
	if len(schema.reactions) != 0 {
		t.Errorf("Expected empty reactions slice, got %d reactions", len(schema.reactions))
	}
}

func TestSchema_WithSpecies(t *testing.T) {
	schema := NewSchema("test").WithSpecies(
		Species{
			Name:        "Species1",
			Description: "First species",
		},
		Species{
			Name:        "Species2",
			Description: "Second species",
		},
	)

	if len(schema.species) != 2 {
		t.Errorf("Expected 2 species, got %d", len(schema.species))
	}

	sp1, ok := schema.Species("Species1")
	if !ok {
		t.Error("Expected to find Species1")
	}
	if sp1.Name != "Species1" || sp1.Description != "First species" {
		t.Errorf("Species1 data incorrect: %+v", sp1)
	}

	sp2, ok := schema.Species("Species2")
	if !ok {
		t.Error("Expected to find Species2")
	}
	if sp2.Name != "Species2" || sp2.Description != "Second species" {
		t.Errorf("Species2 data incorrect: %+v", sp2)
	}
}

func TestSchema_WithSpecies_Chaining(t *testing.T) {
	schema := NewSchema("test").
		WithSpecies(Species{Name: "A", Description: "A desc"}).
		WithSpecies(Species{Name: "B", Description: "B desc"})

	if len(schema.species) != 2 {
		t.Errorf("Expected 2 species after chaining, got %d", len(schema.species))
	}
}

func TestSchema_WithSpecies_Overwrite(t *testing.T) {
	schema := NewSchema("test").
		WithSpecies(Species{Name: "A", Description: "First"}).
		WithSpecies(Species{Name: "A", Description: "Second"})

	sp, ok := schema.Species("A")
	if !ok {
		t.Fatal("Expected to find species A")
	}
	if sp.Description != "Second" {
		t.Errorf("Expected description 'Second' (overwritten), got '%s'", sp.Description)
	}
}

func TestSchema_WithReactions(t *testing.T) {
	r1 := &mockReaction{id: "r1", name: "Reaction 1", rate: 0.5}
	r2 := &mockReaction{id: "r2", name: "Reaction 2", rate: 0.8}

	schema := NewSchema("test").WithReactions(r1, r2)

	reactions := schema.Reactions()
	if len(reactions) != 2 {
		t.Errorf("Expected 2 reactions, got %d", len(reactions))
	}

	if reactions[0].ID() != "r1" {
		t.Errorf("Expected first reaction ID 'r1', got '%s'", reactions[0].ID())
	}
	if reactions[1].ID() != "r2" {
		t.Errorf("Expected second reaction ID 'r2', got '%s'", reactions[1].ID())
	}
}

func TestSchema_WithReactions_Chaining(t *testing.T) {
	r1 := &mockReaction{id: "r1", rate: 0.5}
	r2 := &mockReaction{id: "r2", rate: 0.8}

	schema := NewSchema("test").
		WithReactions(r1).
		WithReactions(r2)

	reactions := schema.Reactions()
	if len(reactions) != 2 {
		t.Errorf("Expected 2 reactions after chaining, got %d", len(reactions))
	}
}

func TestSchema_Species(t *testing.T) {
	schema := NewSchema("test").WithSpecies(
		Species{Name: "Found", Description: "Found species"},
	)

	// Test found
	sp, ok := schema.Species("Found")
	if !ok {
		t.Error("Expected to find 'Found' species")
	}
	if sp.Name != "Found" {
		t.Errorf("Expected name 'Found', got '%s'", sp.Name)
	}

	// Test not found
	_, ok = schema.Species("NotFound")
	if ok {
		t.Error("Expected not to find 'NotFound' species")
	}
}

func TestSchema_Reactions(t *testing.T) {
	schema := NewSchema("test")
	reactions := schema.Reactions()
	if reactions == nil {
		t.Error("Expected non-nil reactions slice")
	}
	if len(reactions) != 0 {
		t.Errorf("Expected empty reactions, got %d", len(reactions))
	}

	r1 := &mockReaction{id: "r1", rate: 0.5}
	schema = schema.WithReactions(r1)
	reactions = schema.Reactions()
	if len(reactions) != 1 {
		t.Errorf("Expected 1 reaction, got %d", len(reactions))
	}
}

