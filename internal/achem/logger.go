package achem

// Logger interface for logging operations, injectable into the achem package.
type Logger interface {
	Debugf(format string, v ...any)
	Infof(format string, v ...any)
	Warnf(format string, v ...any)
	Errorf(format string, v ...any)
}

// NoOpLogger is a logger that does nothing (useful for testing or when logging is disabled)
type NoOpLogger struct{}

func (n *NoOpLogger) Debugf(format string, v ...any) {}
func (n *NoOpLogger) Infof(format string, v ...any)  {}
func (n *NoOpLogger) Warnf(format string, v ...any)  {}
func (n *NoOpLogger) Errorf(format string, v ...any)  {}

// NewNoOpLogger creates a no-op logger
func NewNoOpLogger() Logger {
	return &NoOpLogger{}
}

