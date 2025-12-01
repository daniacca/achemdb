package achem

import (
	"math/rand"
	"sync"
	"time"
)

type Environment struct {
	mu      sync.RWMutex
	schema  *Schema
	time    int64
	mols    map[MoleculeID]Molecule
	rand    *rand.Rand
	stopCh chan struct{}
	isRunning bool
}

func NewEnvironment(schema *Schema) *Environment {
	return &Environment{
		schema: schema,
		mols:   make(map[MoleculeID]Molecule),
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
		time:   0,
		stopCh: make(chan struct{}),
		isRunning: false,
	}
}

// envView is a private adapter that exposes read-only methods
type envView struct {
	molecules []Molecule
}

func (v envView) MoleculesBySpecies(species SpeciesName) []Molecule {
	out := make([]Molecule, 0)
	for _, m := range v.molecules {
		if m.Species == species {
			out = append(out, m)
		}
	}
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
	view := envView{molecules: snapshot}

	ctx := ReactionContext{
		EnvTime: e.time,
		Random:  e.rand.Float64,
	}

	// 1) collect all effects
	consumed := make(map[MoleculeID]struct{})
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
