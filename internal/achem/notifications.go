package achem

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// NotificationEvent represents an event that occurred when a reaction fired
type NotificationEvent struct {
	EnvironmentID EnvironmentID `json:"environment_id"`
	ReactionID    string        `json:"reaction_id"`
	ReactionName  string        `json:"reaction_name"`
	Timestamp     int64         `json:"timestamp"`
	EnvTime       int64         `json:"env_time"`
	
	// Molecules involved in the reaction
	InputMolecule     Molecule   `json:"input_molecule"`
	Partners          []Molecule `json:"partners,omitempty"`
	ConsumedMolecules []Molecule `json:"consumed_molecules,omitempty"`
	CreatedMolecules  []Molecule `json:"created_molecules,omitempty"`
	UpdatedMolecules  []Molecule `json:"updated_molecules,omitempty"`
	
	// Effect summary
	Effect ReactionEffect `json:"effect"`
}

// Notifier is the interface that all notification channels must implement
type Notifier interface {
	// ID returns a unique identifier for this notifier
	ID() string
	
	// Type returns the type of notifier (e.g., "webhook", "websocket", "rabbitmq")
	Type() string
	
	// Notify sends a notification event. Returns an error if notification fails.
	// The context can be used for cancellation and timeout.
	Notify(ctx context.Context, event NotificationEvent) error
	
	// Close closes the notifier and releases any resources
	Close() error
}

// NotificationConfig specifies which notifiers should be triggered for a reaction
type NotificationConfig struct {
	Enabled   bool     `json:"enabled"`   // Whether notifications are enabled for this reaction
	Notifiers []string `json:"notifiers"` // List of notifier IDs to trigger
}

// NotificationManager manages all notifiers and routes notifications
type NotificationManager struct {
	notifiers map[string]Notifier
}

// NewNotificationManager creates a new notification manager
func NewNotificationManager() *NotificationManager {
	return &NotificationManager{
		notifiers: make(map[string]Notifier),
	}
}

// RegisterNotifier registers a notifier with the manager
func (nm *NotificationManager) RegisterNotifier(notifier Notifier) error {
	if notifier == nil {
		return fmt.Errorf("notifier cannot be nil")
	}
	
	id := notifier.ID()
	if id == "" {
		return fmt.Errorf("notifier ID cannot be empty")
	}
	
	if _, exists := nm.notifiers[id]; exists {
		return fmt.Errorf("notifier with ID %s already exists", id)
	}
	
	nm.notifiers[id] = notifier
	return nil
}

// UnregisterNotifier removes a notifier from the manager
func (nm *NotificationManager) UnregisterNotifier(id string) error {
	notifier, exists := nm.notifiers[id]
	if !exists {
		return fmt.Errorf("notifier with ID %s not found", id)
	}
	
	if err := notifier.Close(); err != nil {
		return fmt.Errorf("error closing notifier %s: %w", id, err)
	}
	
	delete(nm.notifiers, id)
	return nil
}

// GetNotifier retrieves a notifier by ID
func (nm *NotificationManager) GetNotifier(id string) (Notifier, bool) {
	notifier, exists := nm.notifiers[id]
	return notifier, exists
}

// ListNotifiers returns a list of all registered notifier IDs
func (nm *NotificationManager) ListNotifiers() []string {
	ids := make([]string, 0, len(nm.notifiers))
	for id := range nm.notifiers {
		ids = append(ids, id)
	}
	return ids
}

// Notify sends a notification event to the specified notifiers
func (nm *NotificationManager) Notify(ctx context.Context, event NotificationEvent, notifierIDs []string) error {
	if len(notifierIDs) == 0 {
		return nil // No notifiers to notify
	}
	
	var errors []error
	for _, id := range notifierIDs {
		notifier, exists := nm.notifiers[id]
		if !exists {
			errors = append(errors, fmt.Errorf("notifier %s not found", id))
			continue
		}
		
		if err := notifier.Notify(ctx, event); err != nil {
			errors = append(errors, fmt.Errorf("notifier %s failed: %w", id, err))
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("notification errors: %v", errors)
	}
	
	return nil
}

// Close closes all registered notifiers
func (nm *NotificationManager) Close() error {
	var errors []error
	for id, notifier := range nm.notifiers {
		if err := notifier.Close(); err != nil {
			errors = append(errors, fmt.Errorf("error closing notifier %s: %w", id, err))
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("errors closing notifiers: %v", errors)
	}
	
	nm.notifiers = make(map[string]Notifier)
	return nil
}

// CreateNotificationEventWithConsumed creates a NotificationEvent with consumed molecules
func CreateNotificationEventWithConsumed(
	envID EnvironmentID,
	reaction Reaction,
	inputMolecule Molecule,
	partners []Molecule,
	effect ReactionEffect,
	consumedMolecules []Molecule,
	envTime int64,
) NotificationEvent {
	// Collect created molecules
	created := effect.NewMolecules
	
	// Collect updated molecules
	updated := make([]Molecule, 0, len(effect.Changes))
	for _, change := range effect.Changes {
		if change.Updated != nil {
			updated = append(updated, *change.Updated)
		}
	}
	
	return NotificationEvent{
		EnvironmentID:     envID,
		ReactionID:        reaction.ID(),
		ReactionName:      reaction.Name(),
		Timestamp:         time.Now().Unix(),
		EnvTime:           envTime,
		InputMolecule:     inputMolecule,
		Partners:          partners,
		ConsumedMolecules: consumedMolecules,
		CreatedMolecules:  created,
		UpdatedMolecules:  updated,
		Effect:            effect,
	}
}

// NotificationEventJSON returns the notification event as JSON bytes
func (ne NotificationEvent) JSON() ([]byte, error) {
	return json.Marshal(ne)
}
