package achem

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
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

// notificationJob represents a job to be processed by the notification queue
type notificationJob struct {
	Event       NotificationEvent
	NotifierIDs []string
}

// NotificationManager manages all notifiers and routes notifications
type NotificationManager struct {
	mu        sync.RWMutex
	notifiers map[string]Notifier
	jobs      chan notificationJob
	closed    bool
	wg        sync.WaitGroup
}

// NewNotificationManager creates a new notification manager
func NewNotificationManager() *NotificationManager {
	mgr := &NotificationManager{
		notifiers: make(map[string]Notifier),
		jobs:      make(chan notificationJob, 1024),
		closed:    false,
	}
	mgr.startWorkers(1)
	return mgr
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
	
	nm.mu.Lock()
	defer nm.mu.Unlock()
	
	if _, exists := nm.notifiers[id]; exists {
		return fmt.Errorf("notifier with ID %s already exists", id)
	}
	
	nm.notifiers[id] = notifier
	return nil
}

// UnregisterNotifier removes a notifier from the manager
func (nm *NotificationManager) UnregisterNotifier(id string) error {
	nm.mu.Lock()
	notifier, exists := nm.notifiers[id]
	nm.mu.Unlock()
	
	if !exists {
		return fmt.Errorf("notifier with ID %s not found", id)
	}
	
	if err := notifier.Close(); err != nil {
		return fmt.Errorf("error closing notifier %s: %w", id, err)
	}
	
	nm.mu.Lock()
	delete(nm.notifiers, id)
	nm.mu.Unlock()
	
	return nil
}

// GetNotifier retrieves a notifier by ID
func (nm *NotificationManager) GetNotifier(id string) (Notifier, bool) {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	notifier, exists := nm.notifiers[id]
	return notifier, exists
}

// ListNotifiers returns a list of all registered notifier IDs
func (nm *NotificationManager) ListNotifiers() []string {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	ids := make([]string, 0, len(nm.notifiers))
	for id := range nm.notifiers {
		ids = append(ids, id)
	}
	return ids
}

// Enqueue enqueues a notification event to be processed asynchronously by worker goroutines.
// This method is non-blocking and will drop notifications if the queue is full.
func (nm *NotificationManager) Enqueue(event NotificationEvent, notifierIDs []string) {
	if len(notifierIDs) == 0 {
		return
	}
	
	nm.mu.RLock()
	closed := nm.closed
	nm.mu.RUnlock()
	
	if closed {
		return
	}
	
	// Best effort: if channel is full, drop or log and return
	select {
	case nm.jobs <- notificationJob{Event: event, NotifierIDs: notifierIDs}:
	default:
		log.Printf("notification queue full, dropping notification: reaction_id=%s", event.ReactionID)
	}
}

// startWorkers starts n worker goroutines to process notification jobs
func (nm *NotificationManager) startWorkers(n int) {
	for range n {
		nm.wg.Add(1)
		go nm.worker()
	}
}

// worker processes notification jobs from the queue
func (nm *NotificationManager) worker() {
	defer nm.wg.Done()
	for job := range nm.jobs {
		nm.dispatchJob(job)
	}
}

// dispatchJob dispatches a notification job to all specified notifiers
func (nm *NotificationManager) dispatchJob(job notificationJob) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// For each notifier ID, attempt delivery with retry/backoff
	for _, id := range job.NotifierIDs {
		nm.notifyWithRetry(ctx, id, job.Event)
	}
}

// notifyWithRetry attempts to send a notification with exponential backoff retry
func (nm *NotificationManager) notifyWithRetry(ctx context.Context, notifierID string, event NotificationEvent) {
	nm.mu.RLock()
	notifier, ok := nm.notifiers[notifierID]
	nm.mu.RUnlock()
	
	if !ok {
		log.Printf("notification failed: notifier=%s error=notifier not found", notifierID)
		return
	}
	
	// Basic retry/backoff policy
	const maxRetries = 3
	backoff := 100 * time.Millisecond
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := notifier.Notify(ctx, event)
		if err == nil {
			return
		}
		
		// Log the failure
		log.Printf("notification failed: notifier=%s attempt=%d error=%v", notifierID, attempt+1, err)
		
		if attempt == maxRetries {
			// Max retries reached, give up
			log.Printf("notification failed after %d attempts: notifier=%s", maxRetries+1, notifierID)
			return
		}
		
		// Exponential backoff
		select {
		case <-ctx.Done():
			// Context cancelled or timed out
			return
		case <-time.After(backoff):
			backoff *= 2 // exponential backoff
		}
	}
}

// Notify sends a notification event to the specified notifiers synchronously.
// This is kept for backward compatibility. For async processing, use Enqueue instead.
func (nm *NotificationManager) Notify(ctx context.Context, event NotificationEvent, notifierIDs []string) error {
	if len(notifierIDs) == 0 {
		return nil // No notifiers to notify
	}
	
	var errors []error
	for _, id := range notifierIDs {
		nm.mu.RLock()
		notifier, exists := nm.notifiers[id]
		nm.mu.RUnlock()
		
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

// Close closes all registered notifiers and shuts down worker goroutines
func (nm *NotificationManager) Close() error {
	// Mark as closed and close the jobs channel
	nm.mu.Lock()
	if nm.closed {
		nm.mu.Unlock()
		return nil
	}
	nm.closed = true
	close(nm.jobs)
	nm.mu.Unlock()
	
	// Wait for all workers to finish processing
	nm.wg.Wait()
	
	// Close all registered notifiers
	nm.mu.Lock()
	var errors []error
	for id, notifier := range nm.notifiers {
		if err := notifier.Close(); err != nil {
			errors = append(errors, fmt.Errorf("error closing notifier %s: %w", id, err))
		}
	}
	nm.notifiers = make(map[string]Notifier)
	nm.mu.Unlock()
	
	if len(errors) > 0 {
		return fmt.Errorf("errors closing notifiers: %v", errors)
	}
	
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
