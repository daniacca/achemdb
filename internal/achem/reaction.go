package achem

type ReactionContext struct {
	EnvTime int64
	Random  func() float64
}

type MoleculeChange struct {
	ID      MoleculeID
	Updated *Molecule // if nil + present in ConsumedIDs => delete
}

// The effect of a reactiion could consume one or more molecules,
// transform one or more molecules to other molecules, or create new molecules
type ReactionEffect struct {
	ConsumedIDs    []MoleculeID 		// molecules to remove
	Changes        []MoleculeChange		// molecules to update
	NewMolecules   []Molecule   		// new molecules to insert
	AdditionalOps  []Operation  		// extendable in the future (e.g. log, metrics)
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

	// EffectiveRate: calculates the effective rate considering catalysts.
	// Returns the base rate modified by any matching catalysts in the environment.
	EffectiveRate(m Molecule, env EnvView) float64

	// Apply: try to apply the reaction to a molecule, given the context.
	// If nothing happens, it can return an empty ReactionEffect.
	Apply(m Molecule, env EnvView, ctx ReactionContext) ReactionEffect
}
