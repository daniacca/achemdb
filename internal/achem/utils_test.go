package achem

import (
	"testing"
)

func TestNewRandomID(t *testing.T) {
	// Test that IDs are generated
	id1 := NewRandomID()
	if id1 == "" {
		t.Error("Expected non-empty ID")
	}

	// Test that IDs are hex strings (16 chars for 8 bytes)
	if len(id1) != 16 {
		t.Errorf("Expected ID length 16, got %d", len(id1))
	}

	// Test uniqueness
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := NewRandomID()
		if ids[id] {
			t.Errorf("Duplicate ID found: %s", id)
		}
		ids[id] = true
	}
}

