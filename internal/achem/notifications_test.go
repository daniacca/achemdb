package achem

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

// mockNotifier is a test implementation of Notifier
type mockNotifier struct {
	id         string
	notifyFunc func(context.Context, NotificationEvent) error
	closeFunc  func() error
	notifyCount int
	mu         sync.Mutex
}

func (m *mockNotifier) ID() string { return m.id }
func (m *mockNotifier) Type() string { return "mock" }
func (m *mockNotifier) Notify(ctx context.Context, event NotificationEvent) error {
	m.mu.Lock()
	m.notifyCount++
	m.mu.Unlock()
	if m.notifyFunc != nil {
		return m.notifyFunc(ctx, event)
	}
	return nil
}
func (m *mockNotifier) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func (m *mockNotifier) getNotifyCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.notifyCount
}

func TestNewNotificationManager(t *testing.T) {
	nm := NewNotificationManager()
	if nm == nil {
		t.Fatal("NewNotificationManager returned nil")
	}
	
	// Test that it's not closed
	notifiers := nm.ListNotifiers()
	if notifiers == nil {
		t.Error("Expected non-nil notifiers list")
	}
	if len(notifiers) != 0 {
		t.Errorf("Expected empty notifiers list, got %d", len(notifiers))
	}
	
	// Cleanup
	if err := nm.Close(); err != nil {
		t.Errorf("Close returned error: %v", err)
	}
}

func TestNotificationManager_RegisterNotifier(t *testing.T) {
	nm := NewNotificationManager()
	defer nm.Close()
	
	// Test successful registration
	notifier := &mockNotifier{id: "test-1"}
	err := nm.RegisterNotifier(notifier)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Test duplicate registration
	err = nm.RegisterNotifier(&mockNotifier{id: "test-1"})
	if err == nil {
		t.Error("Expected error for duplicate registration")
	}
	
	// Test nil notifier
	err = nm.RegisterNotifier(nil)
	if err == nil {
		t.Error("Expected error for nil notifier")
	}
	
	// Test empty ID
	err = nm.RegisterNotifier(&mockNotifier{id: ""})
	if err == nil {
		t.Error("Expected error for empty ID")
	}
	
	// Test multiple notifiers
	nm.RegisterNotifier(&mockNotifier{id: "test-2"})
	nm.RegisterNotifier(&mockNotifier{id: "test-3"})
	
	notifiers := nm.ListNotifiers()
	if len(notifiers) != 3 {
		t.Errorf("Expected 3 notifiers, got %d", len(notifiers))
	}
}

func TestNotificationManager_UnregisterNotifier(t *testing.T) {
	nm := NewNotificationManager()
	defer nm.Close()
	
	// Test unregistering non-existent notifier
	err := nm.UnregisterNotifier("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent notifier")
	}
	
	// Test successful unregistration
	notifier := &mockNotifier{id: "test-1"}
	nm.RegisterNotifier(notifier)
	
	err = nm.UnregisterNotifier("test-1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Verify it's removed
	_, exists := nm.GetNotifier("test-1")
	if exists {
		t.Error("Expected notifier to be removed")
	}
	
	// Test unregistration with close error
	closeErr := &mockNotifier{
		id: "test-close-error",
		closeFunc: func() error {
			return &testError{msg: "close error"}
		},
	}
	nm.RegisterNotifier(closeErr)
	
	err = nm.UnregisterNotifier("test-close-error")
	if err == nil {
		t.Error("Expected error when close fails")
	}
}

func TestNotificationManager_GetNotifier(t *testing.T) {
	nm := NewNotificationManager()
	defer nm.Close()
	
	// Test getting non-existent notifier
	_, exists := nm.GetNotifier("non-existent")
	if exists {
		t.Error("Expected notifier not to exist")
	}
	
	// Test getting existing notifier
	notifier := &mockNotifier{id: "test-1"}
	nm.RegisterNotifier(notifier)
	
	retrieved, exists := nm.GetNotifier("test-1")
	if !exists {
		t.Error("Expected notifier to exist")
	}
	if retrieved.ID() != "test-1" {
		t.Errorf("Expected ID 'test-1', got '%s'", retrieved.ID())
	}
}

func TestNotificationManager_ListNotifiers(t *testing.T) {
	nm := NewNotificationManager()
	defer nm.Close()
	
	// Test empty list
	notifiers := nm.ListNotifiers()
	if len(notifiers) != 0 {
		t.Errorf("Expected empty list, got %d", len(notifiers))
	}
	
	// Test with multiple notifiers
	nm.RegisterNotifier(&mockNotifier{id: "test-1"})
	nm.RegisterNotifier(&mockNotifier{id: "test-2"})
	nm.RegisterNotifier(&mockNotifier{id: "test-3"})
	
	notifiers = nm.ListNotifiers()
	if len(notifiers) != 3 {
		t.Errorf("Expected 3 notifiers, got %d", len(notifiers))
	}
	
	// Verify all IDs are present
	ids := make(map[string]bool)
	for _, id := range notifiers {
		ids[id] = true
	}
	if !ids["test-1"] || !ids["test-2"] || !ids["test-3"] {
		t.Error("Expected all notifier IDs to be present")
	}
}

func TestNotificationManager_Enqueue(t *testing.T) {
	nm := NewNotificationManager()
	defer nm.Close()
	
	// Test with empty notifier list
	event := NotificationEvent{ReactionID: "test-reaction"}
	nm.Enqueue(event, []string{})
	// Should not panic or error
	
	// Test with non-existent notifier (should be handled gracefully)
	nm.Enqueue(event, []string{"non-existent"})
	time.Sleep(50 * time.Millisecond) // Give worker time to process
	
	// Test with valid notifier
	notifier := &mockNotifier{id: "test-1"}
	nm.RegisterNotifier(notifier)
	
	nm.Enqueue(event, []string{"test-1"})
	time.Sleep(100 * time.Millisecond) // Give worker time to process
	
	if notifier.getNotifyCount() != 1 {
		t.Errorf("Expected 1 notification, got %d", notifier.getNotifyCount())
	}
	
	// Test with closed manager
	nm.Close()
	nm.Enqueue(event, []string{"test-1"})
	// Should not panic
}

func TestNotificationManager_Notify(t *testing.T) {
	nm := NewNotificationManager()
	defer nm.Close()
	
	ctx := context.Background()
	event := NotificationEvent{ReactionID: "test-reaction"}
	
	// Test with empty notifier list
	err := nm.Notify(ctx, event, []string{})
	if err != nil {
		t.Errorf("Expected no error with empty list, got %v", err)
	}
	
	// Test with non-existent notifier
	err = nm.Notify(ctx, event, []string{"non-existent"})
	if err == nil {
		t.Error("Expected error for non-existent notifier")
	}
	
	// Test with valid notifier
	notifier := &mockNotifier{id: "test-1"}
	nm.RegisterNotifier(notifier)
	
	err = nm.Notify(ctx, event, []string{"test-1"})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if notifier.getNotifyCount() != 1 {
		t.Errorf("Expected 1 notification, got %d", notifier.getNotifyCount())
	}
	
	// Test with multiple notifiers
	notifier2 := &mockNotifier{id: "test-2"}
	nm.RegisterNotifier(notifier2)
	
	err = nm.Notify(ctx, event, []string{"test-1", "test-2"})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if notifier.getNotifyCount() != 2 {
		t.Errorf("Expected 2 notifications for notifier1, got %d", notifier.getNotifyCount())
	}
	if notifier2.getNotifyCount() != 1 {
		t.Errorf("Expected 1 notification for notifier2, got %d", notifier2.getNotifyCount())
	}
	
	// Test with notifier that fails
	failingNotifier := &mockNotifier{
		id: "test-fail",
		notifyFunc: func(ctx context.Context, event NotificationEvent) error {
			return &testError{msg: "notification failed"}
		},
	}
	nm.RegisterNotifier(failingNotifier)
	
	err = nm.Notify(ctx, event, []string{"test-fail"})
	if err == nil {
		t.Error("Expected error when notifier fails")
	}
	
	// Test with mix of success and failure
	err = nm.Notify(ctx, event, []string{"test-1", "test-fail"})
	if err == nil {
		t.Error("Expected error when one notifier fails")
	}
}

func TestNotificationManager_Close(t *testing.T) {
	nm := NewNotificationManager()
	
	// Register some notifiers
	notifier1 := &mockNotifier{id: "test-1"}
	notifier2 := &mockNotifier{
		id: "test-2",
		closeFunc: func() error {
			return &testError{msg: "close error"}
		},
	}
	nm.RegisterNotifier(notifier1)
	nm.RegisterNotifier(notifier2)
	
	// Test close
	err := nm.Close()
	if err == nil {
		t.Error("Expected error when one notifier fails to close")
	}
	
	// Test double close
	err = nm.Close()
	if err != nil {
		t.Errorf("Expected no error on double close, got %v", err)
	}
	
	// Test that enqueue doesn't panic after close
	event := NotificationEvent{ReactionID: "test"}
	nm.Enqueue(event, []string{"test-1"})
	time.Sleep(50 * time.Millisecond)
}

func TestCreateNotificationEventWithConsumed(t *testing.T) {
	reaction := &mockReaction{
		id:   "test-reaction",
		name: "Test Reaction",
	}
	
	inputMol := NewMolecule("Input", map[string]any{"value": 1}, 0)
	partner1 := NewMolecule("Partner", map[string]any{"value": 2}, 0)
	partner2 := NewMolecule("Partner", map[string]any{"value": 3}, 0)
	partners := []Molecule{partner1, partner2}
	
	consumed1 := NewMolecule("Input", nil, 0)
	consumed2 := NewMolecule("Input", nil, 0)
	consumed := []Molecule{consumed1, consumed2}
	
	created1 := NewMolecule("Output", map[string]any{"value": 10}, 0)
	updated1 := NewMolecule("Input", map[string]any{"value": 5}, 0)
	
	effect := ReactionEffect{
		ConsumedIDs: []MoleculeID{consumed1.ID, consumed2.ID},
		Changes: []MoleculeChange{
			{ID: updated1.ID, Updated: &updated1},
		},
		NewMolecules: []Molecule{created1},
	}
	
	envID := EnvironmentID("test-env")
	envTime := int64(100)
	
	event := CreateNotificationEventWithConsumed(
		envID,
		reaction,
		inputMol,
		partners,
		effect,
		consumed,
		envTime,
	)
	
	if event.EnvironmentID != envID {
		t.Errorf("Expected EnvironmentID %s, got %s", envID, event.EnvironmentID)
	}
	if event.ReactionID != "test-reaction" {
		t.Errorf("Expected ReactionID 'test-reaction', got '%s'", event.ReactionID)
	}
	if event.ReactionName != "Test Reaction" {
		t.Errorf("Expected ReactionName 'Test Reaction', got '%s'", event.ReactionName)
	}
	if event.EnvTime != envTime {
		t.Errorf("Expected EnvTime %d, got %d", envTime, event.EnvTime)
	}
	if event.InputMolecule.ID != inputMol.ID {
		t.Errorf("Expected InputMolecule ID %s, got %s", inputMol.ID, event.InputMolecule.ID)
	}
	if len(event.Partners) != 2 {
		t.Errorf("Expected 2 partners, got %d", len(event.Partners))
	}
	if len(event.ConsumedMolecules) != 2 {
		t.Errorf("Expected 2 consumed molecules, got %d", len(event.ConsumedMolecules))
	}
	if len(event.CreatedMolecules) != 1 {
		t.Errorf("Expected 1 created molecule, got %d", len(event.CreatedMolecules))
	}
	if len(event.UpdatedMolecules) != 1 {
		t.Errorf("Expected 1 updated molecule, got %d", len(event.UpdatedMolecules))
	}
	if event.UpdatedMolecules[0].ID != updated1.ID {
		t.Errorf("Expected updated molecule ID %s, got %s", updated1.ID, event.UpdatedMolecules[0].ID)
	}
}

func TestNotificationEvent_JSON(t *testing.T) {
	event := NotificationEvent{
		EnvironmentID: "test-env",
		ReactionID:    "test-reaction",
		ReactionName:  "Test Reaction",
		Timestamp:     1234567890,
		EnvTime:       100,
		InputMolecule: NewMolecule("Input", map[string]any{"value": 1}, 0),
		Effect: ReactionEffect{
			NewMolecules: []Molecule{NewMolecule("Output", nil, 0)},
		},
	}
	
	jsonData, err := event.JSON()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if len(jsonData) == 0 {
		t.Error("Expected non-empty JSON data")
	}
	
	// Verify it's valid JSON by unmarshaling
	var decoded NotificationEvent
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("Expected valid JSON, got error: %v", err)
	}
	
	if decoded.EnvironmentID != event.EnvironmentID {
		t.Errorf("Expected EnvironmentID %s, got %s", event.EnvironmentID, decoded.EnvironmentID)
	}
	if decoded.ReactionID != event.ReactionID {
		t.Errorf("Expected ReactionID %s, got %s", event.ReactionID, decoded.ReactionID)
	}
}

// testError is a simple error implementation for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

