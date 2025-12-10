package achem

import (
	"testing"
)

func TestIndexKeyFromValue(t *testing.T) {
	// Test with string
	key := indexKeyFromValue("test")
	if key != "test" {
		t.Errorf("Expected 'test', got '%s'", key)
	}
	
	// Test with int
	key = indexKeyFromValue(42)
	if key != "42" {
		t.Errorf("Expected '42', got '%s'", key)
	}
	
	// Test with float
	key = indexKeyFromValue(3.14)
	if key != "3.14" {
		t.Errorf("Expected '3.14', got '%s'", key)
	}
	
	// Test with bool
	key = indexKeyFromValue(true)
	if key != "true" {
		t.Errorf("Expected 'true', got '%s'", key)
	}
	
	// Test with nil
	key = indexKeyFromValue(nil)
	if key != "<nil>" {
		t.Errorf("Expected '<nil>', got '%s'", key)
	}
}

func TestResolveValueRef(t *testing.T) {
	mol := NewMolecule("Test", map[string]any{
		"value": 42,
		"name":  "test-molecule",
	}, 0)
	mol.Energy = 10.5
	mol.Stability = 0.8
	
	// Test non-string value (should return as-is)
	result := resolveValueRef(42, mol)
	if result != 42 {
		t.Errorf("Expected 42, got %v", result)
	}
	
	// Test non-$m. string (should return as-is)
	result = resolveValueRef("normal-string", mol)
	if result != "normal-string" {
		t.Errorf("Expected 'normal-string', got %v", result)
	}
	
	// Test $m.energy
	result = resolveValueRef("$m.energy", mol)
	if result != 10.5 {
		t.Errorf("Expected 10.5, got %v", result)
	}
	
	// Test $m.stability
	result = resolveValueRef("$m.stability", mol)
	if result != 0.8 {
		t.Errorf("Expected 0.8, got %v", result)
	}
	
	// Test $m.id
	result = resolveValueRef("$m.id", mol)
	if result != string(mol.ID) {
		t.Errorf("Expected '%s', got %v", mol.ID, result)
	}
	
	// Test $m.species
	result = resolveValueRef("$m.species", mol)
	if result != string(mol.Species) {
		t.Errorf("Expected '%s', got %v", mol.Species, result)
	}
	
	// Test $m.payload field
	result = resolveValueRef("$m.value", mol)
	if result != 42 {
		t.Errorf("Expected 42, got %v", result)
	}
	
	// Test $m.non-existent field (returns original string if field not found)
	result = resolveValueRef("$m.nonexistent", mol)
	if result != "$m.nonexistent" {
		t.Errorf("Expected '$m.nonexistent' for non-existent field, got %v", result)
	}
	
	// Test short string (should return as-is)
	result = resolveValueRef("$m", mol)
	if result != "$m" {
		t.Errorf("Expected '$m', got %v", result)
	}
}

func TestResolveValueRef_TimestampFields(t *testing.T) {
	mol := NewMolecule("Test", map[string]any{"value": 42}, 100)
	mol.CreatedAt = 100
	mol.LastTouchedAt = 200
	
	// Test $m.created_at (snake_case)
	result := resolveValueRef("$m.created_at", mol)
	if result != int64(100) {
		t.Errorf("Expected 100, got %v", result)
	}
	
	// Test $m.createdAt (camelCase)
	result = resolveValueRef("$m.createdAt", mol)
	if result != int64(100) {
		t.Errorf("Expected 100, got %v", result)
	}
	
	// Test $m.CreatedAt (PascalCase)
	result = resolveValueRef("$m.CreatedAt", mol)
	if result != int64(100) {
		t.Errorf("Expected 100, got %v", result)
	}
	
	// Test $m.last_touched_at (snake_case)
	result = resolveValueRef("$m.last_touched_at", mol)
	if result != int64(200) {
		t.Errorf("Expected 200, got %v", result)
	}
	
	// Test $m.lastTouchedAt (camelCase)
	result = resolveValueRef("$m.lastTouchedAt", mol)
	if result != int64(200) {
		t.Errorf("Expected 200, got %v", result)
	}
	
	// Test $m.LastTouchedAt (PascalCase)
	result = resolveValueRef("$m.LastTouchedAt", mol)
	if result != int64(200) {
		t.Errorf("Expected 200, got %v", result)
	}
}

func TestMatchWhere(t *testing.T) {
	origin := NewMolecule("Origin", map[string]any{
		"value": 100,
		"name":  "origin",
	}, 0)
	
	candidate := NewMolecule("Test", map[string]any{
		"value": 100,
		"name":  "test",
	}, 0)
	
	// Test empty where (should match)
	where := WhereConfig{}
	if !matchWhere(where, candidate, origin) {
		t.Error("Expected match with empty where")
	}
	
	// Test matching condition
	where = WhereConfig{
		"value": EqCondition{Eq: 100},
	}
	if !matchWhere(where, candidate, origin) {
		t.Error("Expected match when value equals 100")
	}
	
	// Test non-matching condition
	where = WhereConfig{
		"value": EqCondition{Eq: 200},
	}
	if matchWhere(where, candidate, origin) {
		t.Error("Expected no match when value doesn't equal 200")
	}
	
	// Test missing field
	where = WhereConfig{
		"missing": EqCondition{Eq: "value"},
	}
	if matchWhere(where, candidate, origin) {
		t.Error("Expected no match when field is missing")
	}
	
	// Test with $m. reference
	where = WhereConfig{
		"value": EqCondition{Eq: "$m.value"},
	}
	if !matchWhere(where, candidate, origin) {
		t.Error("Expected match when using $m.value reference")
	}
	
	// Test multiple conditions (all must match)
	where = WhereConfig{
		"value": EqCondition{Eq: 100},
		"name":  EqCondition{Eq: "test"},
	}
	if !matchWhere(where, candidate, origin) {
		t.Error("Expected match when all conditions match")
	}
	
	// Test multiple conditions (one doesn't match)
	where = WhereConfig{
		"value": EqCondition{Eq: 100},
		"name":  EqCondition{Eq: "wrong"},
	}
	if matchWhere(where, candidate, origin) {
		t.Error("Expected no match when one condition doesn't match")
	}
}

func TestFilterBySpeciesAndWhere(t *testing.T) {
	// Create a simple environment view
	mol1 := NewMolecule("Species1", map[string]any{"value": 1}, 0)
	mol2 := NewMolecule("Species1", map[string]any{"value": 2}, 0)
	mol3 := NewMolecule("Species1", map[string]any{"value": 1, "tag": "a"}, 0)
	mol4 := NewMolecule("Species2", map[string]any{"value": 1}, 0)
	
	env := &mockEnvView{
		molecules: []Molecule{mol1, mol2, mol3, mol4},
		bySpecies: map[SpeciesName][]Molecule{
			"Species1": {mol1, mol2, mol3},
			"Species2": {mol4},
		},
	}
	
	origin := NewMolecule("Origin", map[string]any{"value": 1}, 0)
	
	// Test with empty where (should return all molecules of species)
	results := filterBySpeciesAndWhere(env, "Species1", WhereConfig{}, origin)
	if len(results) != 3 {
		t.Errorf("Expected 3 molecules, got %d", len(results))
	}
	
	// Test with where condition
	where := WhereConfig{
		"value": EqCondition{Eq: 1},
	}
	results = filterBySpeciesAndWhere(env, "Species1", where, origin)
	if len(results) != 2 {
		t.Errorf("Expected 2 molecules with value=1, got %d", len(results))
	}
	
	// Test with $m. reference
	where = WhereConfig{
		"value": EqCondition{Eq: "$m.value"},
	}
	results = filterBySpeciesAndWhere(env, "Species1", where, origin)
	if len(results) != 2 {
		t.Errorf("Expected 2 molecules matching origin value, got %d", len(results))
	}
	
	// Test with non-existent species
	results = filterBySpeciesAndWhere(env, "NonExistent", WhereConfig{}, origin)
	if len(results) != 0 {
		t.Errorf("Expected 0 molecules for non-existent species, got %d", len(results))
	}
	
	// Test with indexed envView
	indexedEnv := envView{
		molecules: []Molecule{mol1, mol2, mol3},
		bySpecies: map[SpeciesName][]Molecule{
			"Species1": {mol1, mol2, mol3},
		},
		bySpeciesFieldValue: map[SpeciesName]map[string]map[string][]Molecule{
			"Species1": {
				"value": {
					"1": {mol1, mol3},
					"2": {mol2},
				},
			},
		},
	}
	
	where = WhereConfig{
		"value": EqCondition{Eq: 1},
	}
	results = filterBySpeciesAndWhere(indexedEnv, "Species1", where, origin)
	if len(results) != 2 {
		t.Errorf("Expected 2 molecules using index, got %d", len(results))
	}
	
	// Test with $m. reference in indexed env
	where = WhereConfig{
		"value": EqCondition{Eq: "$m.value"},
	}
	results = filterBySpeciesAndWhere(indexedEnv, "Species1", where, origin)
	if len(results) != 2 {
		t.Errorf("Expected 2 molecules using index with $m. reference, got %d", len(results))
	}
	
	// Test with multiple where conditions (should fall back to linear scan)
	where = WhereConfig{
		"value": EqCondition{Eq: 1},
		"tag":   EqCondition{Eq: "a"},
	}
	results = filterBySpeciesAndWhere(indexedEnv, "Species1", where, origin)
	if len(results) != 1 {
		t.Errorf("Expected 1 molecule with both conditions, got %d", len(results))
	}
}

// mockEnvView is a simple implementation of EnvView for testing
type mockEnvView struct {
	molecules []Molecule
	bySpecies map[SpeciesName][]Molecule
}

func (m *mockEnvView) MoleculesBySpecies(species SpeciesName) []Molecule {
	mols, ok := m.bySpecies[species]
	if !ok {
		return nil
	}
	return mols
}

func (m *mockEnvView) Find(filter func(Molecule) bool) []Molecule {
	var results []Molecule
	for _, mol := range m.molecules {
		if filter(mol) {
			results = append(results, mol)
		}
	}
	return results
}

