package achem

import (
	"fmt"
	"strings"
)

// ValidationError collects multiple validation issues
type ValidationError struct {
	Issues []string
}

func (e *ValidationError) Error() string {
	if len(e.Issues) == 0 {
		return "invalid schema: unknown validation error"
	}
	if len(e.Issues) == 1 {
		return e.Issues[0]
	}
	return "schema validation errors: " + strings.Join(e.Issues, "; ")
}

func (e *ValidationError) Add(issue string) {
	e.Issues = append(e.Issues, issue)
}

func (e *ValidationError) HasIssues() bool {
	return len(e.Issues) > 0
}

// Valid operators for CountMoleculesConfig.Op
var validOperators = map[string]bool{
	"eq":  true,
	"ne":  true,
	"gt":  true,
	"gte": true,
	"lt":  true,
	"lte": true,
}

// ValidateSchemaConfig performs comprehensive validation of a SchemaConfig
func ValidateSchemaConfig(cfg SchemaConfig) error {
	err := &ValidationError{}

	// Validate schema name
	if cfg.Name == "" {
		err.Add("schema name is required")
	}

	// Build a map of species names for quick lookup
	speciesMap := make(map[string]bool)

	// Validate species
	for _, sp := range cfg.Species {
		if sp.Name == "" {
			err.Add("species name is required")
			continue
		}
		if speciesMap[sp.Name] {
			err.Add("duplicate species name: " + sp.Name)
		} else {
			speciesMap[sp.Name] = true
		}
	}

	// Build a map of reaction IDs for uniqueness check
	reactionIDs := make(map[string]bool)

	// Validate reactions
	for i, rc := range cfg.Reactions {
		reactionPrefix := "reaction"
		if rc.ID != "" {
			reactionPrefix = "reaction '" + rc.ID + "'"
		} else {
			reactionPrefix = "reaction at index " + fmt.Sprintf("%d", i)
		}

		// Validate reaction ID
		if rc.ID == "" {
			err.Add(reactionPrefix + ": reaction ID is required")
		} else if reactionIDs[rc.ID] {
			err.Add("duplicate reaction ID: " + rc.ID)
		} else {
			reactionIDs[rc.ID] = true
		}

		// Validate input
		if rc.Input.Species == "" {
			err.Add(reactionPrefix + ": input species is required")
		} else if !speciesMap[rc.Input.Species] {
			err.Add(reactionPrefix + ": input species '" + rc.Input.Species + "' does not exist")
		}

		// Validate partners
		for j, partner := range rc.Input.Partners {
			partnerPrefix := reactionPrefix + " partner at index " + fmt.Sprintf("%d", j)
			if partner.Species == "" {
				err.Add(partnerPrefix + ": partner species is required")
			} else if !speciesMap[partner.Species] {
				err.Add(partnerPrefix + ": partner species '" + partner.Species + "' does not exist")
			}
		}

		// Validate catalysts
		for j, catalyst := range rc.Catalysts {
			catalystPrefix := reactionPrefix + " catalyst at index " + fmt.Sprintf("%d", j)
			if catalyst.Species == "" {
				err.Add(catalystPrefix + ": catalyst species is required")
			} else if !speciesMap[catalyst.Species] {
				err.Add(catalystPrefix + ": catalyst species '" + catalyst.Species + "' does not exist")
			}
		}

		// Validate effects recursively
		validateEffects(rc.Effects, reactionPrefix, speciesMap, err)
	}

	if err.HasIssues() {
		return err
	}
	return nil
}

// validateEffects recursively validates effects
func validateEffects(effects []EffectConfig, prefix string, speciesMap map[string]bool, err *ValidationError) {
	for i, eff := range effects {
		effectPrefix := prefix + " effect at index " + fmt.Sprintf("%d", i)

		// Validate create effect
		if eff.Create != nil {
			if eff.Create.Species != "" && !speciesMap[eff.Create.Species] {
				err.Add(effectPrefix + ": create effect species '" + eff.Create.Species + "' does not exist")
			}
		}

		// Validate conditional effects
		if eff.If != nil {
			validateIfCondition(eff.If, effectPrefix, speciesMap, err)
		}

		// Recursively validate then/else effects
		if len(eff.Then) > 0 {
			validateEffects(eff.Then, effectPrefix+" then", speciesMap, err)
		}
		if len(eff.Else) > 0 {
			validateEffects(eff.Else, effectPrefix+" else", speciesMap, err)
		}
	}
}

// validateIfCondition validates an IfConditionConfig
func validateIfCondition(cond *IfConditionConfig, prefix string, speciesMap map[string]bool, err *ValidationError) {
	if cond == nil {
		return
	}

	// Check if it's a count_molecules condition
	if cond.CountMolecules != nil {
		validateCountMolecules(cond.CountMolecules, prefix, speciesMap, err)
		return
	}

	// Otherwise, it's a field condition
	// Field and Op should be set together
	if cond.Field != "" && cond.Op == "" {
		err.Add(prefix + ": if condition has field but no operator")
	} else if cond.Field == "" && cond.Op != "" {
		err.Add(prefix + ": if condition has operator but no field")
	}
	// If both are empty, that's also invalid
	if cond.Field == "" && cond.Op == "" && cond.CountMolecules == nil {
		err.Add(prefix + ": if condition must have either count_molecules or field+op")
	}
}

// validateCountMolecules validates a CountMoleculesConfig
func validateCountMolecules(cfg *CountMoleculesConfig, prefix string, speciesMap map[string]bool, err *ValidationError) {
	if cfg == nil {
		return
	}

	// Validate species
	if cfg.Species == "" {
		err.Add(prefix + ": count_molecules species is required")
	} else if !speciesMap[cfg.Species] {
		err.Add(prefix + ": count_molecules species '" + cfg.Species + "' does not exist")
	}

	// Validate Op: must have exactly one entry
	if len(cfg.Op) == 0 {
		err.Add(prefix + ": count_molecules op must have exactly one operator")
	} else if len(cfg.Op) > 1 {
		err.Add(prefix + ": count_molecules op must have exactly one operator, found " + fmt.Sprintf("%d", len(cfg.Op)))
	} else {
		// Check that the operator is valid
		for op := range cfg.Op {
			if !validOperators[op] {
				err.Add(prefix + ": count_molecules op has invalid operator '" + op + "', must be one of: eq, ne, gt, gte, lt, lte")
			}
		}
	}
}

