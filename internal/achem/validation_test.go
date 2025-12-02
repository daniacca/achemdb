package achem

import (
	"strings"
	"testing"
)

func TestValidateSchemaConfig_ValidConfig(t *testing.T) {
	cfg := SchemaConfig{
		Name: "test_schema",
		Species: []SpeciesConfig{
			{Name: "A"},
			{Name: "B"},
		},
		Reactions: []ReactionConfig{
			{
				ID:   "r1",
				Name: "reaction 1",
				Input: InputConfig{
					Species: "A",
				},
				Effects: []EffectConfig{
					{
						Create: &CreateEffectConfig{
							Species: "B",
						},
					},
				},
			},
		},
	}

	err := ValidateSchemaConfig(cfg)
	if err != nil {
		t.Fatalf("expected no validation error, got: %v", err)
	}
}

func TestValidateSchemaConfig_DuplicateSpecies(t *testing.T) {
	cfg := SchemaConfig{
		Name: "test_schema",
		Species: []SpeciesConfig{
			{Name: "A"},
			{Name: "A"}, // duplicate
		},
		Reactions: []ReactionConfig{},
	}

	err := ValidateSchemaConfig(cfg)
	if err == nil {
		t.Fatal("expected validation error for duplicate species, got nil")
	}

	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	if !strings.Contains(err.Error(), "duplicate species name") {
		t.Fatalf("expected error message about duplicate species, got: %v", err)
	}

	if len(validationErr.Issues) == 0 {
		t.Fatal("expected at least one validation issue")
	}
}

func TestValidateSchemaConfig_InvalidReactionSpecies(t *testing.T) {
	cfg := SchemaConfig{
		Name: "test_schema",
		Species: []SpeciesConfig{
			{Name: "A"},
		},
		Reactions: []ReactionConfig{
			{
				ID:   "r1",
				Name: "reaction 1",
				Input: InputConfig{
					Species: "B", // doesn't exist
				},
				Effects: []EffectConfig{},
			},
		},
	}

	err := ValidateSchemaConfig(cfg)
	if err == nil {
		t.Fatal("expected validation error for invalid reaction species, got nil")
	}

	if !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("expected error message about species not existing, got: %v", err)
	}
}

func TestValidateSchemaConfig_CountMoleculesInvalidOp(t *testing.T) {
	tests := []struct {
		name    string
		op      map[string]any
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty op",
			op:      map[string]any{},
			wantErr: true,
			errMsg:  "exactly one operator",
		},
		{
			name:    "multiple operators",
			op:      map[string]any{"eq": 1, "gt": 2},
			wantErr: true,
			errMsg:  "exactly one operator",
		},
		{
			name:    "invalid operator",
			op:      map[string]any{"invalid": 1},
			wantErr: true,
			errMsg:  "invalid operator",
		},
		{
			name:    "valid operator",
			op:      map[string]any{"eq": 1},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := SchemaConfig{
				Name: "test_schema",
				Species: []SpeciesConfig{
					{Name: "A"},
				},
				Reactions: []ReactionConfig{
					{
						ID:   "r1",
						Name: "reaction 1",
						Input: InputConfig{
							Species: "A",
						},
						Effects: []EffectConfig{
							{
								If: &IfConditionConfig{
									CountMolecules: &CountMoleculesConfig{
										Species: "A",
										Op:      tt.op,
									},
								},
							},
						},
					},
				},
			}

			err := ValidateSchemaConfig(cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected validation error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Fatalf("expected error message containing %q, got: %v", tt.errMsg, err)
				}
			} else {
				if err != nil {
					t.Fatalf("expected no validation error, got: %v", err)
				}
			}
		})
	}
}

func TestValidateSchemaConfig_DuplicateReactionID(t *testing.T) {
	cfg := SchemaConfig{
		Name: "test_schema",
		Species: []SpeciesConfig{
			{Name: "A"},
		},
		Reactions: []ReactionConfig{
			{
				ID:   "r1",
				Name: "reaction 1",
				Input: InputConfig{
					Species: "A",
				},
				Effects: []EffectConfig{},
			},
			{
				ID:   "r1", // duplicate
				Name: "reaction 2",
				Input: InputConfig{
					Species: "A",
				},
				Effects: []EffectConfig{},
			},
		},
	}

	err := ValidateSchemaConfig(cfg)
	if err == nil {
		t.Fatal("expected validation error for duplicate reaction ID, got nil")
	}

	if !strings.Contains(err.Error(), "duplicate reaction ID") {
		t.Fatalf("expected error message about duplicate reaction ID, got: %v", err)
	}
}

func TestValidateSchemaConfig_InvalidPartnerSpecies(t *testing.T) {
	cfg := SchemaConfig{
		Name: "test_schema",
		Species: []SpeciesConfig{
			{Name: "A"},
		},
		Reactions: []ReactionConfig{
			{
				ID:   "r1",
				Name: "reaction 1",
				Input: InputConfig{
					Species: "A",
					Partners: []PartnerConfig{
						{Species: "B"}, // doesn't exist
					},
				},
				Effects: []EffectConfig{},
			},
		},
	}

	err := ValidateSchemaConfig(cfg)
	if err == nil {
		t.Fatal("expected validation error for invalid partner species, got nil")
	}

	if !strings.Contains(err.Error(), "partner species") || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("expected error message about partner species not existing, got: %v", err)
	}
}

func TestValidateSchemaConfig_InvalidCreateEffectSpecies(t *testing.T) {
	cfg := SchemaConfig{
		Name: "test_schema",
		Species: []SpeciesConfig{
			{Name: "A"},
		},
		Reactions: []ReactionConfig{
			{
				ID:   "r1",
				Name: "reaction 1",
				Input: InputConfig{
					Species: "A",
				},
				Effects: []EffectConfig{
					{
						Create: &CreateEffectConfig{
							Species: "B", // doesn't exist
						},
					},
				},
			},
		},
	}

	err := ValidateSchemaConfig(cfg)
	if err == nil {
		t.Fatal("expected validation error for invalid create effect species, got nil")
	}

	if !strings.Contains(err.Error(), "create effect species") || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("expected error message about create effect species not existing, got: %v", err)
	}
}

