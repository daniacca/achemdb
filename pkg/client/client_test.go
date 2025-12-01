package client

import (
	"testing"

	"github.com/daniacca/achemdb/internal/achem"
)

func TestSchemaBuilder(t *testing.T) {
	schema := NewSchema("test-schema").
		Species("Species1", "Description 1", nil).
		Species("Species2", "Description 2", map[string]any{"meta": "value"})

	cfg := schema.Build()

	if cfg.Name != "test-schema" {
		t.Errorf("Expected name 'test-schema', got '%s'", cfg.Name)
	}

	if len(cfg.Species) != 2 {
		t.Errorf("Expected 2 species, got %d", len(cfg.Species))
	}

	if cfg.Species[0].Name != "Species1" {
		t.Errorf("Expected first species 'Species1', got '%s'", cfg.Species[0].Name)
	}
}

func TestReactionBuilder(t *testing.T) {
	reaction := NewReaction("test_reaction").
		Name("Test Reaction").
		Input("InputSpecies", WhereEq("status", "active")).
		Rate(0.8).
		Effect(
			Consume(),
			Create("OutputSpecies").
				Payload("field", "value").
				Energy(1.0).
				Stability(0.5),
		)

	cfg := reaction.Build()

	if cfg.ID != "test_reaction" {
		t.Errorf("Expected ID 'test_reaction', got '%s'", cfg.ID)
	}

	if cfg.Name != "Test Reaction" {
		t.Errorf("Expected name 'Test Reaction', got '%s'", cfg.Name)
	}

	if cfg.Input.Species != "InputSpecies" {
		t.Errorf("Expected input species 'InputSpecies', got '%s'", cfg.Input.Species)
	}

	if cfg.Rate != 0.8 {
		t.Errorf("Expected rate 0.8, got %f", cfg.Rate)
	}

	if len(cfg.Effects) != 2 {
		t.Errorf("Expected 2 effects, got %d", len(cfg.Effects))
	}

	if !cfg.Effects[0].Consume {
		t.Error("Expected first effect to be consume")
	}

	if cfg.Effects[1].Create == nil {
		t.Error("Expected second effect to be create")
	}

	if cfg.Effects[1].Create.Species != "OutputSpecies" {
		t.Errorf("Expected create species 'OutputSpecies', got '%s'", cfg.Effects[1].Create.Species)
	}
}

func TestInputBuilder(t *testing.T) {
	input := NewInput("TestSpecies").
		WhereEq("field1", "value1").
		WhereEq("field2", 42)

	cfg := input.Build()

	if cfg.Species != "TestSpecies" {
		t.Errorf("Expected species 'TestSpecies', got '%s'", cfg.Species)
	}

	if len(cfg.Where) != 2 {
		t.Errorf("Expected 2 where conditions, got %d", len(cfg.Where))
	}

	if cfg.Where["field1"].Eq != "value1" {
		t.Errorf("Expected field1='value1', got %v", cfg.Where["field1"].Eq)
	}
}

func TestCreateEffectBuilder(t *testing.T) {
	create := Create("NewSpecies").
		Payload("key1", "value1").
		Payload("key2", Ref("m.field")).
		Energy(2.0).
		Stability(1.5)

	cfg := create.Build()

	if cfg.Species != "NewSpecies" {
		t.Errorf("Expected species 'NewSpecies', got '%s'", cfg.Species)
	}

	if len(cfg.Payload) != 2 {
		t.Errorf("Expected 2 payload fields, got %d", len(cfg.Payload))
	}

	if cfg.Payload["key2"] != "$m.field" {
		t.Errorf("Expected key2='$m.field', got %v", cfg.Payload["key2"])
	}

	if cfg.Energy == nil || *cfg.Energy != 2.0 {
		t.Errorf("Expected energy 2.0, got %v", cfg.Energy)
	}

	if cfg.Stability == nil || *cfg.Stability != 1.5 {
		t.Errorf("Expected stability 1.5, got %v", cfg.Stability)
	}
}

func TestUpdateEffectBuilder(t *testing.T) {
	update := Update().EnergyAdd(0.5)

	cfg := update.Build()

	if cfg.EnergyAdd == nil || *cfg.EnergyAdd != 0.5 {
		t.Errorf("Expected energy_add 0.5, got %v", cfg.EnergyAdd)
	}
}

func TestIfConditionBuilder(t *testing.T) {
	ifEffect := If(NewIfField("energy", "gt", 3.0)).
		Then(
			Create("HighEnergy"),
		).
		Else(
			Create("LowEnergy"),
		)

	// Convert IfEffectBuilder to EffectBuilder
	effectBuilder := &EffectBuilder{ifCond: ifEffect.ifCond}
	cfg := effectBuilder.Build()

	if cfg.If == nil {
		t.Fatal("Expected If condition to be set")
	}

	if cfg.If.Field != "energy" {
		t.Errorf("Expected field 'energy', got '%s'", cfg.If.Field)
	}

	if cfg.If.Op != "gt" {
		t.Errorf("Expected op 'gt', got '%s'", cfg.If.Op)
	}

	if cfg.If.Value != 3.0 {
		t.Errorf("Expected value 3.0, got %v", cfg.If.Value)
	}

	if len(cfg.Then) != 1 {
		t.Errorf("Expected 1 then effect, got %d", len(cfg.Then))
	}

	if len(cfg.Else) != 1 {
		t.Errorf("Expected 1 else effect, got %d", len(cfg.Else))
	}
}

func TestCountMoleculesBuilder(t *testing.T) {
	count := NewCountMolecules("Suspicion").
		WhereEq("ip", Ref("m.ip")).
		Op("gte", 3)

	cfg := count.Build()

	if cfg.Species != "Suspicion" {
		t.Errorf("Expected species 'Suspicion', got '%s'", cfg.Species)
	}

	if len(cfg.Where) != 1 {
		t.Errorf("Expected 1 where condition, got %d", len(cfg.Where))
	}

	if cfg.Op["gte"] != 3 {
		t.Errorf("Expected op gte=3, got %v", cfg.Op["gte"])
	}
}

func TestPartnerBuilder(t *testing.T) {
	partner := NewPartner("PartnerSpecies").
		WhereEq("ip", Ref("m.ip")).
		Count(2)

	cfg := partner.Build()

	if cfg.Species != "PartnerSpecies" {
		t.Errorf("Expected species 'PartnerSpecies', got '%s'", cfg.Species)
	}

	if cfg.Count != 2 {
		t.Errorf("Expected count 2, got %d", cfg.Count)
	}
}

func TestCatalystBuilder(t *testing.T) {
	maxRate := 0.9
	catalyst := NewCatalyst("CatalystSpecies").
		WhereEq("type", "active").
		RateBoost(0.5).
		MaxRate(maxRate)

	cfg := catalyst.Build()

	if cfg.Species != "CatalystSpecies" {
		t.Errorf("Expected species 'CatalystSpecies', got '%s'", cfg.Species)
	}

	if cfg.RateBoost != 0.5 {
		t.Errorf("Expected rate_boost 0.5, got %f", cfg.RateBoost)
	}

	if cfg.MaxRate == nil || *cfg.MaxRate != 0.9 {
		t.Errorf("Expected max_rate 0.9, got %v", cfg.MaxRate)
	}
}

func TestRef(t *testing.T) {
	if Ref("ip") != "$m.ip" {
		t.Errorf("Expected '$m.ip', got '%s'", Ref("ip"))
	}

	if Ref("m.ip") != "$m.ip" {
		t.Errorf("Expected '$m.ip', got '%s'", Ref("m.ip"))
	}

	if Ref("energy") != "$m.energy" {
		t.Errorf("Expected '$m.energy', got '%s'", Ref("energy"))
	}
}

func TestComplexSchema(t *testing.T) {
	schema := NewSchema("security-alerts").
		Species("Event", "Raw events", nil).
		Species("Suspicion", "Suspicious stuff", nil).
		Species("Alert", "Alerts", nil).
		Reaction(NewReaction("login_failure_to_suspicion").
			Input("Event", WhereEq("type", "login_failed")).
			Rate(1.0).
			Effect(
				Consume(),
				Create("Suspicion").
					Payload("ip", Ref("ip")).
					Payload("kind", "login_failed").
					Energy(1.0).
					Stability(1.0),
			),
		).
		Reaction(NewReaction("suspicion_to_alert").
			Input("Suspicion").
			Rate(0.8).
			Effect(
				If(NewIfCount(NewCountMolecules("Suspicion").
					WhereEq("ip", Ref("ip")).
					Op("gte", 3))).
					Then(
						Create("Alert").
							Payload("ip", Ref("ip")).
							Payload("level", "high").
							Energy(5.0),
					),
			),
		)

	cfg := schema.Build()

	if cfg.Name != "security-alerts" {
		t.Errorf("Expected name 'security-alerts', got '%s'", cfg.Name)
	}

	if len(cfg.Species) != 3 {
		t.Errorf("Expected 3 species, got %d", len(cfg.Species))
	}

	if len(cfg.Reactions) != 2 {
		t.Errorf("Expected 2 reactions, got %d", len(cfg.Reactions))
	}

	// Verify first reaction
	r1 := cfg.Reactions[0]
	if r1.ID != "login_failure_to_suspicion" {
		t.Errorf("Expected reaction ID 'login_failure_to_suspicion', got '%s'", r1.ID)
	}

	if len(r1.Effects) != 2 {
		t.Errorf("Expected 2 effects in first reaction, got %d", len(r1.Effects))
	}

	// Verify second reaction has conditional effect
	r2 := cfg.Reactions[1]
	if len(r2.Effects) == 0 {
		t.Fatal("Expected at least one effect in second reaction")
	}

	if r2.Effects[0].If == nil {
		t.Error("Expected first effect of second reaction to be conditional")
	} else {
		if r2.Effects[0].If.CountMolecules == nil {
			t.Error("Expected conditional to use count_molecules")
		}
	}
}

func TestBuildSchemaFromClientConfig(t *testing.T) {
	schema := NewSchema("test").
		Species("Test", "Test species", nil).
		Reaction(NewReaction("test_reaction").
			Input("Test").
			Rate(1.0).
			Effect(Consume()))

	cfg := schema.Build()

	// Verify we can build a Schema from the config
	_, err := achem.BuildSchemaFromConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to build schema from config: %v", err)
	}
}

