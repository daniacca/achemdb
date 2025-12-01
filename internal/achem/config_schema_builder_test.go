package achem

import (
	"testing"
)

// testEnvView is a simple EnvView implementation for testing
type testEnvView struct {
	molecules []Molecule
}

func (v testEnvView) MoleculesBySpecies(species SpeciesName) []Molecule {
	var result []Molecule
	for _, m := range v.molecules {
		if m.Species == species {
			result = append(result, m)
		}
	}
	return result
}

func (v testEnvView) Find(filter func(Molecule) bool) []Molecule {
	var result []Molecule
	for _, m := range v.molecules {
		if filter(m) {
			result = append(result, m)
		}
	}
	return result
}

func TestConfigReaction_IfThenElse_FieldCondition(t *testing.T) {
	cfg := ReactionConfig{
		ID:   "test-if",
		Name: "Test If",
		Input: InputConfig{
			Species: "Input",
		},
		Rate: 1.0,
		Effects: []EffectConfig{
			{
				If: &IfConditionConfig{
					Field: "energy",
					Op:    "gt",
					Value: 3.0,
				},
				Then: []EffectConfig{
					{
						Create: &CreateEffectConfig{
							Species: "HighEnergy",
							Payload: map[string]any{"level": "high"},
						},
					},
				},
				Else: []EffectConfig{
					{
						Create: &CreateEffectConfig{
							Species: "LowEnergy",
							Payload: map[string]any{"level": "low"},
						},
					},
				},
			},
		},
	}

	reaction := &ConfigReaction{cfg: cfg}
	schema := NewSchema("test").WithReactions(reaction)
	env := NewEnvironment(schema)

	// Test with high energy molecule
	highEnergyMol := NewMolecule("Input", nil, 0)
	highEnergyMol.Energy = 5.0
	env.Insert(highEnergyMol)

	// Test with low energy molecule
	lowEnergyMol := NewMolecule("Input", nil, 0)
	lowEnergyMol.Energy = 2.0
	env.Insert(lowEnergyMol)

	env.Step()

	molecules := env.AllMolecules()
	// Should have 2 input molecules + 2 created molecules = 4 total
	if len(molecules) != 4 {
		t.Fatalf("Expected 4 molecules, got %d", len(molecules))
	}

	// Check that HighEnergy was created
	highEnergyCount := 0
	lowEnergyCount := 0
	for _, m := range molecules {
		if m.Species == "HighEnergy" {
			highEnergyCount++
		}
		if m.Species == "LowEnergy" {
			lowEnergyCount++
		}
	}

	if highEnergyCount != 1 {
		t.Errorf("Expected 1 HighEnergy molecule, got %d", highEnergyCount)
	}
	if lowEnergyCount != 1 {
		t.Errorf("Expected 1 LowEnergy molecule, got %d", lowEnergyCount)
	}
}

func TestConfigReaction_IfThenElse_CountMolecules(t *testing.T) {
	cfg := ReactionConfig{
		ID:   "test-count",
		Name: "Test Count",
		Input: InputConfig{
			Species: "Suspicion",
		},
		Rate: 1.0,
		Effects: []EffectConfig{
			{
				If: &IfConditionConfig{
					CountMolecules: &CountMoleculesConfig{
						Species: "Suspicion",
						Where: WhereConfig{
							"ip": EqCondition{Eq: "$m.ip"},
						},
						Op: map[string]any{"gte": 2}, // Each molecule sees 2 others (3 total - 1 self)
					},
				},
				Then: []EffectConfig{
					{
						Create: &CreateEffectConfig{
							Species: "Alert",
							Payload: map[string]any{"type": "multiple_suspicions"},
						},
					},
				},
			},
		},
	}

	reaction := &ConfigReaction{cfg: cfg}
	schema := NewSchema("test").WithReactions(reaction)
	env := NewEnvironment(schema)

	// Create 3 suspicion molecules with same IP
	// Each molecule will see 2 others (excluding itself), so count >= 2 will be true for all
	for i := 0; i < 3; i++ {
		mol := NewMolecule("Suspicion", map[string]any{"ip": "192.168.1.1"}, 0)
		env.Insert(mol)
	}

	env.Step()

	molecules := env.AllMolecules()
	// Should have 3 suspicions + 3 alerts (one per suspicion that matched the condition)
	if len(molecules) < 6 {
		t.Fatalf("Expected at least 6 molecules (3 suspicions + 3 alerts), got %d", len(molecules))
	}

	alertCount := 0
	for _, m := range molecules {
		if m.Species == "Alert" {
			alertCount++
		}
	}

	if alertCount != 3 {
		t.Errorf("Expected 3 Alert molecules (one per suspicion), got %d", alertCount)
	}
}

func TestConfigReaction_Partners_Required(t *testing.T) {
	cfg := ReactionConfig{
		ID:   "test-partners",
		Name: "Test Partners",
		Input: InputConfig{
			Species: "Suspicion",
			Partners: []PartnerConfig{
				{
					Species: "Suspicion",
					Where: WhereConfig{
						"ip": EqCondition{Eq: "$m.ip"},
					},
					Count: 1,
				},
			},
		},
		Rate: 1.0,
		Effects: []EffectConfig{
			{
				Create: &CreateEffectConfig{
					Species: "Alert",
					Payload: map[string]any{"type": "partner_found"},
				},
			},
		},
	}

	reaction := &ConfigReaction{cfg: cfg}
	schema := NewSchema("test").WithReactions(reaction)
	env := NewEnvironment(schema)

	// Create a suspicion molecule
	mol1 := NewMolecule("Suspicion", map[string]any{"ip": "192.168.1.1"}, 0)
	env.Insert(mol1)

	// Step without partner - should not create alert
	env.Step()
	molecules := env.AllMolecules()
	if len(molecules) != 1 {
		t.Fatalf("Expected 1 molecule (no partner, no reaction), got %d", len(molecules))
	}

	// Add a partner with matching IP
	mol2 := NewMolecule("Suspicion", map[string]any{"ip": "192.168.1.1"}, 0)
	env.Insert(mol2)

	// Step with partner - should create alert
	env.Step()
	molecules = env.AllMolecules()
	// Should have 2 suspicions + 2 alerts (each suspicion finds the other as partner)
	if len(molecules) < 3 {
		t.Fatalf("Expected at least 3 molecules (2 suspicions + alerts), got %d", len(molecules))
	}

	alertCount := 0
	for _, m := range molecules {
		if m.Species == "Alert" {
			alertCount++
		}
	}

	if alertCount == 0 {
		t.Error("Expected at least one Alert to be created when partner is found")
	}
}

func TestConfigReaction_Partners_DifferentIP(t *testing.T) {
	cfg := ReactionConfig{
		ID:   "test-partners-diff",
		Name: "Test Partners Different IP",
		Input: InputConfig{
			Species: "Suspicion",
			Partners: []PartnerConfig{
				{
					Species: "Suspicion",
					Where: WhereConfig{
						"ip": EqCondition{Eq: "$m.ip"},
					},
					Count: 1,
				},
			},
		},
		Rate: 1.0,
		Effects: []EffectConfig{
			{
				Create: &CreateEffectConfig{
					Species: "Alert",
				},
			},
		},
	}

	reaction := &ConfigReaction{cfg: cfg}
	schema := NewSchema("test").WithReactions(reaction)
	env := NewEnvironment(schema)

	// Create suspicions with different IPs
	mol1 := NewMolecule("Suspicion", map[string]any{"ip": "192.168.1.1"}, 0)
	mol2 := NewMolecule("Suspicion", map[string]any{"ip": "192.168.1.2"}, 0)
	env.Insert(mol1)
	env.Insert(mol2)

	env.Step()

	molecules := env.AllMolecules()
	// Should have only 2 suspicions (no alerts, no matching partners)
	if len(molecules) != 2 {
		t.Fatalf("Expected 2 molecules (no matching partners), got %d", len(molecules))
	}

	for _, m := range molecules {
		if m.Species == "Alert" {
			t.Error("Expected no Alert to be created when partners have different IPs")
		}
	}
}

func TestConfigReaction_FieldComparison_Operators(t *testing.T) {
	testCases := []struct {
		name     string
		field    string
		op       string
		value    any
		molValue any
		expected bool
	}{
		{"eq_true", "energy", "eq", 5.0, 5.0, true},
		{"eq_false", "energy", "eq", 5.0, 3.0, false},
		{"gt_true", "energy", "gt", 3.0, 5.0, true},
		{"gt_false", "energy", "gt", 5.0, 3.0, false},
		{"gte_true_equal", "energy", "gte", 5.0, 5.0, true},
		{"gte_true_greater", "energy", "gte", 3.0, 5.0, true},
		{"gte_false", "energy", "gte", 5.0, 3.0, false},
		{"lt_true", "energy", "lt", 5.0, 3.0, true},
		{"lt_false", "energy", "lt", 3.0, 5.0, false},
		{"lte_true_equal", "energy", "lte", 5.0, 5.0, true},
		{"lte_true_less", "energy", "lte", 5.0, 3.0, true},
		{"lte_false", "energy", "lte", 3.0, 5.0, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := ReactionConfig{
				ID:   "test-op",
				Name: "Test Operator",
				Input: InputConfig{
					Species: "Test",
				},
				Rate: 1.0,
				Effects: []EffectConfig{
					{
						If: &IfConditionConfig{
							Field: tc.field,
							Op:    tc.op,
							Value: tc.value,
						},
						Then: []EffectConfig{
							{
								Create: &CreateEffectConfig{
									Species: "Result",
								},
							},
						},
					},
				},
			}

			reaction := &ConfigReaction{cfg: cfg}
			schema := NewSchema("test").WithReactions(reaction)
			env := NewEnvironment(schema)

			mol := NewMolecule("Test", nil, 0)
			switch tc.field {
			case "energy":
				if val, ok := tc.molValue.(float64); ok {
					mol.Energy = val
				}
			}
			env.Insert(mol)

			env.Step()

			molecules := env.AllMolecules()
			resultCount := 0
			for _, m := range molecules {
				if m.Species == "Result" {
					resultCount++
				}
			}

			if tc.expected && resultCount == 0 {
				t.Errorf("Expected Result molecule to be created, but it wasn't")
			}
			if !tc.expected && resultCount > 0 {
				t.Errorf("Expected no Result molecule, but %d were created", resultCount)
			}
		})
	}
}

func TestConfigReaction_ResolveValueFromMolecule(t *testing.T) {
	mol := NewMolecule("Test", map[string]any{"ip": "192.168.1.1", "port": 8080}, 0)
	mol.Energy = 5.0
	mol.Stability = 0.8

	testCases := []struct {
		name     string
		input    any
		expected any
	}{
		{"direct_value", "test", "test"},
		{"payload_ref", "$m.ip", "192.168.1.1"},
		{"payload_ref_int", "$m.port", 8080},
		{"energy_ref", "$m.energy", 5.0},
		{"stability_ref", "$m.stability", 0.8},
		{"non_string", 42, 42},
		{"missing_field", "$m.missing", "$m.missing"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := resolveValueFromMolecule(tc.input, mol)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestConfigReaction_NestedIfConditions(t *testing.T) {
	cfg := ReactionConfig{
		ID:   "test-nested",
		Name: "Test Nested If",
		Input: InputConfig{
			Species: "Input",
		},
		Rate: 1.0,
		Effects: []EffectConfig{
			{
				If: &IfConditionConfig{
					Field: "energy",
					Op:    "gt",
					Value: 5.0,
				},
				Then: []EffectConfig{
					{
						If: &IfConditionConfig{
							Field: "energy",
							Op:    "lt",
							Value: 10.0,
						},
						Then: []EffectConfig{
							{
								Create: &CreateEffectConfig{
									Species: "Medium",
								},
							},
						},
						Else: []EffectConfig{
							{
								Create: &CreateEffectConfig{
									Species: "High",
								},
							},
						},
					},
				},
				Else: []EffectConfig{
					{
						Create: &CreateEffectConfig{
							Species: "Low",
						},
					},
				},
			},
		},
	}

	reaction := &ConfigReaction{cfg: cfg}
	schema := NewSchema("test").WithReactions(reaction)
	env := NewEnvironment(schema)

	// Test medium energy (5 < energy < 10)
	mediumMol := NewMolecule("Input", nil, 0)
	mediumMol.Energy = 7.0
	env.Insert(mediumMol)

	env.Step()

	molecules := env.AllMolecules()
	mediumCount := 0
	for _, m := range molecules {
		if m.Species == "Medium" {
			mediumCount++
		}
	}

	if mediumCount != 1 {
		t.Errorf("Expected 1 Medium molecule, got %d", mediumCount)
	}
}

func TestConfigReaction_Partners_CountZero(t *testing.T) {
	cfg := ReactionConfig{
		ID:   "test-partners-zero",
		Name: "Test Partners Count Zero",
		Input: InputConfig{
			Species: "Suspicion",
			Partners: []PartnerConfig{
				{
					Species: "Suspicion",
					Where: WhereConfig{
						"ip": EqCondition{Eq: "$m.ip"},
					},
					Count: 0, // Should default to 1
				},
			},
		},
		Rate: 1.0,
		Effects: []EffectConfig{
			{
				Create: &CreateEffectConfig{
					Species: "Alert",
				},
			},
		},
	}

	reaction := &ConfigReaction{cfg: cfg}
	schema := NewSchema("test").WithReactions(reaction)
	env := NewEnvironment(schema)

	// Create a suspicion without partner
	mol1 := NewMolecule("Suspicion", map[string]any{"ip": "192.168.1.1"}, 0)
	env.Insert(mol1)

	env.Step()

	molecules := env.AllMolecules()
	// Should have only 1 molecule (no partner, no reaction)
	if len(molecules) != 1 {
		t.Fatalf("Expected 1 molecule (count 0 defaults to 1, no partner), got %d", len(molecules))
	}
}

func TestConfigReaction_Catalysts_Basic(t *testing.T) {
	cfg := ReactionConfig{
		ID:   "test-catalyst",
		Name: "Test Catalyst",
		Input: InputConfig{
			Species: "Input",
		},
		Rate: 0.5, // Base rate 0.5
		Catalysts: []CatalystConfig{
			{
				Species:   "Catalyst",
				RateBoost: 0.3, // Should boost to 0.8
			},
		},
		Effects: []EffectConfig{
			{
				Create: &CreateEffectConfig{
					Species: "Output",
				},
			},
		},
	}

	reaction := &ConfigReaction{cfg: cfg}
	schema := NewSchema("test").WithReactions(reaction)
	env := NewEnvironment(schema)

	// Create input molecule
	inputMol := NewMolecule("Input", nil, 0)
	env.Insert(inputMol)

	// Check effective rate without catalyst
	effectiveRate := reaction.EffectiveRate(inputMol, testEnvView{molecules: []Molecule{inputMol}})
	if effectiveRate != 0.5 {
		t.Errorf("Expected effective rate 0.5 without catalyst, got %f", effectiveRate)
	}

	// Add catalyst
	catalystMol := NewMolecule("Catalyst", nil, 0)
	env.Insert(catalystMol)

	// Check effective rate with catalyst
	allMolecules := env.AllMolecules()
	effectiveRate = reaction.EffectiveRate(inputMol, testEnvView{molecules: allMolecules})
	if effectiveRate != 0.8 {
		t.Errorf("Expected effective rate 0.8 with catalyst, got %f", effectiveRate)
	}
}

func TestConfigReaction_Catalysts_WithWhere(t *testing.T) {
	cfg := ReactionConfig{
		ID:   "test-catalyst-where",
		Name: "Test Catalyst With Where",
		Input: InputConfig{
			Species: "Input",
		},
		Rate: 0.3,
		Catalysts: []CatalystConfig{
			{
				Species:   "Catalyst",
				Where:     WhereConfig{"type": EqCondition{Eq: "$m.type"}},
				RateBoost: 0.4,
			},
		},
		Effects: []EffectConfig{
			{
				Create: &CreateEffectConfig{
					Species: "Output",
				},
			},
		},
	}

	reaction := &ConfigReaction{cfg: cfg}
	schema := NewSchema("test").WithReactions(reaction)
	env := NewEnvironment(schema)

	// Create input molecule with type
	inputMol := NewMolecule("Input", map[string]any{"type": "A"}, 0)
	env.Insert(inputMol)

	// Add catalyst with matching type
	catalystMol1 := NewMolecule("Catalyst", map[string]any{"type": "A"}, 0)
	env.Insert(catalystMol1)

	// Add catalyst with non-matching type
	catalystMol2 := NewMolecule("Catalyst", map[string]any{"type": "B"}, 0)
	env.Insert(catalystMol2)

	allMolecules := env.AllMolecules()
	effectiveRate := reaction.EffectiveRate(inputMol, testEnvView{molecules: allMolecules})
	// Should be 0.3 (base) + 0.4 (boost from matching catalyst) = 0.7
	if effectiveRate != 0.7 {
		t.Errorf("Expected effective rate 0.7 with matching catalyst, got %f", effectiveRate)
	}
}

func TestConfigReaction_Catalysts_MaxRate(t *testing.T) {
	maxRate := 0.6
	cfg := ReactionConfig{
		ID:   "test-catalyst-max",
		Name: "Test Catalyst Max Rate",
		Input: InputConfig{
			Species: "Input",
		},
		Rate: 0.3,
		Catalysts: []CatalystConfig{
			{
				Species:   "Catalyst",
				RateBoost: 0.5, // Would make it 0.8, but max is 0.6
				MaxRate:   &maxRate,
			},
		},
		Effects: []EffectConfig{
			{
				Create: &CreateEffectConfig{
					Species: "Output",
				},
			},
		},
	}

	reaction := &ConfigReaction{cfg: cfg}
	schema := NewSchema("test").WithReactions(reaction)
	env := NewEnvironment(schema)

	inputMol := NewMolecule("Input", nil, 0)
	env.Insert(inputMol)

	catalystMol := NewMolecule("Catalyst", nil, 0)
	env.Insert(catalystMol)

	allMolecules := env.AllMolecules()
	effectiveRate := reaction.EffectiveRate(inputMol, testEnvView{molecules: allMolecules})
	// Should be capped at 0.6 (max rate), not 0.8
	if effectiveRate != 0.6 {
		t.Errorf("Expected effective rate 0.6 (capped by max), got %f", effectiveRate)
	}
}

func TestConfigReaction_Catalysts_Multiple(t *testing.T) {
	cfg := ReactionConfig{
		ID:   "test-catalyst-multiple",
		Name: "Test Multiple Catalysts",
		Input: InputConfig{
			Species: "Input",
		},
		Rate: 0.2,
		Catalysts: []CatalystConfig{
			{
				Species:   "Catalyst1",
				RateBoost: 0.2,
			},
			{
				Species:   "Catalyst2",
				RateBoost: 0.3,
			},
		},
		Effects: []EffectConfig{
			{
				Create: &CreateEffectConfig{
					Species: "Output",
				},
			},
		},
	}

	reaction := &ConfigReaction{cfg: cfg}
	schema := NewSchema("test").WithReactions(reaction)
	env := NewEnvironment(schema)

	inputMol := NewMolecule("Input", nil, 0)
	env.Insert(inputMol)

	// Add both catalysts
	catalyst1 := NewMolecule("Catalyst1", nil, 0)
	catalyst2 := NewMolecule("Catalyst2", nil, 0)
	env.Insert(catalyst1)
	env.Insert(catalyst2)

	allMolecules := env.AllMolecules()
	effectiveRate := reaction.EffectiveRate(inputMol, testEnvView{molecules: allMolecules})
	// Should be 0.2 (base) + 0.2 (catalyst1) + 0.3 (catalyst2) = 0.7
	if effectiveRate != 0.7 {
		t.Errorf("Expected effective rate 0.7 with multiple catalysts, got %f", effectiveRate)
	}
}

func TestConfigReaction_Catalysts_DefaultBoost(t *testing.T) {
	cfg := ReactionConfig{
		ID:   "test-catalyst-default",
		Name: "Test Catalyst Default Boost",
		Input: InputConfig{
			Species: "Input",
		},
		Rate: 0.4,
		Catalysts: []CatalystConfig{
			{
				Species: "Catalyst",
				// No RateBoost specified, should default to 0.1
			},
		},
		Effects: []EffectConfig{
			{
				Create: &CreateEffectConfig{
					Species: "Output",
				},
			},
		},
	}

	reaction := &ConfigReaction{cfg: cfg}
	schema := NewSchema("test").WithReactions(reaction)
	env := NewEnvironment(schema)

	inputMol := NewMolecule("Input", nil, 0)
	env.Insert(inputMol)

	catalystMol := NewMolecule("Catalyst", nil, 0)
	env.Insert(catalystMol)

	allMolecules := env.AllMolecules()
	effectiveRate := reaction.EffectiveRate(inputMol, testEnvView{molecules: allMolecules})
	// Should be 0.4 (base) + 0.1 (default boost) = 0.5
	if effectiveRate != 0.5 {
		t.Errorf("Expected effective rate 0.5 with default boost, got %f", effectiveRate)
	}
}

func TestConfigReaction_Catalysts_RateCappedAtOne(t *testing.T) {
	cfg := ReactionConfig{
		ID:   "test-catalyst-cap",
		Name: "Test Catalyst Rate Cap",
		Input: InputConfig{
			Species: "Input",
		},
		Rate: 0.8,
		Catalysts: []CatalystConfig{
			{
				Species:   "Catalyst",
				RateBoost: 0.5, // Would make it 1.3, but should cap at 1.0
			},
		},
		Effects: []EffectConfig{
			{
				Create: &CreateEffectConfig{
					Species: "Output",
				},
			},
		},
	}

	reaction := &ConfigReaction{cfg: cfg}
	schema := NewSchema("test").WithReactions(reaction)
	env := NewEnvironment(schema)

	inputMol := NewMolecule("Input", nil, 0)
	env.Insert(inputMol)

	catalystMol := NewMolecule("Catalyst", nil, 0)
	env.Insert(catalystMol)

	allMolecules := env.AllMolecules()
	effectiveRate := reaction.EffectiveRate(inputMol, testEnvView{molecules: allMolecules})
	// Should be capped at 1.0
	if effectiveRate != 1.0 {
		t.Errorf("Expected effective rate 1.0 (capped), got %f", effectiveRate)
	}
}

func TestConfigReaction_Catalysts_CanBeSameMolecule(t *testing.T) {
	cfg := ReactionConfig{
		ID:   "test-catalyst-self",
		Name: "Test Catalyst Can Be Self",
		Input: InputConfig{
			Species: "Input",
		},
		Rate: 0.3,
		Catalysts: []CatalystConfig{
			{
				Species:   "Input", // Same species as input
				RateBoost: 0.2,
			},
		},
		Effects: []EffectConfig{
			{
				Create: &CreateEffectConfig{
					Species: "Output",
				},
			},
		},
	}

	reaction := &ConfigReaction{cfg: cfg}
	schema := NewSchema("test").WithReactions(reaction)
	env := NewEnvironment(schema)

	// Create input molecule - it can catalyze itself
	inputMol := NewMolecule("Input", nil, 0)
	env.Insert(inputMol)

	allMolecules := env.AllMolecules()
	effectiveRate := reaction.EffectiveRate(inputMol, testEnvView{molecules: allMolecules})
	// Should be 0.3 (base) + 0.2 (self-catalysis) = 0.5
	if effectiveRate != 0.5 {
		t.Errorf("Expected effective rate 0.5 with self-catalysis, got %f", effectiveRate)
	}
}

func TestConfigReaction_Catalysts_Integration(t *testing.T) {
	// Test that catalysts actually increase the probability of reaction firing
	cfg := ReactionConfig{
		ID:   "test-catalyst-integration",
		Name: "Test Catalyst Integration",
		Input: InputConfig{
			Species: "Input",
		},
		Rate: 0.1, // Low base rate
		Catalysts: []CatalystConfig{
			{
				Species:   "Catalyst",
				RateBoost: 0.8, // High boost to make it 0.9
			},
		},
		Effects: []EffectConfig{
			{
				Create: &CreateEffectConfig{
					Species: "Output",
				},
			},
		},
	}

	reaction := &ConfigReaction{cfg: cfg}
	schema := NewSchema("test").WithReactions(reaction)
	env := NewEnvironment(schema)

	// Create input molecule
	inputMol := NewMolecule("Input", nil, 0)
	env.Insert(inputMol)

	// Without catalyst, reaction should rarely fire (rate 0.1)
	// We can't test probability deterministically, but we can test that
	// the effective rate is higher with catalyst

	// Check without catalyst
	effectiveRateNoCat := reaction.EffectiveRate(inputMol, testEnvView{molecules: []Molecule{inputMol}})
	if effectiveRateNoCat != 0.1 {
		t.Errorf("Expected effective rate 0.1 without catalyst, got %f", effectiveRateNoCat)
	}

	// Add catalyst
	catalystMol := NewMolecule("Catalyst", nil, 0)
	env.Insert(catalystMol)

	// Check with catalyst
	allMolecules := env.AllMolecules()
	effectiveRateWithCat := reaction.EffectiveRate(inputMol, testEnvView{molecules: allMolecules})
	if effectiveRateWithCat != 0.9 {
		t.Errorf("Expected effective rate 0.9 with catalyst, got %f", effectiveRateWithCat)
	}

	if effectiveRateWithCat <= effectiveRateNoCat {
		t.Error("Expected effective rate to be higher with catalyst")
	}
}

