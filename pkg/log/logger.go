package log

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
)

var (
	logger *slog.Logger
	mu     sync.RWMutex
)

// ParseLogLevel converts a string log level to a slog.Level.
// Valid values are "debug", "info", "warn", "error".
// If an invalid value is provided, it defaults to debug.
func ParseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		// Default to debug for unknown levels
		return slog.LevelDebug
	}
}

// InitLog initializes or reinitializes the logger with the specified log level.
// This can be called multiple times to change the log level at runtime.
// It will override any previously configured logger instance.
func InitLog(logLevel string) {
	level := ParseLogLevel(logLevel)

	mu.Lock()
	defer mu.Unlock()

	// Always create a new logger instance (override existing)
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	logger = slog.New(handler)
}

// GetLog returns the slog.Logger instance configured for the application.
// The logger emits JSON-formatted logs at the configured level to stdout.
// This format is easy to parse both by humans and log aggregation tools
// while still being structured.
// If the logger hasn't been initialized yet, it defaults to info level.
func GetLog() *slog.Logger {
	mu.RLock()
	if logger != nil {
		defer mu.RUnlock()
		return logger
	}
	mu.RUnlock()

	// Logger not initialized, create default one
	mu.Lock()
	defer mu.Unlock()

	// Double-check after acquiring write lock
	if logger == nil {
		handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
		logger = slog.New(handler)
	}

	return logger
}

// Convenience wrappers ------------------------------------------------------
// The following helpers allow the rest of the codebase to keep using familiar
// Printf/Fatalf style helpers while internally switching to structured slog.
// This drastically reduces the amount of refactoring required while migrating
// to slog. New code is encouraged to use the structured methods (Info, Debug,
// Warn, Error, etc.) directly.

// Debug logs a message at Debug level.
func Debug(msg string, args ...any) { GetLog().Debug(msg, args...) }

// Info logs a message at Info level.
func Info(msg string, args ...any) { GetLog().Info(msg, args...) }

// Warn logs a message at Warn level.
func Warn(msg string, args ...any) { GetLog().Warn(msg, args...) }

// Error logs a message at Error level.
func Error(msg string, args ...any) { GetLog().Error(msg, args...) }

// Printf is a drop-in replacement for log.Printf using Info as the log level.
func Printf(format string, args ...any) {
	GetLog().Debug(fmt.Sprintf(format, args...))
}

// Fatalf logs a formatted message and exits.
func Fatalf(format string, args ...any) {
	GetLog().Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}

// Fatal logs a message with optional arguments and exits.
func Fatal(args ...any) {
	GetLog().Error(fmt.Sprint(args...))
	os.Exit(1)
}

// Errorf creates an error with a formatted message.
// This is a drop-in replacement for fmt.Errorf.
func Errorf(format string, args ...any) error {
	Error(format, args...)
	return fmt.Errorf(format, args...)
}
