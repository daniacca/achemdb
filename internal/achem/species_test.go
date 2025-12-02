package achem

import (
	"testing"
)

func TestSpeciesName(t *testing.T) {
	// Test SpeciesName as a string type
	var name SpeciesName = "TestSpecies"
	
	if name != "TestSpecies" {
		t.Errorf("Expected 'TestSpecies', got '%s'", name)
	}
	
	// Test conversion from string
	str := "AnotherSpecies"
	name = SpeciesName(str)
	if name != SpeciesName("AnotherSpecies") {
		t.Errorf("Expected 'AnotherSpecies', got '%s'", name)
	}
	
	// Test conversion to string
	str = string(name)
	if str != "AnotherSpecies" {
		t.Errorf("Expected 'AnotherSpecies', got '%s'", str)
	}
}

func TestSpecies(t *testing.T) {
	// Test creating a Species with minimal fields
	species := Species{
		Name:        "TestSpecies",
		Description: "A test species",
	}
	
	if species.Name != "TestSpecies" {
		t.Errorf("Expected Name 'TestSpecies', got '%s'", species.Name)
	}
	
	if species.Description != "A test species" {
		t.Errorf("Expected Description 'A test species', got '%s'", species.Description)
	}
	
	if species.Meta != nil {
		t.Error("Expected nil Meta for species without meta")
	}
	
	// Test Species with Meta
	meta := map[string]any{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}
	
	species = Species{
		Name:        "TestSpecies",
		Description: "A test species with meta",
		Meta:        meta,
	}
	
	if species.Meta == nil {
		t.Error("Expected non-nil Meta")
	}
	
	if species.Meta["key1"] != "value1" {
		t.Errorf("Expected meta key1 'value1', got %v", species.Meta["key1"])
	}
	
	if species.Meta["key2"] != 42 {
		t.Errorf("Expected meta key2 42, got %v", species.Meta["key2"])
	}
	
	if species.Meta["key3"] != true {
		t.Errorf("Expected meta key3 true, got %v", species.Meta["key3"])
	}
	
	// Test Species with empty description
	species = Species{
		Name: "MinimalSpecies",
	}
	
	if species.Description != "" {
		t.Errorf("Expected empty description, got '%s'", species.Description)
	}
	
	// Test Species with empty name
	species = Species{
		Name:        "",
		Description: "Empty name species",
	}
	
	if species.Name != "" {
		t.Errorf("Expected empty name, got '%s'", species.Name)
	}
}

func TestSpecies_Equality(t *testing.T) {
	species1 := Species{
		Name:        "TestSpecies",
		Description: "Description",
		Meta:        map[string]any{"key": "value"},
	}
	
	species2 := Species{
		Name:        "TestSpecies",
		Description: "Description",
		Meta:        map[string]any{"key": "value"},
	}
	
	// Test that they have the same name (used as key in maps)
	if species1.Name != species2.Name {
		t.Error("Expected species to have the same name")
	}
	
	// Test different species
	species3 := Species{
		Name: "DifferentSpecies",
	}
	
	if species1.Name == species3.Name {
		t.Error("Expected different species names")
	}
}

