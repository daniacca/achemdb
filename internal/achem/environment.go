package achem

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Environment represents an isolated artificial chemistry environment.
// It contains molecules, reactions, and manages the simulation state.
// Environments are thread-safe and can run continuously or step-by-step.
type Environment struct {
	mu                sync.RWMutex
	schema            *Schema
	time              int64
	mols              map[MoleculeID]Molecule
	rand              *rand.Rand
	stopCh            chan struct{}
	isRunning         bool
	envID             EnvironmentID
	notifierMgr       *NotificationManager
	snapshotDir       string
	snapshotEveryNTicks int
	snapshotMu        sync.Mutex
}

// NewEnvironment creates a new environment with the given schema.
// The environment starts at time 0 with no molecules.
func NewEnvironment(schema *Schema) *Environment {
	return &Environment{
		schema:            schema,
		mols:              make(map[MoleculeID]Molecule),
		rand:              rand.New(rand.NewSource(time.Now().UnixNano())),
		time:              0,
		stopCh:            make(chan struct{}),
		isRunning:         false,
		notifierMgr:       NewNotificationManager(),
		snapshotEveryNTicks: 1000, // default value
	}
}

// SetEnvironmentID sets the environment ID (used for notifications)
func (e *Environment) SetEnvironmentID(id EnvironmentID) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.envID = id
}

// SetNotificationManager sets a custom notification manager
func (e *Environment) SetNotificationManager(mgr *NotificationManager) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.notifierMgr = mgr
}

// GetNotificationManager returns the notification manager
func (e *Environment) GetNotificationManager() *NotificationManager {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.notifierMgr
}

// SetSnapshotDir sets the directory where snapshots will be saved.
// If set to empty string, snapshots are disabled.
func (e *Environment) SetSnapshotDir(dir string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.snapshotDir = dir
}

// SetSnapshotEveryNTicks sets how often snapshots should be taken (in ticks).
// Snapshots are taken when time % snapshotEveryNTicks == 0.
// If set to 0 or negative, snapshots are disabled.
func (e *Environment) SetSnapshotEveryNTicks(n int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.snapshotEveryNTicks = n
}

// envView is a private adapter that exposes read-only methods
type envView struct {
	molecules []Molecule
	bySpecies map[SpeciesName][]Molecule

	// Optional: per-tick index to speed up simple equality where filters
	bySpeciesFieldValue map[SpeciesName]map[string]map[string][]Molecule
}

func (v envView) MoleculesBySpecies(species SpeciesName) []Molecule {
	if v.bySpecies == nil {
		// fallback for safety (should not happen if Step sets it)
		out := make([]Molecule, 0)
		for _, m := range v.molecules {
			if m.Species == species {
				out = append(out, m)
			}
		}
		return out
	}

	mols, ok := v.bySpecies[species]
	if !ok {
		return nil
	}

	// return a copy to keep immutability guarantees
	out := make([]Molecule, len(mols))
	copy(out, mols)
	return out
}

func (v envView) Find(filter func(Molecule) bool) []Molecule {
	out := make([]Molecule, 0)
	for _, m := range v.molecules {
		if filter(m) {
			out = append(out, m)
		}
	}
	return out
}

func (e *Environment) now() int64 {
	return e.time
}

func (e *Environment) Insert(m Molecule) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if m.ID == "" {
		m.ID = MoleculeID(NewRandomID())
	}
	if m.CreatedAt == 0 {
		m.CreatedAt = e.now()
		m.LastTouchedAt = e.now()
	}
	e.mols[m.ID] = m
}

func (e *Environment) AllMolecules() []Molecule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]Molecule, 0, len(e.mols))
	for _, m := range e.mols {
		out = append(out, m)
	}
	return out
}

// RegisterCallback registers a callback function for a given ID, used within the go lang runtime.
func (e *Environment) RegisterCallback(id string, callback func(NotificationEvent)) {
	e.notifierMgr.RegisterCallback(id, callback)
}

// UnregisterCallback unregisters a callback function for a given ID, used within the go lang runtime.
func (e *Environment) UnregisterCallback(id string) {
	e.notifierMgr.UnregisterCallback(id)
}

// a single step inside the environment, it will apply all reactions to all the molecules
// collected in the snapshot. 
func (e *Environment) Step() {
	// 1) SNAPSHOT PHASE (under lock)
	e.mu.Lock()
	e.time++

	// snapshot
	snapshot := make([]Molecule, 0, len(e.mols))
	for _, m := range e.mols {
		snapshot = append(snapshot, m)
	}

	// build per-species index for fast lookup
	bySpecies := make(map[SpeciesName][]Molecule)
	for _, m := range snapshot {
		bySpecies[m.Species] = append(bySpecies[m.Species], m)
	}

	// build per-tick field index to speed up simple equality where filters
	bySpeciesFieldValue := make(map[SpeciesName]map[string]map[string][]Molecule)
	for _, m := range snapshot {
		if len(m.Payload) == 0 {
			continue
		}

		species := m.Species
		if _, ok := bySpeciesFieldValue[species]; !ok {
			bySpeciesFieldValue[species] = make(map[string]map[string][]Molecule)
		}

		for field, value := range m.Payload {
			fieldMap := bySpeciesFieldValue[species][field]
			if fieldMap == nil {
				fieldMap = make(map[string][]Molecule)
				bySpeciesFieldValue[species][field] = fieldMap
			}

			key := indexKeyFromValue(value)
			fieldMap[key] = append(fieldMap[key], m)
		}
	}

	// build an ID→Molecule map for convenient lookups during the compute phase
	snapshotByID := make(map[MoleculeID]Molecule, len(snapshot))
	for _, m := range snapshot {
		snapshotByID[m.ID] = m
	}

	view := envView{
		molecules:            snapshot,
		bySpecies:            bySpecies,
		bySpeciesFieldValue:  bySpeciesFieldValue,
	}

	ctx := ReactionContext{
		EnvTime: e.time,
		Random:  e.rand.Float64,
	}

	// capture reactions once (schema is immutable once loaded)
	reactions := e.schema.Reactions()

	// capture envID and notifierMgr for use in compute phase (to avoid data races)
	envID := e.envID
	notifierMgr := e.notifierMgr

	e.mu.Unlock()

	// 2) COMPUTE PHASE (no lock)
	consumed := make(map[MoleculeID]struct{})
	consumedMolecules := make(map[MoleculeID]Molecule)
	changes := make(map[MoleculeID]Molecule)
	newMolecules := make([]Molecule, 0)

	for _, m := range snapshot {
		// skip molecules already marked as consumed
		if _, ok := consumed[m.ID]; ok {
			continue
		}

		for _, r := range reactions {
			if !r.InputPattern(m) {
				continue
			}

			// Use effective rate (base rate + catalyst effects)
			effectiveRate := r.EffectiveRate(m, view)
			if ctx.Random() > effectiveRate {
				continue
			}

			eff := r.Apply(m, view, ctx)

			// Check if reaction produced any effects (non-empty effect)
			hasEffects := len(eff.ConsumedIDs) > 0 || len(eff.Changes) > 0 || len(eff.NewMolecules) > 0

			// collect consumed molecules using the snapshot, not e.mols
			for _, id := range eff.ConsumedIDs {
				if mol, exists := snapshotByID[id]; exists {
					consumedMolecules[id] = mol
				}
			}

			// Send notification if reaction fired and has effects
			if hasEffects {
				e.sendNotificationWithContext(r, m, view, eff, ctx, consumedMolecules, envID, notifierMgr)
			}

			// mark consumed
			for _, id := range eff.ConsumedIDs {
				consumed[id] = struct{}{}
			}

			// apply changes (last-wins)
			for _, ch := range eff.Changes {
				if ch.Updated != nil {
					changes[ch.ID] = *ch.Updated
				}
			}

			newMolecules = append(newMolecules, eff.NewMolecules...)
		}
	}

	// 3) APPLY PHASE (under lock again)
	e.mu.Lock()
	defer e.mu.Unlock()

	// 3.1 - remove consumed molecules
	for id := range consumed {
		delete(e.mols, id)
	}

	// 3.2 - apply changes
	for id, m := range changes {
		if _, removed := consumed[id]; removed {
			continue
		}
		e.mols[id] = m
	}

	// 3.3 - insert new molecules
	for _, nm := range newMolecules {
		if nm.ID == "" {
			nm.ID = MoleculeID(NewRandomID())
		}
		if nm.CreatedAt == 0 {
			nm.CreatedAt = e.time
			nm.LastTouchedAt = e.time
		}
		e.mols[nm.ID] = nm
	}

	// 4) SNAPSHOT PHASE (if needed, non-blocking)
	if e.snapshotDir != "" && e.snapshotEveryNTicks > 0 && e.time % int64(e.snapshotEveryNTicks) == 0 {
		go e.SaveSnapshot()
	}
}

// Run will start the environment in a goroutine, starting it's own ticker that will
// run until the stop channel is closed. It can be called multiple times to restart
// after stopping.
func (e *Environment) Run(interval time.Duration) {
	e.mu.Lock()
	if e.isRunning {
		e.mu.Unlock()
		return
	}
	// Create a new stop channel for this run (allows restart after stop)
	e.stopCh = make(chan struct{})
	e.isRunning = true
	e.mu.Unlock()

	// Run in a goroutine so it doesn't block the caller
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				e.Step()
			case <-e.stopCh:
				e.mu.Lock()
				e.isRunning = false
				e.mu.Unlock()
				return
			}
		}
	}()
}

// Stop will stop the environment by closing the stop channel.
// After stopping, Run() can be called again to restart.
func (e *Environment) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.isRunning {
		return
	}
	
	// Close the channel to signal stop
	// The goroutine will detect this and set isRunning to false
	close(e.stopCh)
}

// sendNotificationWithContext sends a notification using the provided envID and notifierMgr
// This version is safe to call without holding the environment lock
func (e *Environment) sendNotificationWithContext(r Reaction, m Molecule, view EnvView, eff ReactionEffect, ctx ReactionContext, consumedMolecules map[MoleculeID]Molecule, envID EnvironmentID, notifierMgr *NotificationManager) {
	// Get notification config from reaction if it's a ConfigReaction
	notifyCfg := e.getNotificationConfig(r)
	if notifyCfg == nil || !notifyCfg.Enabled {
		return
	}

	// Check if there are callbacks registered - if so, we should enqueue even without notifiers
	hasCallbacks := notifierMgr.hasCallbacks()
	
	// If there are no notifiers and no callbacks, skip enqueuing
	if len(notifyCfg.Notifiers) == 0 && !hasCallbacks {
		return
	}

	// Find partners if this was a partner-based reaction
	partners := e.findPartnersForNotification(r, m, view)

	// Collect consumed molecules for the notification
	consumed := make([]Molecule, 0, len(eff.ConsumedIDs))
	for _, id := range eff.ConsumedIDs {
		if mol, exists := consumedMolecules[id]; exists {
			consumed = append(consumed, mol)
		}
	}

	// Create notification event
	event := CreateNotificationEventWithConsumed(
		envID,
		r,
		m,
		partners,
		eff,
		consumed,
		ctx.EnvTime,
	)

	// Enqueue notification for async processing (non-blocking)
	notifierMgr.Enqueue(event, notifyCfg.Notifiers)
}

// getNotificationConfig extracts notification config from a reaction
func (e *Environment) getNotificationConfig(r Reaction) *NotificationConfig {
	// Check if it's a ConfigReaction
	if cr, ok := r.(*ConfigReaction); ok {
		if cr.cfg.Notify != nil {
			return cr.cfg.Notify
		}
	}
	return nil
}

// findPartnersForNotification finds partners that were used in the reaction
func (e *Environment) findPartnersForNotification(r Reaction, m Molecule, view EnvView) []Molecule {
	if cr, ok := r.(*ConfigReaction); ok {
		partners := make([]Molecule, 0)
		for _, partnerCfg := range cr.cfg.Input.Partners {
			// Use the findPartners function from config_schema_builder
			foundPartners := findPartnersForReaction(partnerCfg, m, view)
			partners = append(partners, foundPartners...)
		}
		return partners
	}
	return nil
}

// findPartnersForReaction is a wrapper to access the private findPartners function
// We need to make findPartners accessible or create a public wrapper
func findPartnersForReaction(partnerCfg PartnerConfig, m Molecule, env EnvView) []Molecule {
	// Get all molecules of the specified species that match where conditions
	candidates := filterBySpeciesAndWhere(env, SpeciesName(partnerCfg.Species), partnerCfg.Where, m)

	// Filter out the molecule itself
	var matches []Molecule
	for _, candidate := range candidates {
		if candidate.ID != m.ID {
			matches = append(matches, candidate)
		}
	}

	// Return up to the required count
	count := partnerCfg.Count
	if count <= 0 {
		count = 1 // default
	}
	if len(matches) > count {
		return matches[:count]
	}
	return matches
}

// SnapshotPath returns the file path for the snapshot based on the environment ID.
// Format: "<SnapshotDir>/<envID>.snapshot.json"
func (e *Environment) SnapshotPath() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return filepath.Join(e.snapshotDir, string(e.envID)+".snapshot.json")
}

// createSnapshot creates a snapshot of the current environment state.
// It runs under a read lock to safely capture the state without blocking readers.
func (e *Environment) createSnapshot() (Snapshot, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	molecules := make([]Molecule, 0, len(e.mols))
	for _, m := range e.mols {
		molecules = append(molecules, m)
	}

	return Snapshot{
		EnvironmentID: e.envID,
		Time:          e.time,
		Molecules:     molecules,
	}, nil
}

// SaveSnapshot saves the current environment state to disk atomically.
// It uses a temporary file and rename operation to ensure atomic writes.
// This method is safe to call concurrently and will serialize snapshot attempts.
func (e *Environment) SaveSnapshot() error {
	// Serialize snapshot attempts to avoid concurrent writes
	e.snapshotMu.Lock()
	defer e.snapshotMu.Unlock()

	// Check if snapshot directory is configured
	if e.snapshotDir == "" {
		return nil // Snapshot disabled, silently skip
	}

	// Create snapshot
	snapshot, err := e.createSnapshot()
	if err != nil {
		log.Printf("snapshot failed: failed to create snapshot: %v", err)
		return err
	}

	// Encode to JSON
	data, err := EncodeSnapshotJSON(snapshot)
	if err != nil {
		log.Printf("snapshot failed: failed to encode snapshot: %v", err)
		return err
	}

	// Get snapshot path
	path := e.SnapshotPath()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		log.Printf("snapshot failed: failed to create snapshot directory: %v", err)
		return err
	}

	// Write atomically using temp file + rename
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		log.Printf("snapshot failed: failed to write temp file: %v", err)
		return err
	}

	if err := os.Rename(tempPath, path); err != nil {
		// Clean up temp file on error
		os.Remove(tempPath)
		log.Printf("snapshot failed: failed to rename temp file: %v", err)
		return err
	}

	log.Printf("snapshot created: env_id=%s time=%d molecules=%d path=%s", snapshot.EnvironmentID, snapshot.Time, len(snapshot.Molecules), path)
	return nil
}

// LoadSnapshot loads a snapshot from disk and restores the environment state.
// If the snapshot directory is not configured or the snapshot file does not exist, this is a no-op and returns nil.
// The snapshot is validated to ensure:
//   - The snapshot's EnvironmentID matches the environment's ID
//   - All molecule species exist in the schema
//
// On success, the environment's time and molecules are restored from the snapshot.
func (e *Environment) LoadSnapshot() error {
	// Check if snapshot directory is configured
	e.mu.RLock()
	snapshotDir := e.snapshotDir
	e.mu.RUnlock()

	if snapshotDir == "" {
		return nil // Snapshot directory not configured, nothing to load
	}

	// Get snapshot path
	path := e.SnapshotPath()

	// Check if file exists - if not, no-op
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // Snapshot doesn't exist, nothing to load
	}

	// Read snapshot file
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read snapshot file: %w", err)
	}

	// Decode JSON
	snapshot, err := DecodeSnapshotJSON(data)
	if err != nil {
		return fmt.Errorf("failed to decode snapshot: %w", err)
	}

	// Validate snapshot EnvironmentID matches
	e.mu.RLock()
	envID := e.envID
	schema := e.schema
	e.mu.RUnlock()

	if snapshot.EnvironmentID != envID {
		return fmt.Errorf("snapshot environment ID mismatch: expected %s, got %s", envID, snapshot.EnvironmentID)
	}

	// Validate snapshot (checks species exist in schema)
	if err := ValidateSnapshot(snapshot, schema); err != nil {
		return fmt.Errorf("snapshot validation failed: %w", err)
	}

	// Restore environment state under lock
	e.mu.Lock()
	defer e.mu.Unlock()

	e.time = snapshot.Time

	// Restore molecules
	e.mols = make(map[MoleculeID]Molecule, len(snapshot.Molecules))
	for _, m := range snapshot.Molecules {
		e.mols[m.ID] = m
	}

	log.Printf("snapshot loaded: env_id=%s time=%d molecules=%d path=%s", snapshot.EnvironmentID, snapshot.Time, len(snapshot.Molecules), path)
	return nil
}
