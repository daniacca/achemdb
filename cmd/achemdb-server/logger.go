package main

import (
	"fmt"
	"log"
	"strings"
)

// LogLevel represents the logging level
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "debug"
	case LogLevelInfo:
		return "info"
	case LogLevelWarn:
		return "warn"
	case LogLevelError:
		return "error"
	default:
		return "unknown"
	}
}

// parseLogLevel parses a string log level (case-insensitive) into a LogLevel
func parseLogLevel(level string) LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return LogLevelDebug
	case "info":
		return LogLevelInfo
	case "warn", "warning":
		return LogLevelWarn
	case "error":
		return LogLevelError
	default:
		return LogLevelInfo // default to info
	}
}

// Logger provides leveled logging functionality
type Logger struct {
	level LogLevel
}

// NewLogger creates a new logger with the specified log level
func NewLogger(level string) *Logger {
	return &Logger{
		level: parseLogLevel(level),
	}
}

// shouldLog returns true if the given level should be logged
func (l *Logger) shouldLog(level LogLevel) bool {
	return level >= l.level
}

// Debugf logs a debug message
func (l *Logger) Debugf(format string, v ...any) {
	if l.shouldLog(LogLevelDebug) {
		log.Printf("[DEBUG] "+format, v...)
	}
}

// Infof logs an info message
func (l *Logger) Infof(format string, v ...any) {
	if l.shouldLog(LogLevelInfo) {
		log.Printf("[INFO] "+format, v...)
	}
}

// Warnf logs a warning message
func (l *Logger) Warnf(format string, v ...any) {
	if l.shouldLog(LogLevelWarn) {
		log.Printf("[WARN] "+format, v...)
	}
}

// Errorf logs an error message
func (l *Logger) Errorf(format string, v ...any) {
	if l.shouldLog(LogLevelError) {
		log.Printf("[ERROR] "+format, v...)
	}
}

// Fatalf logs an error message and exits
func (l *Logger) Fatalf(format string, v ...any) {
	log.Fatalf("[FATAL] "+format, v...)
}

// Debug logs a debug message
func (l *Logger) Debug(v ...any) {
	if l.shouldLog(LogLevelDebug) {
		log.Print("[DEBUG] ", fmt.Sprint(v...))
	}
}

// Info logs an info message
func (l *Logger) Info(v ...any) {
	if l.shouldLog(LogLevelInfo) {
		log.Print("[INFO] ", fmt.Sprint(v...))
	}
}

// Warn logs a warning message
func (l *Logger) Warn(v ...any) {
	if l.shouldLog(LogLevelWarn) {
		log.Print("[WARN] ", fmt.Sprint(v...))
	}
}

// Error logs an error message
func (l *Logger) Error(v ...any) {
	if l.shouldLog(LogLevelError) {
		log.Print("[ERROR] ", fmt.Sprint(v...))
	}
}

