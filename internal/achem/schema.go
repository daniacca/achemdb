package achem

// Schema defines the structure of an artificial chemistry system.
// It contains species definitions and reaction rules that govern
// how molecules interact.
type Schema struct {
	Name      string
	species   map[SpeciesName]Species
	reactions []Reaction
}

// NewSchema creates a new schema with the given name.
// The schema starts with no species or reactions.
func NewSchema(name string) *Schema {
	return &Schema{
		Name:      name,
		species:   make(map[SpeciesName]Species),
		reactions: make([]Reaction, 0),
	}
}

// WithSpecies adds species definitions to the schema and returns the schema
// for method chaining.
func (s *Schema) WithSpecies(species ...Species) *Schema {
	for _, sp := range species {
		s.species[sp.Name] = sp
	}
	return s
}

// WithReactions adds reaction definitions to the schema and returns the schema
// for method chaining.
func (s *Schema) WithReactions(reactions ...Reaction) *Schema {
	s.reactions = append(s.reactions, reactions...)
	return s
}

// Species retrieves a species definition by name.
// Returns the species and a boolean indicating if it was found.
func (s *Schema) Species(name SpeciesName) (Species, bool) {
	sp, ok := s.species[name]
	return sp, ok
}

// Reactions returns all reaction definitions in the schema.
func (s *Schema) Reactions() []Reaction {
	return s.reactions
}
