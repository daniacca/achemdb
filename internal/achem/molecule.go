package achem

type MoleculeID string

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
