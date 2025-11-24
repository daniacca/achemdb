package achem

type Schema struct {
	Name      string
	species   map[SpeciesName]Species
	reactions []Reaction
}

func NewSchema(name string) *Schema {
	return &Schema{
		Name:      name,
		species:   make(map[SpeciesName]Species),
		reactions: make([]Reaction, 0),
	}
}

func (s *Schema) WithSpecies(species ...Species) *Schema {
	for _, sp := range species {
		s.species[sp.Name] = sp
	}
	return s
}

func (s *Schema) WithReactions(reactions ...Reaction) *Schema {
	s.reactions = append(s.reactions, reactions...)
	return s
}

func (s *Schema) Species(name SpeciesName) (Species, bool) {
	sp, ok := s.species[name]
	return sp, ok
}

func (s *Schema) Reactions() []Reaction {
	return s.reactions
}
