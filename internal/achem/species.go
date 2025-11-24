package achem

type SpeciesName string

type Species struct {
	Name        SpeciesName
	Description string
	Meta map[string]any
}
