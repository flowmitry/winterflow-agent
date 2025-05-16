package log

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
)

var (
	logger *slog.Logger
	once   sync.Once
)

// GetLog returns the singleton slog.Logger instance configured for the application.
// The logger is initialised once in a thread-safe manner, emitting JSON-formatted logs
// at the Debug level to stdout. This format is easy to parse both by humans and log
// aggregation tools while still being structured.
func GetLog() *slog.Logger {
	once.Do(func() {
		// Create a JSON handler that writes to stdout. Adjust options here if you
		// need to lower the default log level or change the output destination.
		handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
		logger = slog.New(handler)
	})
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
	GetLog().Info(fmt.Sprintf(format, args...))
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
