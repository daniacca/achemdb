package achem

// resolveValueRef resolves $m.* references into values from the origin molecule.
// It supports:
//   - $m.energy
//   - $m.stability
//   - $m.id
//   - $m.species
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

