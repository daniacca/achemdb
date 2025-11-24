package achem

import (
	"testing"
)

func TestNewMolecule(t *testing.T) {
	tests := []struct {
		name     string
		species  SpeciesName
		payload  map[string]any
		time     int64
		validate func(t *testing.T, m Molecule)
	}{
		{
			name:    "basic molecule creation",
			species: "TestSpecies",
			payload: map[string]any{"key": "value"},
			time:    100,
			validate: func(t *testing.T, m Molecule) {
				if m.Species != "TestSpecies" {
					t.Errorf("Expected species 'TestSpecies', got '%s'", m.Species)
				}
				if m.Payload["key"] != "value" {
					t.Errorf("Expected payload key='value', got '%v'", m.Payload["key"])
				}
				if m.Energy != 1.0 {
					t.Errorf("Expected default energy 1.0, got %f", m.Energy)
				}
				if m.Stability != 1.0 {
					t.Errorf("Expected default stability 1.0, got %f", m.Stability)
				}
				if m.CreatedAt != 100 {
					t.Errorf("Expected CreatedAt 100, got %d", m.CreatedAt)
				}
				if m.LastTouchedAt != 100 {
					t.Errorf("Expected LastTouchedAt 100, got %d", m.LastTouchedAt)
				}
				if m.ID == "" {
					t.Error("Expected non-empty ID")
				}
				if m.Tags != nil {
					t.Error("Expected nil tags")
				}
			},
		},
		{
			name:    "molecule with nil payload",
			species: "EmptySpecies",
			payload: nil,
			time:    0,
			validate: func(t *testing.T, m Molecule) {
				if m.Species != "EmptySpecies" {
					t.Errorf("Expected species 'EmptySpecies', got '%s'", m.Species)
				}
				// Payload can be nil, which is valid
				if len(m.Payload) != 0 {
					t.Errorf("Expected nil or empty payload, got %v", m.Payload)
				}
			},
		},
		{
			name:    "molecule with complex payload",
			species: "ComplexSpecies",
			payload: map[string]any{
				"string": "test",
				"int":    42,
				"float":  3.14,
				"bool":   true,
			},
			time: 999,
			validate: func(t *testing.T, m Molecule) {
				if m.Payload["string"] != "test" {
					t.Errorf("Expected string='test', got '%v'", m.Payload["string"])
				}
				if m.Payload["int"] != 42 {
					t.Errorf("Expected int=42, got '%v'", m.Payload["int"])
				}
				if m.Payload["float"] != 3.14 {
					t.Errorf("Expected float=3.14, got '%v'", m.Payload["float"])
				}
				if m.Payload["bool"] != true {
					t.Errorf("Expected bool=true, got '%v'", m.Payload["bool"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMolecule(tt.species, tt.payload, tt.time)
			tt.validate(t, m)
		})
	}
}

func TestNewMolecule_UniqueIDs(t *testing.T) {
	// Test that each molecule gets a unique ID
	ids := make(map[MoleculeID]bool)
	for i := 0; i < 100; i++ {
		m := NewMolecule("Test", nil, 0)
		if ids[m.ID] {
			t.Errorf("Duplicate ID found: %s", m.ID)
		}
		ids[m.ID] = true
	}
}

