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
		if v, ok := m.Payload[field]; ok {
			return v
		}
	}
	return val
}

// Apply will apply the effects of the reaction to the molecule
func (r *ConfigReaction) Apply(m Molecule, env EnvView, ctx ReactionContext) ReactionEffect {
	effect := ReactionEffect{
		ConsumedIDs:   []MoleculeID{},
		Changes:       []MoleculeChange{},
		NewMolecules:  []Molecule{},
	}

	for _, eff := range r.cfg.Effects {
		// if at least one effect says to consume a molecule, we will consume it
		if eff.Consume {
			// Add the molecule ID to ConsumedIDs if not already present
			found := slices.Contains(effect.ConsumedIDs, m.ID)
			if !found {
				effect.ConsumedIDs = append(effect.ConsumedIDs, m.ID)
			}
		}

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

		if eff.Create != nil {
			nm := NewMolecule(
				SpeciesName(eff.Create.Species),
				map[string]any{},
				ctx.EnvTime,
			)

			// copy payload to the new molecule
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

	return effect
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
