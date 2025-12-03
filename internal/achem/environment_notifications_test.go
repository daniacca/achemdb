package achem

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestEnvironment_Notifications_Webhook(t *testing.T) {
	// Create a schema with a reaction that has notifications enabled
	cfg := SchemaConfig{
		Name:    "test",
		Species: []SpeciesConfig{{Name: "Input"}, {Name: "Output"}},
		Reactions: []ReactionConfig{
			{
				ID:   "test-reaction",
				Name: "Test Reaction",
				Input: InputConfig{
					Species: "Input",
				},
				Rate: 1.0,
				Effects: []EffectConfig{
					{
						Consume: true,
					},
					{
						Create: &CreateEffectConfig{
							Species: "Output",
						},
					},
				},
				Notify: &NotificationConfig{
					Enabled:   true,
					Notifiers: []string{"test-webhook"},
				},
			},
		},
	}

	schema, err := BuildSchemaFromConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to build schema: %v", err)
	}

	env := NewEnvironment(schema)
	env.SetEnvironmentID(EnvironmentID("test-env"))

	// Create a mock webhook notifier that tracks notifications
	var notifiedEvents []NotificationEvent
	var mu sync.Mutex

	mockNotifier := &mockNotifierForTest{
		id: "test-webhook",
		notifyFunc: func(ctx context.Context, event NotificationEvent) error {
			mu.Lock()
			notifiedEvents = append(notifiedEvents, event)
			mu.Unlock()
			return nil
		},
	}

	nm := NewNotificationManager()
	nm.RegisterNotifier(mockNotifier)
	env.SetNotificationManager(nm)

	// Insert a molecule
	mol := NewMolecule("Input", nil, 0)
	env.Insert(mol)

	// Run a step
	env.Step()

	// Wait a bit for async notification
	time.Sleep(100 * time.Millisecond)

	// Check if notification was sent
	mu.Lock()
	notifiedCount := len(notifiedEvents)
	mu.Unlock()

	if notifiedCount == 0 {
		t.Error("Expected notification to be sent")
	} else {
		event := notifiedEvents[0]
		if event.ReactionID != "test-reaction" {
			t.Errorf("Expected reaction ID 'test-reaction', got '%s'", event.ReactionID)
		}
		if event.EnvironmentID != "test-env" {
			t.Errorf("Expected environment ID 'test-env', got '%s'", event.EnvironmentID)
		}
		if len(event.CreatedMolecules) != 1 {
			t.Errorf("Expected 1 created molecule, got %d", len(event.CreatedMolecules))
		}
	}
}

func TestEnvironment_Notifications_Disabled(t *testing.T) {
	cfg := SchemaConfig{
		Name:    "test",
		Species: []SpeciesConfig{{Name: "Input"}},
		Reactions: []ReactionConfig{
			{
				ID:   "test-reaction",
				Name: "Test Reaction",
				Input: InputConfig{
					Species: "Input",
				},
				Rate: 1.0,
				Effects: []EffectConfig{
					{Consume: true},
				},
				Notify: &NotificationConfig{
					Enabled: false, // Notifications disabled
				},
			},
		},
	}

	schema, _ := BuildSchemaFromConfig(cfg)
	env := NewEnvironment(schema)

	var notified bool
	mockNotifier := &mockNotifierForTest{
		id: "test-webhook",
		notifyFunc: func(ctx context.Context, event NotificationEvent) error {
			notified = true
			return nil
		},
	}

	nm := NewNotificationManager()
	nm.RegisterNotifier(mockNotifier)
	env.SetNotificationManager(nm)

	env.Insert(NewMolecule("Input", nil, 0))
	env.Step()

	time.Sleep(100 * time.Millisecond)

	if notified {
		t.Error("Expected no notification when disabled")
	}
}

func TestEnvironment_Notifications_NoNotifiers(t *testing.T) {
	cfg := SchemaConfig{
		Name:    "test",
		Species: []SpeciesConfig{{Name: "Input"}},
		Reactions: []ReactionConfig{
			{
				ID:   "test-reaction",
				Name: "Test Reaction",
				Input: InputConfig{
					Species: "Input",
				},
				Rate: 1.0,
				Effects: []EffectConfig{
					{Consume: true},
				},
				Notify: &NotificationConfig{
					Enabled:   true,
					Notifiers: []string{}, // No notifiers specified
				},
			},
		},
	}

	schema, _ := BuildSchemaFromConfig(cfg)
	env := NewEnvironment(schema)

	var notified bool
	mockNotifier := &mockNotifierForTest{
		id: "test-webhook",
		notifyFunc: func(ctx context.Context, event NotificationEvent) error {
			notified = true
			return nil
		},
	}

	nm := NewNotificationManager()
	nm.RegisterNotifier(mockNotifier)
	env.SetNotificationManager(nm)

	env.Insert(NewMolecule("Input", nil, 0))
	env.Step()

	time.Sleep(100 * time.Millisecond)

	if notified {
		t.Error("Expected no notification when no notifiers specified")
	}
}

// mockNotifierForTest is a test implementation of Notifier
type mockNotifierForTest struct {
	id         string
	notifyFunc func(context.Context, NotificationEvent) error
}

func (m *mockNotifierForTest) ID() string { return m.id }
func (m *mockNotifierForTest) Type() string { return "mock" }
func (m *mockNotifierForTest) Notify(ctx context.Context, event NotificationEvent) error {
	if m.notifyFunc != nil {
		return m.notifyFunc(ctx, event)
	}
	return nil
}
func (m *mockNotifierForTest) Close() error { return nil }

func TestEnvironment_Callbacks_WithoutNotifiers(t *testing.T) {
	// This test verifies that callbacks work even when no notifiers are configured
	cfg := SchemaConfig{
		Name:    "test",
		Species: []SpeciesConfig{{Name: "Input"}, {Name: "Output"}},
		Reactions: []ReactionConfig{
			{
				ID:   "test-reaction",
				Name: "Test Reaction",
				Input: InputConfig{
					Species: "Input",
				},
				Rate: 1.0,
				Effects: []EffectConfig{
					{Consume: true},
					{
						Create: &CreateEffectConfig{
							Species: "Output",
						},
					},
				},
				Notify: &NotificationConfig{
					Enabled:   true,
					Notifiers: []string{}, // Empty notifiers list - callbacks should still work
				},
			},
		},
	}

	schema, err := BuildSchemaFromConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to build schema: %v", err)
	}

	env := NewEnvironment(schema)
	env.SetEnvironmentID(EnvironmentID("test-env"))

	// Register a callback - this should work even without notifiers
	var receivedEvents []NotificationEvent
	var mu sync.Mutex

	callbackID := "test-callback"
	env.RegisterCallback(callbackID, func(event NotificationEvent) {
		mu.Lock()
		receivedEvents = append(receivedEvents, event)
		mu.Unlock()
	})

	// Insert a molecule and run a step
	env.Insert(NewMolecule("Input", nil, 0))
	env.Step()

	// Wait for async notification
	time.Sleep(100 * time.Millisecond)

	// Check if callback was called
	mu.Lock()
	eventCount := len(receivedEvents)
	mu.Unlock()

	if eventCount == 0 {
		t.Error("Expected callback to receive at least one event even without notifiers")
	} else {
		event := receivedEvents[0]
		if event.ReactionID != "test-reaction" {
			t.Errorf("Expected ReactionID 'test-reaction', got '%s'", event.ReactionID)
		}
		if event.EnvironmentID != "test-env" {
			t.Errorf("Expected EnvironmentID 'test-env', got '%s'", event.EnvironmentID)
		}
		if len(event.CreatedMolecules) != 1 {
			t.Errorf("Expected 1 created molecule, got %d", len(event.CreatedMolecules))
		}
	}

	// Clean up
	env.UnregisterCallback(callbackID)
}

func TestEnvironment_RegisterCallback(t *testing.T) {
	cfg := SchemaConfig{
		Name:    "test",
		Species: []SpeciesConfig{{Name: "Input"}, {Name: "Output"}},
		Reactions: []ReactionConfig{
			{
				ID:   "test-reaction",
				Name: "Test Reaction",
				Input: InputConfig{
					Species: "Input",
				},
				Rate: 1.0,
				Effects: []EffectConfig{
					{Consume: true},
					{
						Create: &CreateEffectConfig{
							Species: "Output",
						},
					},
				},
				Notify: &NotificationConfig{
					Enabled:   true,
					Notifiers: []string{"dummy"}, // Dummy notifier, callbacks work regardless
				},
			},
		},
	}

	schema, err := BuildSchemaFromConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to build schema: %v", err)
	}

	env := NewEnvironment(schema)
	env.SetEnvironmentID(EnvironmentID("test-env"))

	// Register a callback
	var receivedEvents []NotificationEvent
	var mu sync.Mutex

	callbackID := "test-callback"
	env.RegisterCallback(callbackID, func(event NotificationEvent) {
		mu.Lock()
		receivedEvents = append(receivedEvents, event)
		mu.Unlock()
	})

	// Insert a molecule and run a step
	env.Insert(NewMolecule("Input", nil, 0))
	env.Step()

	// Wait for async notification
	time.Sleep(100 * time.Millisecond)

	// Check if callback was called
	mu.Lock()
	eventCount := len(receivedEvents)
	mu.Unlock()

	if eventCount == 0 {
		t.Error("Expected callback to receive at least one event")
	} else {
		event := receivedEvents[0]
		if event.ReactionID != "test-reaction" {
			t.Errorf("Expected ReactionID 'test-reaction', got '%s'", event.ReactionID)
		}
		if event.EnvironmentID != "test-env" {
			t.Errorf("Expected EnvironmentID 'test-env', got '%s'", event.EnvironmentID)
		}
	}
}

func TestEnvironment_UnregisterCallback(t *testing.T) {
	cfg := SchemaConfig{
		Name:    "test",
		Species: []SpeciesConfig{{Name: "Input"}, {Name: "Output"}},
		Reactions: []ReactionConfig{
			{
				ID:   "test-reaction",
				Name: "Test Reaction",
				Input: InputConfig{
					Species: "Input",
				},
				Rate: 1.0,
				Effects: []EffectConfig{
					{Consume: true},
					{
						Create: &CreateEffectConfig{
							Species: "Output",
						},
					},
				},
				Notify: &NotificationConfig{
					Enabled:   true,
					Notifiers: []string{"dummy"},
				},
			},
		},
	}

	schema, _ := BuildSchemaFromConfig(cfg)
	env := NewEnvironment(schema)

	var callCount int
	var mu sync.Mutex

	callbackID := "test-callback"
	env.RegisterCallback(callbackID, func(event NotificationEvent) {
		mu.Lock()
		callCount++
		mu.Unlock()
	})

	// Insert and step - callback should be called
	env.Insert(NewMolecule("Input", nil, 0))
	env.Step()
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	beforeUnregister := callCount
	mu.Unlock()

	if beforeUnregister == 0 {
		t.Error("Expected callback to be called before unregister")
	}

	// Unregister callback
	env.UnregisterCallback(callbackID)

	// Insert and step again - callback should NOT be called
	env.Insert(NewMolecule("Input", nil, 0))
	env.Step()
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	afterUnregister := callCount
	mu.Unlock()

	if afterUnregister != beforeUnregister {
		t.Errorf("Expected callback count to remain %d after unregister, got %d", beforeUnregister, afterUnregister)
	}
}

func TestEnvironment_RegisterCallback_Multiple(t *testing.T) {
	cfg := SchemaConfig{
		Name:    "test",
		Species: []SpeciesConfig{{Name: "Input"}, {Name: "Output"}},
		Reactions: []ReactionConfig{
			{
				ID:   "test-reaction",
				Name: "Test Reaction",
				Input: InputConfig{
					Species: "Input",
				},
				Rate: 1.0,
				Effects: []EffectConfig{
					{Consume: true},
					{
						Create: &CreateEffectConfig{
							Species: "Output",
						},
					},
				},
				Notify: &NotificationConfig{
					Enabled:   true,
					Notifiers: []string{"dummy"},
				},
			},
		},
	}

	schema, _ := BuildSchemaFromConfig(cfg)
	env := NewEnvironment(schema)

	var callback1Count, callback2Count int
	var mu sync.Mutex

	env.RegisterCallback("callback-1", func(event NotificationEvent) {
		mu.Lock()
		callback1Count++
		mu.Unlock()
	})

	env.RegisterCallback("callback-2", func(event NotificationEvent) {
		mu.Lock()
		callback2Count++
		mu.Unlock()
	})

	// Insert and step
	env.Insert(NewMolecule("Input", nil, 0))
	env.Step()
	time.Sleep(100 * time.Millisecond)

	// Both callbacks should be called
	mu.Lock()
	c1 := callback1Count
	c2 := callback2Count
	mu.Unlock()

	if c1 == 0 {
		t.Error("Expected callback-1 to be called")
	}
	if c2 == 0 {
		t.Error("Expected callback-2 to be called")
	}
}

