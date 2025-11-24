package achem

type ReactionContext struct {
	EnvTime int64
	Random  func() float64
}

type ReactionEffect struct {
	Consume        bool       // se true, la molecola di input viene rimossa
	Update         *Molecule  // versione aggiornata della molecola di input
	NewMolecules   []Molecule // nuove molecole da inserire
	AdditionalOps  []Operation // estendibile in futuro (es. log, metrics)
}

// per il PoC possiamo anche ignorare Operation e usare solo Consume/Update/NewMolecules.
type Operation struct {
	// placeholder
}

type EnvView interface {
	// Query semplice: restituisce tutte le molecole di una specie
	MoleculesBySpecies(species SpeciesName) []Molecule

	// Query più flessibile
	Find(filter func(Molecule) bool) []Molecule
}

type Reaction interface {
	ID() string
	Name() string

	// InputPattern: la reazione è interessata a questa molecola?
	InputPattern(m Molecule) bool

	// Rate: "intensità" di base (0..1). Poi verrà modificata da catalizzatori, ecc.
	Rate() float64

	// Apply: prova ad applicare la reazione a una molecola, dato il contesto.
	// Se non avviene nulla, può restituire un ReactionEffect vuoto.
	Apply(m Molecule, env EnvView, ctx ReactionContext) ReactionEffect
}
