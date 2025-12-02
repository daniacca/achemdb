package achem

import (
	"context"
	"math/rand"
	"sync"
	"time"
)

type Environment struct {
	mu          sync.RWMutex
	schema      *Schema
	time        int64
	mols        map[MoleculeID]Molecule
	rand        *rand.Rand
	stopCh      chan struct{}
	isRunning   bool
	envID       EnvironmentID
	notifierMgr *NotificationManager
}

func NewEnvironment(schema *Schema) *Environment {
	return &Environment{
		schema:      schema,
		mols:        make(map[MoleculeID]Molecule),
		rand:        rand.New(rand.NewSource(time.Now().UnixNano())),
		time:        0,
		stopCh:      make(chan struct{}),
		isRunning:   false,
		notifierMgr: NewNotificationManager(),
	}
}

// SetEnvironmentID sets the environment ID (used for notifications)
func (e *Environment) SetEnvironmentID(id EnvironmentID) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.envID = id
}

// SetNotificationManager sets a custom notification manager
func (e *Environment) SetNotificationManager(mgr *NotificationManager) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.notifierMgr = mgr
}

// GetNotificationManager returns the notification manager
func (e *Environment) GetNotificationManager() *NotificationManager {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.notifierMgr
}

// envView is a private adapter that exposes read-only methods
type envView struct {
	molecules []Molecule
	bySpecies map[SpeciesName][]Molecule
}

func (v envView) MoleculesBySpecies(species SpeciesName) []Molecule {
	if v.bySpecies == nil {
		// fallback for safety (should not happen if Step sets it)
		out := make([]Molecule, 0)
		for _, m := range v.molecules {
			if m.Species == species {
				out = append(out, m)
			}
		}
		return out
	}

	mols, ok := v.bySpecies[species]
	if !ok {
		return nil
	}

	// return a copy to keep immutability guarantees
	out := make([]Molecule, len(mols))
	copy(out, mols)
	return out
}

func (v envView) Find(filter func(Molecule) bool) []Molecule {
	out := make([]Molecule, 0)
	for _, m := range v.molecules {
		if filter(m) {
			out = append(out, m)
		}
	}
	return out
}

func (e *Environment) now() int64 {
	return e.time
}

func (e *Environment) Insert(m Molecule) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if m.ID == "" {
		m.ID = MoleculeID(NewRandomID())
	}
	if m.CreatedAt == 0 {
		m.CreatedAt = e.now()
		m.LastTouchedAt = e.now()
	}
	e.mols[m.ID] = m
}

func (e *Environment) AllMolecules() []Molecule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]Molecule, 0, len(e.mols))
	for _, m := range e.mols {
		out = append(out, m)
	}
	return out
}

// a single step inside the environment, it will apply all reactions to all the molecules
// collected in the snapshot. 
func (e *Environment) Step() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.time++

	// snapshot
	snapshot := make([]Molecule, 0, len(e.mols))
	for _, m := range e.mols {
		snapshot = append(snapshot, m)
	}

	// build per-species index for fast lookup
	bySpecies := make(map[SpeciesName][]Molecule)
	for _, m := range snapshot {
		bySpecies[m.Species] = append(bySpecies[m.Species], m)
	}

	view := envView{
		molecules: snapshot,
		bySpecies: bySpecies,
	}

	ctx := ReactionContext{
		EnvTime: e.time,
		Random:  e.rand.Float64,
	}

	// 1) collect all effects
	consumed := make(map[MoleculeID]struct{})
	consumedMolecules := make(map[MoleculeID]Molecule) // Store molecules before deletion for notifications
	changes := make(map[MoleculeID]Molecule)
	newMolecules := make([]Molecule, 0)

	for _, m := range snapshot {
		if _, ok := consumed[m.ID]; ok {
			continue
		}

		for _, r := range e.schema.Reactions() {
			if !r.InputPattern(m) {
				continue
			}

			// Use effective rate (base rate + catalyst effects)
			effectiveRate := r.EffectiveRate(m, view)
			if ctx.Random() > effectiveRate {
				continue
			}

			eff := r.Apply(m, view, ctx)

			// Check if reaction produced any effects (non-empty effect)
			hasEffects := len(eff.ConsumedIDs) > 0 || len(eff.Changes) > 0 || len(eff.NewMolecules) > 0

			// Store consumed molecules before marking them for deletion (for notifications)
			for _, id := range eff.ConsumedIDs {
				if mol, exists := e.mols[id]; exists {
					consumedMolecules[id] = mol
				}
			}

			// Send notification if reaction fired and has effects
			if hasEffects {
				e.sendNotification(r, m, view, eff, ctx, consumedMolecules)
			}

			// mark consumed
			for _, id := range eff.ConsumedIDs {
				consumed[id] = struct{}{}
			}

			// apply changes (last-wins)
			for _, ch := range eff.Changes {
				if ch.Updated != nil {
					changes[ch.ID] = *ch.Updated
				}
			}

			newMolecules = append(newMolecules, eff.NewMolecules...)
		}
	}

	// 2) Apply changes to the environment

	// 2.1 - remove consumed molecules
	for id := range consumed {
		delete(e.mols, id)
	}

	// 2.2 - apply changes
	for id, m := range changes {
		if _, removed := consumed[id]; removed {
			continue
		}
		e.mols[id] = m
	}

	// 2.3 - insert new molecules
	for _, nm := range newMolecules {
		if nm.ID == "" {
			nm.ID = MoleculeID(NewRandomID())
		}
		if nm.CreatedAt == 0 {
			nm.CreatedAt = e.time
			nm.LastTouchedAt = e.time
		}
		e.mols[nm.ID] = nm
	}
}

// Run will start the environment in a goroutine, starting it's own ticker that will
// run until the stop channel is closed. It can be called multiple times to restart
// after stopping.
func (e *Environment) Run(interval time.Duration) {
	e.mu.Lock()
	if e.isRunning {
		e.mu.Unlock()
		return
	}
	// Create a new stop channel for this run (allows restart after stop)
	e.stopCh = make(chan struct{})
	e.isRunning = true
	e.mu.Unlock()

	// Run in a goroutine so it doesn't block the caller
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				e.Step()
			case <-e.stopCh:
				e.mu.Lock()
				e.isRunning = false
				e.mu.Unlock()
				return
			}
		}
	}()
}

// Stop will stop the environment by closing the stop channel.
// After stopping, Run() can be called again to restart.
func (e *Environment) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.isRunning {
		return
	}
	
	// Close the channel to signal stop
	// The goroutine will detect this and set isRunning to false
	close(e.stopCh)
}

// sendNotification sends a notification if the reaction is configured to do so
func (e *Environment) sendNotification(r Reaction, m Molecule, view EnvView, eff ReactionEffect, ctx ReactionContext, consumedMolecules map[MoleculeID]Molecule) {
	// Get notification config from reaction if it's a ConfigReaction
	notifyCfg := e.getNotificationConfig(r)
	if notifyCfg == nil || !notifyCfg.Enabled {
		return
	}

	if len(notifyCfg.Notifiers) == 0 {
		return
	}

	// Find partners if this was a partner-based reaction
	partners := e.findPartnersForNotification(r, m, view)

	// Collect consumed molecules for the notification
	consumed := make([]Molecule, 0, len(eff.ConsumedIDs))
	for _, id := range eff.ConsumedIDs {
		if mol, exists := consumedMolecules[id]; exists {
			consumed = append(consumed, mol)
		}
	}

	// Create notification event
	event := CreateNotificationEventWithConsumed(
		e.envID,
		r,
		m,
		partners,
		eff,
		consumed,
		ctx.EnvTime,
	)

	// Send notification asynchronously to avoid blocking the step
	go func() {
		notifyCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = e.notifierMgr.Notify(notifyCtx, event, notifyCfg.Notifiers)
	}()
}

// getNotificationConfig extracts notification config from a reaction
func (e *Environment) getNotificationConfig(r Reaction) *NotificationConfig {
	// Check if it's a ConfigReaction
	if cr, ok := r.(*ConfigReaction); ok {
		if cr.cfg.Notify != nil {
			return cr.cfg.Notify
		}
	}
	return nil
}

// findPartnersForNotification finds partners that were used in the reaction
func (e *Environment) findPartnersForNotification(r Reaction, m Molecule, view EnvView) []Molecule {
	if cr, ok := r.(*ConfigReaction); ok {
		partners := make([]Molecule, 0)
		for _, partnerCfg := range cr.cfg.Input.Partners {
			// Use the findPartners function from config_schema_builder
			foundPartners := findPartnersForReaction(partnerCfg, m, view)
			partners = append(partners, foundPartners...)
		}
		return partners
	}
	return nil
}

// findPartnersForReaction is a wrapper to access the private findPartners function
// We need to make findPartners accessible or create a public wrapper
func findPartnersForReaction(partnerCfg PartnerConfig, m Molecule, env EnvView) []Molecule {
	// Get all molecules of the specified species
	candidates := env.MoleculesBySpecies(SpeciesName(partnerCfg.Species))

	var matches []Molecule
	for _, candidate := range candidates {
		// Skip the molecule itself
		if candidate.ID == m.ID {
			continue
		}

		// Check where conditions using shared helper
		if matchWhere(partnerCfg.Where, candidate, m) {
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
