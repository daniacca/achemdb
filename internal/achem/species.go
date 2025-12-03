package achem

// SpeciesName is the name/identifier of a species type.
type SpeciesName string

// Species represents a type of molecule in the artificial chemistry system.
// Each species has a name, description, and optional metadata.
type Species struct {
	Name        SpeciesName
	Description string
	Meta map[string]any
}
