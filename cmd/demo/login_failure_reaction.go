package main

import "github.com/daniacca/achemdb/internal/achem"

// ------------------------------
// 1) LoginFailureToSuspicionReaction
// ------------------------------

type LoginFailureToSuspicionReaction struct {
	baseRate float64
}

func NewLoginFailureToSuspicionReaction() achem.Reaction {
	return &LoginFailureToSuspicionReaction{
		baseRate: 1.0, // for the PoC: always
	}
}

func (r *LoginFailureToSuspicionReaction) ID() string   { return "login_failure_to_suspicion" }
func (r *LoginFailureToSuspicionReaction) Name() string { return "Promote login failures to suspicion" }
func (r *LoginFailureToSuspicionReaction) Rate() float64 {
	return r.baseRate
}

func (r *LoginFailureToSuspicionReaction) InputPattern(m achem.Molecule) bool {
	if m.Species != achem.SpeciesName("Event") {
		return false
	}
	t, ok := m.Payload["type"].(string)
	return ok && t == "login_failed"
}

func (r *LoginFailureToSuspicionReaction) Apply(m achem.Molecule, env achem.EnvView, ctx achem.ReactionContext) achem.ReactionEffect {
	ip, _ := m.Payload["ip"].(string)

	susp := achem.NewMolecule(achem.SpeciesName("Suspicion"), map[string]any{
		"ip":   ip,
		"kind": "login_failed",
	}, ctx.EnvTime)
	susp.Energy = 1.0
	susp.Stability = 1.0

	return achem.ReactionEffect{
		Consume:      true, // consumiamo l'Event
		Update:       nil,
		NewMolecules: []achem.Molecule{susp},
	}
}