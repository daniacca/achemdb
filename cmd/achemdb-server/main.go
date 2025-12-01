package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/daniacca/achemdb/internal/achem"
)

type Server struct {
	mu  sync.RWMutex
	env *achem.Environment
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// POST /schema
// Body: SchemaConfig JSON
func (s *Server) handleSchema(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var cfg achem.SchemaConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, "invalid schema json: " + err.Error(), http.StatusBadRequest)
		return
	}

	schema, err := achem.BuildSchemaFromConfig(cfg)
	if err != nil {
		http.Error(w, "cannot build schema: " + err.Error(), http.StatusBadRequest)
		return
	}

	env := achem.NewEnvironment(schema)

	s.mu.Lock()
	s.env = env
	s.mu.Unlock()

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("schema loaded"))
}

// POST /molecule
// Body: { "species": "...", "payload": { ... } }
type insertMoleculeRequest struct {
	Species string         `json:"species"`
	Payload map[string]any `json:"payload"`
}

func (s *Server) handleInsertMolecule(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	s.mu.RLock()
	env := s.env
	s.mu.RUnlock()

	if env == nil {
		http.Error(w, "no schema loaded", http.StatusBadRequest)
		return
	}

	var req insertMoleculeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json: " + err.Error(), http.StatusBadRequest)
		return
	}

	m := achem.NewMolecule(achem.SpeciesName(req.Species), req.Payload, 0)
	env.Insert(m)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// POST /tick
// Manually trigger a single step (useful for testing/debugging when auto-running is disabled)
func (s *Server) handleTick(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	env := s.env
	s.mu.RUnlock()

	if env == nil {
		http.Error(w, "no schema loaded", http.StatusBadRequest)
		return
	}

	env.Step()
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ticked"))
}

// POST /start
// Start the environment auto-running with the specified interval (in milliseconds)
// Query param: interval (default: 1000ms)
func (s *Server) handleStart(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	env := s.env
	s.mu.RUnlock()

	if env == nil {
		http.Error(w, "no schema loaded", http.StatusBadRequest)
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

// POST /stop
// Stop the environment auto-running
func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	env := s.env
	s.mu.RUnlock()

	if env == nil {
		http.Error(w, "no schema loaded", http.StatusBadRequest)
		return
	}

	env.Stop()
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("environment stopped"))
}

// GET /molecules
func (s *Server) handleListMolecules(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	env := s.env
	s.mu.RUnlock()

	if env == nil {
		http.Error(w, "no schema loaded", http.StatusBadRequest)
		return
	}

	mols := env.AllMolecules()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(mols); err != nil {
		http.Error(w, "cannot encode: " + err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {
	srv := NewServer()

	http.HandleFunc("/healthz", srv.handleHealth)
	http.HandleFunc("/schema", srv.handleSchema)
	http.HandleFunc("/molecule", srv.handleInsertMolecule)
	http.HandleFunc("/tick", srv.handleTick)
	http.HandleFunc("/start", srv.handleStart)
	http.HandleFunc("/stop", srv.handleStop)
	http.HandleFunc("/molecules", srv.handleListMolecules)

	log.Println("achemdb-server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
