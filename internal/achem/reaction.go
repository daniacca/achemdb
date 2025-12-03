package achem

// ReactionContext provides context information to reactions when they are applied.
// It includes the current environment time and a random number generator.
type ReactionContext struct {
	EnvTime int64
	Random  func() float64
}

// MoleculeChange represents an update to an existing molecule.
// If Updated is nil and the ID is in ConsumedIDs, the molecule is deleted.
type MoleculeChange struct {
	ID      MoleculeID
	Updated *Molecule // if nil + present in ConsumedIDs => delete
}

// ReactionEffect describes the changes that occur when a reaction fires.
// It can consume molecules, update existing ones, create new ones, and
// perform additional operations.
type ReactionEffect struct {
	ConsumedIDs    []MoleculeID 		// molecules to remove
	Changes        []MoleculeChange		// molecules to update
	NewMolecules   []Molecule   		// new molecules to insert
	AdditionalOps  []Operation  		// extendable in the future (e.g. log, metrics)
}

// Operation is a placeholder for future extensible operations
// that reactions can perform (e.g., logging, metrics).
type Operation struct {
	// To be done...
}

// EnvView provides a read-only view of the environment for reactions.
// Reactions use this interface to query molecules without modifying
// the environment state.
type EnvView interface {
	// Simple query: returns all molecules of a species
	MoleculesBySpecies(species SpeciesName) []Molecule

	// Flexible query
	Find(filter func(Molecule) bool) []Molecule
}

// Reaction defines the interface that all reactions must implement.
// Reactions determine which molecules they match, their firing rate,
// and what effects they produce when they fire.
type Reaction interface {
	ID() string
	Name() string

	// InputPattern: is the reaction interested in this molecule?
	InputPattern(m Molecule) bool

	// Rate: base intensity (0..1). It will be modified by catalysts, etc.
	Rate() float64

	// EffectiveRate: calculates the effective rate considering catalysts.
	// Returns the base rate modified by any matching catalysts in the environment.
	// NOTE: this method must NOT HAVE ANY SIDE EFFECTS. It should only return the 
	// effective rate, clamped between 0 and 1.
	EffectiveRate(m Molecule, env EnvView) float64

	// Apply: try to apply the reaction to a molecule, given the context.
	// If nothing happens, it can return an empty ReactionEffect.
	Apply(m Molecule, env EnvView, ctx ReactionContext) ReactionEffect
}
