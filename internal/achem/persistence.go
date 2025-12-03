package achem

import (
	"encoding/json"
	"fmt"
)

// Snapshot represents a point-in-time capture of an environment's state.
// It includes the environment ID, current time, and all molecules.
type Snapshot struct {
	EnvironmentID EnvironmentID `json:"environment_id"`
	Time          int64         `json:"time"`
	Molecules     []Molecule    `json:"molecules"`
}

// ValidateSnapshot performs validation checks on a snapshot.
// It verifies that:
//   - All molecule IDs are non-empty
//   - All species exist in the provided schema (if schema is not nil)
//
// If schema is nil, only ID validation is performed.
// Returns an error if validation fails, nil otherwise.
func ValidateSnapshot(snapshot Snapshot, schema *Schema) error {
	// Track seen IDs to detect duplicates
	seenIDs := make(map[MoleculeID]struct{})

	for i, mol := range snapshot.Molecules {
		// Validate molecule ID is non-empty
		if mol.ID == "" {
			return fmt.Errorf("molecule at index %d has empty ID", i)
		}

		// Check for duplicate IDs
		if _, exists := seenIDs[mol.ID]; exists {
			return fmt.Errorf("duplicate molecule ID: %s", mol.ID)
		}
		seenIDs[mol.ID] = struct{}{}

		// Validate species exists in schema (if schema provided)
		if schema != nil {
			if _, exists := schema.Species(mol.Species); !exists {
				return fmt.Errorf("molecule %s has invalid species: %s (not found in schema)", mol.ID, mol.Species)
			}
		}
	}

	return nil
}

// EncodeSnapshotJSON encodes a snapshot to JSON format.
// Returns the JSON bytes and any encoding error.
func EncodeSnapshotJSON(snapshot Snapshot) ([]byte, error) {
	data, err := json.Marshal(snapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to encode snapshot: %w", err)
	}
	return data, nil
}

// DecodeSnapshotJSON decodes a snapshot from JSON format.
// Returns the decoded snapshot and any decoding error.
func DecodeSnapshotJSON(data []byte) (Snapshot, error) {
	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return Snapshot{}, fmt.Errorf("failed to decode snapshot: %w", err)
	}
	return snapshot, nil
}

