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

	// snapshot delle molecole per passarlo alla EnvView
	snapshot := make([]Molecule, 0, len(e.mols))
	for _, m := range e.mols {
		snapshot = append(snapshot, m)
	}
	view := envView{molecules: snapshot}

	ctx := ReactionContext{
		EnvTime: e.time,
		Random:  e.rand.Float64,
	}

	// Applichiamo reazioni molecola per molecola
	for id, m := range e.mols {
		for _, r := range e.schema.Reactions() {
			if !r.InputPattern(m) {
				continue
			}

			// probabilità che la reazione "scatti"
			if ctx.Random() > r.Rate() {
				continue
			}

			effect := r.Apply(m, view, ctx)

			if effect.Consume {
				delete(e.mols, id)
			} else if effect.Update != nil {
				e.mols[id] = *effect.Update
				m = *effect.Update
			}

			for _, nm := range effect.NewMolecules {
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
	}
}
