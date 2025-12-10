package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/daniacca/achemdb/internal/achem"
	achemnotifiers "github.com/daniacca/achemdb/internal/achem/notifiers"
)

// extractEnvID extracts the environment ID from a path like "/env/{envID}/..."
// Returns the environment ID and the remaining path, or empty string if not found
func extractEnvID(path string) (achem.EnvironmentID, string) {
	if !strings.HasPrefix(path, "/env/") {
		return "", ""
	}

	// Remove "/env/" prefix
	rest := path[5:]

	// Find the next "/"
	idx := strings.Index(rest, "/")
	if idx == -1 {
		// No more path segments, the whole thing is the env ID
		return achem.EnvironmentID(rest), ""
	}

	envID := achem.EnvironmentID(rest[:idx])
	remainingPath := rest[idx:]
	return envID, remainingPath
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// POST /env/{envID}/schema
// Body: SchemaConfig JSON
// Creates a new environment with the given ID and schema, or updates existing one
func (s *Server) handleSchema(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	envID, _ := extractEnvID(r.URL.Path)
	if envID == "" {
		http.Error(w, "environment ID is required in path: /env/{envID}/schema", http.StatusBadRequest)
		return
	}

	var cfg achem.SchemaConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, "invalid schema json: "+err.Error(), http.StatusBadRequest)
		return
	}

	schema, err := achem.BuildSchemaFromConfig(cfg)
	if err != nil {
		http.Error(w, "cannot build schema: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Try to create new environment, or update existing one
	err = s.manager.CreateEnvironment(envID, schema)
	if err != nil {
		// Environment already exists, update its schema
		if err := s.manager.UpdateEnvironmentSchema(envID, schema); err != nil {
			s.logger.Errorf("Failed to update environment schema: env_id=%s error=%v", envID, err)
			http.Error(w, "cannot update environment: "+err.Error(), http.StatusInternalServerError)
			return
		}
		s.logger.Infof("Environment schema updated: env_id=%s schema_name=%s", envID, cfg.Name)
	} else {
		s.logger.Infof("Environment created: env_id=%s schema_name=%s", envID, cfg.Name)
	}

	// Set the notification manager and snapshot config for the environment
	env, exists := s.manager.GetEnvironment(envID)
	if exists {
		env.SetNotificationManager(s.globalNotifierMgr)
		// Set snapshot directory if configured
		if s.snapshotDir != "" {
			env.SetSnapshotDir(s.snapshotDir)
		}
		// Set snapshot frequency
		if s.snapshotEveryTicks >= 0 {
			env.SetSnapshotEveryNTicks(s.snapshotEveryTicks)
		}
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("schema loaded"))
}

// POST /env/{envID}/molecule
// Body: { "species": "...", "payload": { ... } }
type insertMoleculeRequest struct {
	Species string         `json:"species"`
	Payload map[string]any `json:"payload"`
}

func (s *Server) handleInsertMolecule(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	envID, _ := extractEnvID(r.URL.Path)
	if envID == "" {
		http.Error(w, "environment ID is required in path: /env/{envID}/molecule", http.StatusBadRequest)
		return
	}

	env, exists := s.manager.GetEnvironment(envID)
	if !exists {
		http.Error(w, "environment not found", http.StatusNotFound)
		return
	}

	var req insertMoleculeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}

	m := achem.NewMolecule(achem.SpeciesName(req.Species), req.Payload, 0)
	env.Insert(m)

	s.logger.Debugf("Molecule inserted: env_id=%s species=%s", envID, req.Species)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// POST /env/{envID}/tick
// Manually trigger a single step (useful for testing/debugging when auto-running is disabled)
func (s *Server) handleTick(w http.ResponseWriter, r *http.Request) {
	envID, _ := extractEnvID(r.URL.Path)
	if envID == "" {
		http.Error(w, "environment ID is required in path: /env/{envID}/tick", http.StatusBadRequest)
		return
	}

	env, exists := s.manager.GetEnvironment(envID)
	if !exists {
		http.Error(w, "environment not found", http.StatusNotFound)
		return
	}

	env.Step()
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ticked"))
}

// POST /env/{envID}/start
// Start the environment auto-running with the specified interval (in milliseconds)
// Query param: interval (default: 1000ms)
func (s *Server) handleStart(w http.ResponseWriter, r *http.Request) {
	envID, _ := extractEnvID(r.URL.Path)
	if envID == "" {
		http.Error(w, "environment ID is required in path: /env/{envID}/start", http.StatusBadRequest)
		return
	}

	env, exists := s.manager.GetEnvironment(envID)
	if !exists {
		http.Error(w, "environment not found", http.StatusNotFound)
		return
	}

	// Parse interval from query param (default: 1 second)
	interval := 1000 * time.Millisecond
	if intervalStr := r.URL.Query().Get("interval"); intervalStr != "" {
		if ms, err := strconv.Atoi(intervalStr); err == nil && ms > 0 {
			interval = time.Duration(ms) * time.Millisecond
		} else {
			http.Error(w, "invalid interval: must be a positive integer (milliseconds)", http.StatusBadRequest)
			return
		}
	}

	env.Run(interval)
	s.logger.Infof("Environment started: env_id=%s interval=%v", envID, interval)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("environment started"))
}

// POST /env/{envID}/stop
// Stop the environment auto-running
func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	envID, _ := extractEnvID(r.URL.Path)
	if envID == "" {
		http.Error(w, "environment ID is required in path: /env/{envID}/stop", http.StatusBadRequest)
		return
	}

	env, exists := s.manager.GetEnvironment(envID)
	if !exists {
		http.Error(w, "environment not found", http.StatusNotFound)
		return
	}

	env.Stop()
	s.logger.Infof("Environment stopped: env_id=%s", envID)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("environment stopped"))
}

// GET /env/{envID}/molecules
func (s *Server) handleListMolecules(w http.ResponseWriter, r *http.Request) {
	envID, _ := extractEnvID(r.URL.Path)
	if envID == "" {
		http.Error(w, "environment ID is required in path: /env/{envID}/molecules", http.StatusBadRequest)
		return
	}

	env, exists := s.manager.GetEnvironment(envID)
	if !exists {
		http.Error(w, "environment not found", http.StatusNotFound)
		return
	}

	mols := env.AllMolecules()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(mols); err != nil {
		http.Error(w, "cannot encode: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// GET /envs
// List all environment IDs
func (s *Server) handleListEnvironments(w http.ResponseWriter, r *http.Request) {
	envIDs := s.manager.ListEnvironments()

	// Convert to strings for JSON encoding
	ids := make([]string, len(envIDs))
	for i, id := range envIDs {
		ids[i] = string(id)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string][]string{"environments": ids}); err != nil {
		http.Error(w, "cannot encode: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// DELETE /env/{envID}
// Delete an environment
func (s *Server) handleDeleteEnvironment(w http.ResponseWriter, r *http.Request) {
	envID, _ := extractEnvID(r.URL.Path)
	if envID == "" {
		http.Error(w, "environment ID is required in path: /env/{envID}", http.StatusBadRequest)
		return
	}

	if err := s.manager.DeleteEnvironment(envID); err != nil {
		s.logger.Warnf("Failed to delete environment: env_id=%s error=%v", envID, err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	s.logger.Infof("Environment deleted: env_id=%s", envID)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("environment deleted"))
}

// handleEnvironmentRoutes routes requests to environment-specific handlers
// Handles paths like /env/{envID}/schema, /env/{envID}/molecule, etc.
func (s *Server) handleEnvironmentRoutes(w http.ResponseWriter, r *http.Request) {
	envID, remainingPath := extractEnvID(r.URL.Path)
	if envID == "" {
		http.Error(w, "environment ID is required in path: /env/{envID}/...", http.StatusBadRequest)
		return
	}

	switch {
	case remainingPath == "/schema" && r.Method == http.MethodPost:
		s.handleSchema(w, r)
	case remainingPath == "/molecule" && r.Method == http.MethodPost:
		s.handleInsertMolecule(w, r)
	case remainingPath == "/tick" && r.Method == http.MethodPost:
		s.handleTick(w, r)
	case remainingPath == "/start" && r.Method == http.MethodPost:
		s.handleStart(w, r)
	case remainingPath == "/stop" && r.Method == http.MethodPost:
		s.handleStop(w, r)
	case remainingPath == "/molecules" && r.Method == http.MethodGet:
		s.handleListMolecules(w, r)
	case remainingPath == "/snapshot" && r.Method == http.MethodPost:
		s.handleSaveSnapshot(w, r)
	case remainingPath == "/snapshot" && r.Method == http.MethodGet:
		s.handleGetSnapshot(w, r)
	case remainingPath == "" && r.Method == http.MethodDelete:
		s.handleDeleteEnvironment(w, r)
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

// handleNotifiersRoutes handles notifier management endpoints
func (s *Server) handleNotifiersRoutes(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/notifiers" && r.Method == http.MethodGet:
		s.handleListNotifiers(w, r)
	case r.URL.Path == "/notifiers" && r.Method == http.MethodPost:
		s.handleRegisterNotifier(w, r)
	case strings.HasPrefix(r.URL.Path, "/notifiers/") && r.Method == http.MethodDelete:
		s.handleUnregisterNotifier(w, r)
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

// GET /notifiers
// List all registered notifiers
func (s *Server) handleListNotifiers(w http.ResponseWriter, _ *http.Request) {
	notifierIDs := s.globalNotifierMgr.ListNotifiers()

	// Get notifier types
	notifiers := make([]map[string]string, 0, len(notifierIDs))
	for _, id := range notifierIDs {
		notifier, exists := s.globalNotifierMgr.GetNotifier(id)
		if exists {
			notifiers = append(notifiers, map[string]string{
				"id":   id,
				"type": notifier.Type(),
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{"notifiers": notifiers}); err != nil {
		http.Error(w, "cannot encode: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// POST /notifiers
// Register a new notifier
// Body: { "type": "webhook", "id": "my-webhook", "config": { "url": "http://..." } }
type registerNotifierRequest struct {
	Type   string         `json:"type"`
	ID     string         `json:"id"`
	Config map[string]any `json:"config"`
}

func (s *Server) handleRegisterNotifier(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req registerNotifierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		http.Error(w, "notifier ID is required", http.StatusBadRequest)
		return
	}

	var notifier achem.Notifier
	var err error

	switch req.Type {
	case "webhook":
		url, ok := req.Config["url"].(string)
		if !ok || url == "" {
			http.Error(w, "webhook URL is required", http.StatusBadRequest)
			return
		}
		wh := achemnotifiers.NewWebhookNotifier(req.ID, url)

		// Set custom headers if provided
		if headers, ok := req.Config["headers"].(map[string]any); ok {
			for k, v := range headers {
				if vStr, ok := v.(string); ok {
					wh.SetHeader(k, vStr)
				}
			}
		}

		notifier = wh
	default:
		http.Error(w, "unknown notifier type: "+req.Type, http.StatusBadRequest)
		return
	}

	if err = s.globalNotifierMgr.RegisterNotifier(notifier); err != nil {
		http.Error(w, "cannot register notifier: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("notifier registered"))
}

// DELETE /notifiers/{id}
// Unregister a notifier
func (s *Server) handleUnregisterNotifier(w http.ResponseWriter, r *http.Request) {
	// Extract notifier ID from path
	path := r.URL.Path
	if !strings.HasPrefix(path, "/notifiers/") {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	notifierID := strings.TrimPrefix(path, "/notifiers/")
	if notifierID == "" {
		http.Error(w, "notifier ID is required", http.StatusBadRequest)
		return
	}

	if err := s.globalNotifierMgr.UnregisterNotifier(notifierID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("notifier unregistered"))
}

// POST /env/{envID}/snapshot
// Triggers a synchronous snapshot save
func (s *Server) handleSaveSnapshot(w http.ResponseWriter, r *http.Request) {
	envID, _ := extractEnvID(r.URL.Path)
	if envID == "" {
		http.Error(w, "environment ID is required in path: /env/{envID}/snapshot", http.StatusBadRequest)
		return
	}

	env, exists := s.manager.GetEnvironment(envID)
	if !exists {
		http.Error(w, "environment not found", http.StatusNotFound)
		return
	}

	// Check if snapshot directory is configured
	if s.snapshotDir == "" {
		http.Error(w, "snapshot directory not configured", http.StatusInternalServerError)
		return
	}

	// Ensure snapshot directory is set on environment
	env.SetSnapshotDir(s.snapshotDir)

	// Save snapshot synchronously
	if err := env.SaveSnapshot(); err != nil {
		s.logger.Errorf("Failed to save snapshot: env_id=%s error=%v", envID, err)
		http.Error(w, "failed to save snapshot: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get snapshot path for response
	path := env.SnapshotPath()
	s.logger.Debugf("Snapshot saved: env_id=%s path=%s", envID, path)

	response := map[string]string{
		"status": "ok",
		"path":   path,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "cannot encode response: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// GET /env/{envID}/snapshot
// Returns the raw snapshot JSON if it exists
func (s *Server) handleGetSnapshot(w http.ResponseWriter, r *http.Request) {
	envID, _ := extractEnvID(r.URL.Path)
	if envID == "" {
		http.Error(w, "environment ID is required in path: /env/{envID}/snapshot", http.StatusBadRequest)
		return
	}

	env, exists := s.manager.GetEnvironment(envID)
	if !exists {
		http.Error(w, "environment not found", http.StatusNotFound)
		return
	}

	// Check if snapshot directory is configured
	if s.snapshotDir == "" {
		http.Error(w, "snapshot directory not configured", http.StatusInternalServerError)
		return
	}

	// Ensure snapshot directory is set on environment
	env.SetSnapshotDir(s.snapshotDir)

	// Get snapshot path
	path := env.SnapshotPath()

	// Read snapshot file
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "snapshot not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to read snapshot: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return raw JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

