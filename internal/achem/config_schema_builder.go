package achem

import (
	"fmt"
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
	effect := ReactionEffect{}

	for _, eff := range r.cfg.Effects {
		// if at least one effect says to consume a molecule, we will consume it
		if eff.Consume {
			effect.Consume = true
		}

		if eff.Update != nil {
			if effect.Update == nil {
				copy := m
				effect.Update = &copy
			}
			if eff.Update.EnergyAdd != nil {
				effect.Update.Energy += *eff.Update.EnergyAdd
				effect.Update.LastTouchedAt = ctx.EnvTime
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
