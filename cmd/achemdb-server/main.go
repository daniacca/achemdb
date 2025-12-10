package main

import (
	"net/http"

	"github.com/daniacca/achemdb/internal/achem"
)

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
