package notifiers

import (
	"context"
	"testing"

	"github.com/daniacca/achemdb/internal/achem"
)

func TestWebhookNotifier(t *testing.T) {
	notifier := NewWebhookNotifier("test-webhook", "http://localhost:9999/webhook")

	if notifier.ID() != "test-webhook" {
		t.Errorf("Expected ID 'test-webhook', got '%s'", notifier.ID())
	}

	if notifier.Type() != "webhook" {
		t.Errorf("Expected type 'webhook', got '%s'", notifier.Type())
	}

	// Test notification (will fail to send, but that's ok for unit test)
	event := achem.CreateNotificationEventWithConsumed(
		achem.EnvironmentID("test-env"),
		&mockReaction{id: "test"},
		achem.NewMolecule("Test", nil, 0),
		[]achem.Molecule{},
		achem.ReactionEffect{},
		nil,
		0,
	)

	// This will fail because there's no server, but we can test the structure
	ctx := context.Background()
	err := notifier.Notify(ctx, event)
	// We expect an error since there's no server running
	if err == nil {
		t.Log("Note: Webhook test would need a running server to fully test")
	}

	// Test close
	err = notifier.Close()
	if err != nil {
		t.Errorf("Close should not return error: %v", err)
	}
}

// mockReaction is a minimal reaction implementation for testing
type mockReaction struct {
	id string
}

func (m *mockReaction) ID() string   { return m.id }
func (m *mockReaction) Name() string { return "Mock Reaction" }
func (m *mockReaction) Rate() float64 { return 1.0 }
func (m *mockReaction) EffectiveRate(mol achem.Molecule, env achem.EnvView) float64 { return 1.0 }
func (m *mockReaction) InputPattern(mol achem.Molecule) bool { return true }
func (m *mockReaction) Apply(mol achem.Molecule, env achem.EnvView, ctx achem.ReactionContext) achem.ReactionEffect {
	return achem.ReactionEffect{}
}

