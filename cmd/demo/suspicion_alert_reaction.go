package main

import "github.com/daniacca/achemdb/internal/achem"

type SuspicionToAlertReaction struct {
	baseRate       float64
	thresholdCount int
}

func NewSuspicionToAlertReaction() achem.Reaction {
	return &SuspicionToAlertReaction{
		baseRate:       0.8, // high probability
		thresholdCount: 3,   // 3 suspicions per IP
	}
}

func (r *SuspicionToAlertReaction) ID() string   { return "suspicion_to_alert" }
func (r *SuspicionToAlertReaction) Name() string { return "Promote suspicions to alerts" }
func (r *SuspicionToAlertReaction) Rate() float64 {
	return r.baseRate
}

func (r *SuspicionToAlertReaction) InputPattern(m achem.Molecule) bool {
	return m.Species == achem.SpeciesName("Suspicion")
}

func (r *SuspicionToAlertReaction) Apply(m achem.Molecule, env achem.EnvView, ctx achem.ReactionContext) achem.ReactionEffect {
	ip, _ := m.Payload["ip"].(string)
	if ip == "" {
		return achem.ReactionEffect{}
	}

	// How many Suspicion for this IP?
	susps := env.Find(func(x achem.Molecule) bool {
		if x.Species != achem.SpeciesName("Suspicion") {
			return false
		}
		xip, _ := x.Payload["ip"].(string)
		return xip == ip
	})

	if len(susps) < r.thresholdCount {
		return achem.ReactionEffect{}
	}

	// Does an Alert already exist for this IP?
	alerts := env.Find(func(x achem.Molecule) bool {
		if x.Species != achem.SpeciesName("Alert") {
			return false
		}
		xip, _ := x.Payload["ip"].(string)
		return xip == ip
	})
	if len(alerts) > 0 {
		return achem.ReactionEffect{}
	}

	alert := achem.NewMolecule(achem.SpeciesName("Alert"), map[string]any{
		"ip":    ip,
		"level": "high",
	}, ctx.EnvTime)
	alert.Energy = 5.0
	alert.Stability = 2.0

	updated := m
	updated.Energy += 0.5
	updated.LastTouchedAt = ctx.EnvTime

	return achem.ReactionEffect{
		ConsumedIDs: []achem.MoleculeID{},
		Changes: []achem.MoleculeChange{
			{ID: m.ID, Updated: &updated},
		},
		NewMolecules: []achem.Molecule{
			alert,
		},
	}
}
