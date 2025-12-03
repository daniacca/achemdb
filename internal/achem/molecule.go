package achem

// MoleculeID is a unique identifier for a molecule.
type MoleculeID string

// Molecule represents a data entity in the artificial chemistry system.
// Molecules have a species, payload data, energy, stability, and timestamps.
type Molecule struct {
	ID            MoleculeID
	Species       SpeciesName
	Payload       map[string]any
	Energy        float64
	Stability     float64
	Tags          []string
	CreatedAt     int64
	LastTouchedAt int64
}

// NewMolecule creates a new molecule with the specified species and payload.
// The molecule is assigned a random ID and initialized with default energy
// and stability values of 1.0. The time parameter sets both CreatedAt and
// LastTouchedAt timestamps.
func NewMolecule(species SpeciesName, payload map[string]any, time int64) Molecule {
	return Molecule{
		ID:            MoleculeID(NewRandomID()),
		Species:       species,
		Payload:       payload,
		Energy:        1.0,
		Stability:     1.0,
		Tags:          nil,
		CreatedAt:     time,
		LastTouchedAt: time,
	}
}
