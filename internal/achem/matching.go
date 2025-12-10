package achem

import "fmt"

// indexKeyFromValue converts a value to a string key for indexing.
// This is a simple approach; we accept that different types with the same
// string representation might collide (it's an optimization, not a strict semantic guarantee).
func indexKeyFromValue(v any) string {
	return fmt.Sprintf("%v", v)
}

// resolveValueRef resolves $m.* references into values from the origin molecule.
// It supports:
//   - $m.energy
//   - $m.stability
//   - $m.id
//   - $m.species
//   - $m.created_at / $m.createdAt / $m.CreatedAt
//   - $m.last_touched_at / $m.lastTouchedAt / $m.LastTouchedAt
//   - $m.<payloadField>
// Any non-string value is returned as-is.
func resolveValueRef(val any, origin Molecule) any {
	s, ok := val.(string)
	if !ok {
		return val
	}
	if len(s) > 3 && s[:3] == "$m." {
		field := s[3:]
		// Check if it's a molecule field (energy, stability, etc.)
		switch field {
		case "energy":
			return origin.Energy
		case "stability":
			return origin.Stability
		case "id":
			return string(origin.ID)
		case "species":
			return string(origin.Species)
		case "created_at", "createdAt", "CreatedAt":
			return origin.CreatedAt
		case "last_touched_at", "lastTouchedAt", "LastTouchedAt":
			return origin.LastTouchedAt
		default:
			// Otherwise, check payload
			if v, ok := origin.Payload[field]; ok {
				return v
			}
		}
	}
	return val
}

// matchWhere checks if a candidate molecule matches the WhereConfig conditions.
// The origin molecule is used for resolving $m.* references in the conditions.
// Returns true only if all conditions match.
func matchWhere(where WhereConfig, candidate Molecule, origin Molecule) bool {
	for field, cond := range where {
		// Resolve the condition value (might be "$m.field")
		condValue := resolveValueRef(cond.Eq, origin)
		
		candidateValue, ok := candidate.Payload[field]
		if !ok || candidateValue != condValue {
			return false
		}
	}
	return true
}

// filterBySpeciesAndWhere returns molecules of a given species that match the given where,
// using indexes when possible, and falling back to a linear scan otherwise.
func filterBySpeciesAndWhere(env EnvView, species SpeciesName, where WhereConfig, origin Molecule) []Molecule {
	// If where is empty, just return by species
	if len(where) == 0 {
		return env.MoleculesBySpecies(species)
	}

	// Try to use index only for the simple case:
	// - underlying env is our concrete envView
	// - where has exactly one field
	if v, ok := env.(envView); ok && v.bySpeciesFieldValue != nil && len(where) == 1 {
		for field, cond := range where {
			// resolve the comparison value (may involve $m.*)
			targetValue := resolveValueRef(cond.Eq, origin)
			key := indexKeyFromValue(targetValue)

			if fieldMap, ok := v.bySpeciesFieldValue[species]; ok {
				if mols, ok := fieldMap[field]; ok {
					if indexed, ok := mols[key]; ok {
						// Return a copy to avoid external modification
						out := make([]Molecule, len(indexed))
						copy(out, indexed)
						return out
					}
				}
			}
		}
		// If we don't find indexed matches, we'll fall back to the generic path below
	}

	// Fallback: linear scan by species + where
	candidates := env.MoleculesBySpecies(species)
	out := make([]Molecule, 0, len(candidates))
	for _, candidate := range candidates {
		if matchWhere(where, candidate, origin) {
			out = append(out, candidate)
		}
	}
	return out
}

