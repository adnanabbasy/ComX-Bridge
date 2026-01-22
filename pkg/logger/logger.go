package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Logger is a wrapper around slog.Logger to provide consistent logging across the application.
type Logger struct {
	*slog.Logger
}

// Config holds logger configuration.
type Config struct {
	Level  string // "debug", "info", "warn", "error"
	Format string // "text", "json"
	Output string // "stdout", "file"
	File   string // Path to log file
}

var globalLogger *Logger

// New creates a new Logger instance.
func New(config Config) *Logger {
	var handler slog.Handler
	var level slog.Level

	switch strings.ToLower(config.Level) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	// Output destination
	writer := os.Stdout
	if config.Output == "file" && config.File != "" {
		f, err := os.OpenFile(config.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			writer = f
		} else {
			// Fallback to stdout if file fails, maybe log this error?
			// fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
		}
	}

	if strings.ToLower(config.Format) == "json" {
		handler = slog.NewJSONHandler(writer, opts)
	} else {
		handler = slog.NewTextHandler(writer, opts)
	}

	l := &Logger{
		Logger: slog.New(handler),
	}

	// Set as global logger for simplicity if needed
	if globalLogger == nil {
		globalLogger = l
	}

	return l
}

// Global returns the global logger instance.
func Global() *Logger {
	if globalLogger == nil {
		// Default to info level, text format
		return New(Config{Level: "info", Format: "text"})
	}
	return globalLogger
}

// SetGlobal sets the global logger instance.
func SetGlobal(l *Logger) {
	globalLogger = l
}

// Helper methods for semantic logging if needed, or just use slog methods directly.
// We expose *slog.Logger directly via embedding, so methods like Info, Error are available.
