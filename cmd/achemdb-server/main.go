package main

import (
	"net/http"

	"github.com/daniacca/achemdb/internal/achem"
)

// Server represents the HTTP server for AChemDB
type Server struct {
	manager           *achem.EnvironmentManager
	globalNotifierMgr *achem.NotificationManager
	snapshotDir       string
	snapshotEveryTicks int
	logger            *Logger
}

// NewServer creates a new server instance
func NewServer(logger *Logger) *Server {
	globalMgr := achem.NewNotificationManager()
	return &Server{
		manager:           achem.NewEnvironmentManager(),
		globalNotifierMgr: globalMgr,
		logger:            logger,
	}
}

// SetSnapshotDir sets the snapshot directory for all environments
func (s *Server) SetSnapshotDir(dir string) {
	s.snapshotDir = dir
}

// SetSnapshotEveryTicks sets the snapshot frequency for all environments
func (s *Server) SetSnapshotEveryTicks(ticks int) {
	s.snapshotEveryTicks = ticks
}

func main() {
	cfg := loadServerConfig()

	// Create logger with configured log level
	logger := NewLogger(cfg.LogLevel)

	srv := NewServer(logger)
	srv.SetSnapshotDir(cfg.SnapshotDir)
	srv.SetSnapshotEveryTicks(cfg.SnapshotEveryTicks)

	// Load initial schema if provided
	if cfg.SchemaFile != "" {
		logger.Infof("Loading initial schema from %s into environment %s", cfg.SchemaFile, cfg.DefaultEnvID)
		if err := applyInitialSchemaToEnvironment(srv.manager, srv.globalNotifierMgr, cfg.SchemaFile, achem.EnvironmentID(cfg.DefaultEnvID), cfg.SnapshotDir, cfg.SnapshotEveryTicks); err != nil {
			logger.Fatalf("Failed to load initial schema: %v", err)
		}
		logger.Infof("Initial schema loaded successfully")
	}

	// Register HTTP handlers
	http.HandleFunc("/healthz", srv.handleHealth)
	http.HandleFunc("/envs", srv.handleListEnvironments)
	http.HandleFunc("/notifiers", srv.handleNotifiersRoutes)
	http.HandleFunc("/notifiers/", srv.handleNotifiersRoutes)
	http.HandleFunc("/env/", srv.handleEnvironmentRoutes)

	logger.Infof("achemdb-server listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, nil); err != nil {
		logger.Fatalf("Server stopped: %v", err)
	}
}
