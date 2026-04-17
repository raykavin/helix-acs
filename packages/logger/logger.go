package logger

import (
	"context"
	"time"
)

// Logger defines the core logging interface for standard logging operations.
// It provides methods for structured logging with fields and error context,
// as well as standard leveled logging with both plain and formatted variants.
type Logger interface {
	// WithField returns a new Logger instance with the specified key-value pair added to the logging context.
	// This enables structured logging by attaching additional metadata to all subsequent log entries.
	WithField(key string, value any) Logger

	// WithFields returns a new Logger instance with multiple key-value pairs added to the logging context.
	// The provided fields map will be merged with any existing context fields.
	WithFields(fields map[string]any) Logger

	// WithError returns a new Logger instance with error context added.
	// The error details will be included in all subsequent log entries, typically under an "error" key.
	WithError(err error) Logger

	// Standard log functions - output plain messages at various log levels

	// Print logs a message at the default log level (typically INFO)
	Print(args ...any)

	// Debug logs a message at DEBUG level for detailed troubleshooting information
	Debug(args ...any)

	// Info logs a message at INFO level for general operational information
	Info(args ...any)

	// Warn logs a message at WARN level for potentially harmful situations
	Warn(args ...any)

	// Error logs a message at ERROR level for error conditions that should be investigated
	Error(args ...any)

	// Fatal logs a message at FATAL level then calls os.Exit(1)
	Fatal(args ...any)

	// Panic logs a message at PANIC level then panics
	Panic(args ...any)

	// Formatted log functions - output formatted messages at various log levels

	// Printf logs a formatted message at the default log level
	Printf(format string, args ...any)

	// Debugf logs a formatted message at DEBUG level
	Debugf(format string, args ...any)

	// Infof logs a formatted message at INFO level
	Infof(format string, args ...any)

	// Warnf logs a formatted message at WARN level
	Warnf(format string, args ...any)

	// Errorf logs a formatted message at ERROR level
	Errorf(format string, args ...any)

	// Fatalf logs a formatted message at FATAL level then calls os.Exit(1)
	Fatalf(format string, args ...any)

	// Panicf logs a formatted message at PANIC level then panics
	Panicf(format string, args ...any)
}

// Observability extends the Logger interface with specialized methods for observability and monitoring.
// It provides enhanced logging capabilities for application performance monitoring, API tracking,
// and operational intelligence in distributed systems.
type Observability interface {
	// Embedded Logger interface provides all standard logging capabilities
	Logger

	// Success logs a success event with structured context for tracking positive outcomes
	Success(msg string)

	// Failure logs a failure event with structured context for tracking negative outcomes
	Failure(msg string)

	// Benchmark logs performance benchmark data for tracking operation durations
	Benchmark(name string, duration time.Duration)

	// API logs API request/response details including method, path, IP, status code, and duration
	// This is particularly useful for monitoring REST APIs and web services
	API(method, path, ipAddress string, statusCode int, duration time.Duration)

	// WithContext returns a new Observability instance with the given context.Context.
	// This enables context-aware logging with trace IDs, span IDs, and other distributed tracing information.
	WithContext(ctx context.Context) Observability
}
