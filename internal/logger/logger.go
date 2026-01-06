package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// Logger defines the logging interface
type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
}

// Level represents logging level
type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

// StandardLogger implements Logger using standard library
type StandardLogger struct {
	level  Level
	logger *log.Logger
}

// New creates a new logger with the specified level
func New(levelStr string) Logger {
	level := parseLevel(levelStr)

	return &StandardLogger{
		level:  level,
		logger: log.New(os.Stdout, "", 0),
	}
}

// Debug logs a debug message
func (l *StandardLogger) Debug(msg string, keysAndValues ...interface{}) {
	if l.level <= DebugLevel {
		l.log("DEBUG", msg, keysAndValues...)
	}
}

// Info logs an info message
func (l *StandardLogger) Info(msg string, keysAndValues ...interface{}) {
	if l.level <= InfoLevel {
		l.log("INFO", msg, keysAndValues...)
	}
}

// Warn logs a warning message
func (l *StandardLogger) Warn(msg string, keysAndValues ...interface{}) {
	if l.level <= WarnLevel {
		l.log("WARN", msg, keysAndValues...)
	}
}

// Error logs an error message
func (l *StandardLogger) Error(msg string, keysAndValues ...interface{}) {
	if l.level <= ErrorLevel {
		l.log("ERROR", msg, keysAndValues...)
	}
}

// log formats and writes a log message
func (l *StandardLogger) log(level, msg string, keysAndValues ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	
	// Format key-value pairs
	var kvPairs []string
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			key := keysAndValues[i]
			value := keysAndValues[i+1]
			kvPairs = append(kvPairs, fmt.Sprintf("%v=%v", key, value))
		}
	}

	var kvStr string
	if len(kvPairs) > 0 {
		kvStr = " " + strings.Join(kvPairs, " ")
	}

	l.logger.Printf("[%s] %s: %s%s", timestamp, level, msg, kvStr)
}

// parseLevel converts a string to a log level
func parseLevel(levelStr string) Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn", "warning":
		return WarnLevel
	case "error":
		return ErrorLevel
	default:
		return InfoLevel
	}
}
