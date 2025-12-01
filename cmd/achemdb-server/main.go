package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/daniacca/achemdb/internal/achem"
)

type Server struct {
	manager *achem.EnvironmentManager
}

func NewServer() *Server {
	return &Server{
		manager: achem.NewEnvironmentManager(),
	}
}

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
			http.Error(w, "cannot update environment: "+err.Error(), http.StatusInternalServerError)
			return
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
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("environment deleted"))
}

func main() {
	srv := NewServer()

	http.HandleFunc("/healthz", srv.handleHealth)
	http.HandleFunc("/envs", srv.handleListEnvironments)
	http.HandleFunc("/env/", srv.handleEnvironmentRoutes)

	log.Println("achemdb-server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
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
		case remainingPath == "" && r.Method == http.MethodDelete:
			s.handleDeleteEnvironment(w, r)
		default:
			http.Error(w, "not found", http.StatusNotFound)
	}
}
