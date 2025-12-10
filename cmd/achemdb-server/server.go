package main

import "github.com/daniacca/achemdb/internal/achem"

// achemLoggerAdapter adapts the server's Logger to the achem.Logger interface
type achemLoggerAdapter struct {
	logger *Logger
}

func (a *achemLoggerAdapter) Debugf(format string, v ...any) {
	a.logger.Debugf(format, v...)
}

func (a *achemLoggerAdapter) Infof(format string, v ...any) {
	a.logger.Infof(format, v...)
}

func (a *achemLoggerAdapter) Warnf(format string, v ...any) {
	a.logger.Warnf(format, v...)
}

func (a *achemLoggerAdapter) Errorf(format string, v ...any) {
	a.logger.Errorf(format, v...)
}

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
	// Convert server logger to achem.Logger interface
	achemLogger := &achemLoggerAdapter{logger: logger}
	globalMgr := achem.NewNotificationManagerWithLogger(achemLogger)
	return &Server{
		manager:           achem.NewEnvironmentManagerWithLogger(achemLogger),
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