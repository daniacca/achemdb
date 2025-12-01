package achem

import (
	"fmt"
	"slices"
)

// ConfigReaction will be used to build a Reaction from a ReactionConfig
type ConfigReaction struct {
	cfg ReactionConfig
}

func (r *ConfigReaction) ID() string   { return r.cfg.ID }
func (r *ConfigReaction) Name() string { return r.cfg.Name }
func (r *ConfigReaction) Rate() float64 {
	if r.cfg.Rate <= 0 {
		return 1.0
	}
	return r.cfg.Rate
}

// match species + where.eq on payload
// Note: Partner matching is done in Apply, not here, for performance reasons
func (r *ConfigReaction) InputPattern(m Molecule) bool {
	if string(m.Species) != r.cfg.Input.Species {
		return false
	}

	if len(r.cfg.Input.Where) == 0 {
		return true
	}

	for field, cond := range r.cfg.Input.Where {
		val, ok := m.Payload[field]
		if !ok {
			return false
		}
		if cond.Eq != nil && val != cond.Eq {
			return false
		}
	}

	return true
}

// inner helper function to resolve values from molecules
func resolveValueFromMolecule(val any, m Molecule) any {
	s, ok := val.(string)
	if !ok {
		return val
	}
	if len(s) > 3 && s[:3] == "$m." {
		field := s[3:]
		// Check if it's a molecule field (energy, stability, etc.)
		switch field {
		case "energy":
			return m.Energy
		case "stability":
			return m.Stability
		case "id":
			return string(m.ID)
		case "species":
			return string(m.Species)
		default:
			// Otherwise, check payload
			if v, ok := m.Payload[field]; ok {
				return v
			}
		}
	}
	return val
}

// getFieldValue retrieves a field value from a molecule
// Supports "energy", "stability", "id", "species", or payload fields
func getFieldValue(field string, m Molecule) (any, bool) {
	switch field {
	case "energy":
		return m.Energy, true
	case "stability":
		return m.Stability, true
	case "id":
		return string(m.ID), true
	case "species":
		return string(m.Species), true
	default:
		// Check if it's a payload field reference like "$m.field"
		if len(field) > 3 && field[:3] == "$m." {
			payloadField := field[3:]
			if v, ok := m.Payload[payloadField]; ok {
				return v, true
			}
			return nil, false
		}
		// Direct payload field
		if v, ok := m.Payload[field]; ok {
			return v, true
		}
		return nil, false
	}
}

// compareValues compares two values using the specified operator
func compareValues(left, right any, op string) bool {
	// Handle nil cases
	if left == nil && right == nil {
		return op == "eq"
	}
	if left == nil || right == nil {
		return op == "ne"
	}

	// Try numeric comparison first
	leftFloat, leftIsFloat := toFloat64(left)
	rightFloat, rightIsFloat := toFloat64(right)
	
	if leftIsFloat && rightIsFloat {
		switch op {
		case "eq":
			return leftFloat == rightFloat
		case "ne":
			return leftFloat != rightFloat
		case "gt":
			return leftFloat > rightFloat
		case "gte":
			return leftFloat >= rightFloat
		case "lt":
			return leftFloat < rightFloat
		case "lte":
			return leftFloat <= rightFloat
		}
	}

	// Fall back to string comparison
	leftStr := fmt.Sprintf("%v", left)
	rightStr := fmt.Sprintf("%v", right)
	
	switch op {
	case "eq":
		return leftStr == rightStr
	case "ne":
		return leftStr != rightStr
	case "gt":
		return leftStr > rightStr
	case "gte":
		return leftStr >= rightStr
	case "lt":
		return leftStr < rightStr
	case "lte":
		return leftStr <= rightStr
	}
	
	return false
}

// toFloat64 attempts to convert a value to float64
func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case int32:
		return float64(val), true
	case uint:
		return float64(val), true
	case uint64:
		return float64(val), true
	case uint32:
		return float64(val), true
	default:
		return 0, false
	}
}

// evaluateIfCondition evaluates an IfConditionConfig and returns true if condition is met
func evaluateIfCondition(cond *IfConditionConfig, m Molecule, env EnvView) bool {
	if cond == nil {
		return false
	}

	// Check if it's a count_molecules condition
	if cond.CountMolecules != nil {
		return evaluateCountMolecules(cond.CountMolecules, m, env)
	}

	// Otherwise, it's a field condition
	if cond.Field == "" || cond.Op == "" {
		return false
	}

	fieldValue, ok := getFieldValue(cond.Field, m)
	if !ok {
		return false
	}

	// Resolve the comparison value (might be a reference like "$m.ip")
	compareValue := resolveValueFromMolecule(cond.Value, m)

	return compareValues(fieldValue, compareValue, cond.Op)
}

// evaluateCountMolecules evaluates a count_molecules aggregation
func evaluateCountMolecules(cfg *CountMoleculesConfig, m Molecule, env EnvView) bool {
	// Get all molecules of the specified species
	candidates := env.MoleculesBySpecies(SpeciesName(cfg.Species))

	// Filter by where conditions
	var matches []Molecule
	for _, candidate := range candidates {
		// Skip the molecule itself
		if candidate.ID == m.ID {
			continue
		}

		// Check where conditions
		matchesWhere := true
		for field, cond := range cfg.Where {
			// Resolve the condition value (might be "$m.field")
			condValue := resolveValueFromMolecule(cond.Eq, m)
			
			candidateValue, ok := candidate.Payload[field]
			if !ok || candidateValue != condValue {
				matchesWhere = false
				break
			}
		}

		if matchesWhere {
			matches = append(matches, candidate)
		}
	}

	count := len(matches)

	// Evaluate the operator
	for op, opValue := range cfg.Op {
		opValueFloat, ok := toFloat64(opValue)
		if !ok {
			opValueFloat = float64(count) // fallback
		}
		return compareValues(float64(count), opValueFloat, op)
	}

	return false
}

// findPartners finds partner molecules matching the partner config
func findPartners(partnerCfg PartnerConfig, m Molecule, env EnvView) []Molecule {
	// Get all molecules of the specified species
	candidates := env.MoleculesBySpecies(SpeciesName(partnerCfg.Species))

	var matches []Molecule
	for _, candidate := range candidates {
		// Skip the molecule itself
		if candidate.ID == m.ID {
			continue
		}

		// Check where conditions
		matchesWhere := true
		for field, cond := range partnerCfg.Where {
			// Resolve the condition value (might be "$m.field")
			condValue := resolveValueFromMolecule(cond.Eq, m)
			
			candidateValue, ok := candidate.Payload[field]
			if !ok || candidateValue != condValue {
				matchesWhere = false
				break
			}
		}

		if matchesWhere {
			matches = append(matches, candidate)
		}
	}

	// Return up to the required count
	count := partnerCfg.Count
	if count <= 0 {
		count = 1 // default
	}
	if len(matches) > count {
		return matches[:count]
	}
	return matches
}

// Apply will apply the effects of the reaction to the molecule
func (r *ConfigReaction) Apply(m Molecule, env EnvView, ctx ReactionContext) ReactionEffect {
	effect := ReactionEffect{
		ConsumedIDs:   []MoleculeID{},
		Changes:       []MoleculeChange{},
		NewMolecules:  []Molecule{},
	}

	// Check for partners if required
	partners := make([]Molecule, 0)
	if len(r.cfg.Input.Partners) > 0 {
		for _, partnerCfg := range r.cfg.Input.Partners {
			requiredCount := partnerCfg.Count
			if requiredCount <= 0 {
				requiredCount = 1 // default to 1 if not specified
			}
			
			foundPartners := findPartners(partnerCfg, m, env)
			if len(foundPartners) < requiredCount {
				// Not enough partners found, return empty effect
				return effect
			}
			partners = append(partners, foundPartners...)
		}
	}

	// Apply effects
	r.applyEffects(r.cfg.Effects, m, partners, env, ctx, &effect)

	return effect
}

// applyEffects recursively applies effects, handling conditional logic
func (r *ConfigReaction) applyEffects(effects []EffectConfig, m Molecule, partners []Molecule, env EnvView, ctx ReactionContext, effect *ReactionEffect) {
	for _, eff := range effects {
		// Handle conditional effects
		if eff.If != nil {
			conditionMet := evaluateIfCondition(eff.If, m, env)
			if conditionMet {
				// Apply "then" effects
				if len(eff.Then) > 0 {
					r.applyEffects(eff.Then, m, partners, env, ctx, effect)
				}
			} else {
				// Apply "else" effects
				if len(eff.Else) > 0 {
					r.applyEffects(eff.Else, m, partners, env, ctx, effect)
				}
			}
			// Skip other effects in this config if it's conditional
			continue
		}

		// Apply consume effect
		if eff.Consume {
			// Add the molecule ID to ConsumedIDs if not already present
			found := slices.Contains(effect.ConsumedIDs, m.ID)
			if !found {
				effect.ConsumedIDs = append(effect.ConsumedIDs, m.ID)
			}
		}

		// Apply update effect
		if eff.Update != nil {
			// Find existing change for this molecule, or create a new one
			var change *MoleculeChange
			for i := range effect.Changes {
				if effect.Changes[i].ID == m.ID {
					change = &effect.Changes[i]
					break
				}
			}
			
			if change == nil {
				// Create a copy of the molecule for the change
				copy := m
				effect.Changes = append(effect.Changes, MoleculeChange{
					ID:      m.ID,
					Updated: &copy,
				})
				change = &effect.Changes[len(effect.Changes)-1]
			}

			if eff.Update.EnergyAdd != nil && change.Updated != nil {
				change.Updated.Energy += *eff.Update.EnergyAdd
				change.Updated.LastTouchedAt = ctx.EnvTime
			}
		}

		// Apply create effect
		if eff.Create != nil {
			nm := NewMolecule(
				SpeciesName(eff.Create.Species),
				map[string]any{},
				ctx.EnvTime,
			)

			// copy payload to the new molecule, resolving references
			for k, v := range eff.Create.Payload {
				nm.Payload[k] = resolveValueFromMolecule(v, m)
			}

			if eff.Create.Energy != nil {
				nm.Energy = *eff.Create.Energy
			}

			if eff.Create.Stability != nil {
				nm.Stability = *eff.Create.Stability
			}

			effect.NewMolecules = append(effect.NewMolecules, nm)
		}
	}
}

// BuildSchemaFromConfig converts a SchemaConfig to a Schema
func BuildSchemaFromConfig(cfg SchemaConfig) (*Schema, error) {
	if cfg.Name == "" {
		return nil, fmt.Errorf("schema name is required")
	}

	s := NewSchema(cfg.Name)

	// Species
	for _, sp := range cfg.Species {
		if sp.Name == "" {
			return nil, fmt.Errorf("species name is required")
		}
		s = s.WithSpecies(Species{
			Name:        SpeciesName(sp.Name),
			Description: sp.Description,
			Meta:        sp.Meta,
		})
	}

	// Reactions
	for _, rc := range cfg.Reactions {
		if rc.ID == "" {
			return nil, fmt.Errorf("reaction id is required")
		}
		cr := &ConfigReaction{cfg: rc}
		s = s.WithReactions(cr)
	}

	return s, nil
}
