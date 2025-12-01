package achem

type ReactionContext struct {
	EnvTime int64
	Random  func() float64
}

type ReactionEffect struct {
	Consume        bool         // if true, the input molecule will be removed
	Update         *Molecule    // updated version of the input molecule
	NewMolecules   []Molecule   // new molecules to insert
	AdditionalOps  []Operation  // extendable in the future (e.g. log, metrics)
}

// Actually a placeholder for future operations
type Operation struct {
	// To be done...
}

type EnvView interface {
	// Simple query: returns all molecules of a species
	MoleculesBySpecies(species SpeciesName) []Molecule

	// Flexible query
	Find(filter func(Molecule) bool) []Molecule
}

type Reaction interface {
	ID() string
	Name() string

	// InputPattern: is the reaction interested in this molecule?
	InputPattern(m Molecule) bool

	// Rate: base intensity (0..1). It will be modified by catalysts, etc.
	Rate() float64

	// Apply: try to apply the reaction to a molecule, given the context.
	// If nothing happens, it can return an empty ReactionEffect.
	Apply(m Molecule, env EnvView, ctx ReactionContext) ReactionEffect
}
