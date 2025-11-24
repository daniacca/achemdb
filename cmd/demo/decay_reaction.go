package main

import "github.com/daniacca/achemdb/internal/achem"

type DecayReaction struct {
	baseRate     float64
	decayPerTick float64
}

func NewDecayReaction() achem.Reaction {
	return &DecayReaction{
		baseRate:     1.0,
		decayPerTick: 0.1,
	}
}

func (r *DecayReaction) ID() string   { return "decay" }
func (r *DecayReaction) Name() string { return "Natural decay of Suspicion/Alert" }
func (r *DecayReaction) Rate() float64 {
	return r.baseRate
}

func (r *DecayReaction) InputPattern(m achem.Molecule) bool {
	return m.Species == achem.SpeciesName("Suspicion") ||
		m.Species == achem.SpeciesName("Alert")
}

func (r *DecayReaction) Apply(m achem.Molecule, env achem.EnvView, ctx achem.ReactionContext) achem.ReactionEffect {
	updated := m
	updated.Energy -= r.decayPerTick
	updated.LastTouchedAt = ctx.EnvTime

	if updated.Energy <= 0 {
		return achem.ReactionEffect{
			Consume: true,
		}
	}

	return achem.ReactionEffect{
		Consume: false,
		Update:  &updated,
	}
}
