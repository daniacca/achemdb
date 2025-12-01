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
}

func NewEnvironment(schema *Schema) *Environment {
	return &Environment{
		schema: schema,
		mols:   make(map[MoleculeID]Molecule),
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
		time:   0,
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

			if ctx.Random() > r.Rate() {
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
