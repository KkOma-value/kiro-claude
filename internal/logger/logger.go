package logger

import (
	"fmt"
	"os"
)

// LogLevel represents logging level
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

// Logger interface for structured logging
type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	WithFields(fields map[string]interface{}) Logger
}

// SimpleLogger provides basic JSON-like logging to stdout/stderr
type SimpleLogger struct {
	level LogLevel
}

// NewSimpleLogger creates a new simple logger
func NewSimpleLogger(level LogLevel) Logger {
	return &SimpleLogger{
		level: level,
	}
}

// Debugf logs a debug message
func (l *SimpleLogger) Debugf(format string, args ...interface{}) {
	if l.level <= LevelDebug {
		fmt.Fprintf(os.Stdout, "[DEBUG] "+format+"\n", args...)
	}
}

// Infof logs an info message
func (l *SimpleLogger) Infof(format string, args ...interface{}) {
	if l.level <= LevelInfo {
		fmt.Fprintf(os.Stdout, "[INFO] "+format+"\n", args...)
	}
}

// Warnf logs a warn message
func (l *SimpleLogger) Warnf(format string, args ...interface{}) {
	if l.level <= LevelWarn {
		fmt.Fprintf(os.Stderr, "[WARN] "+format+"\n", args...)
	}
}

// Errorf logs an error message
func (l *SimpleLogger) Errorf(format string, args ...interface{}) {
	if l.level <= LevelError {
		fmt.Fprintf(os.Stderr, "[ERROR] "+format+"\n", args...)
	}
}

// WithFields returns a logger with additional fields (stub for now)
func (l *SimpleLogger) WithFields(fields map[string]interface{}) Logger {
	// For simplicity, just return the same logger
	// In production, would create a structured logger with context
	return l
}

// ParseLevel parses a log level string
func ParseLevel(s string) LogLevel {
	switch s {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}
