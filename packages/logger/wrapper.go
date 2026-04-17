package logger

import (
	"fmt"

	gklogger "github.com/raykavin/gokit/logger"
)

// LoggerWrapper is an adapter that wraps our custom Logger to implement the gklogger.Logger interface.
// This allows us to use our Logger with libraries that expect a gklogger.Logger without modifying their code.
type LoggerWrapper struct {
	logger *gklogger.Zerolog
}

var _ Logger = (*LoggerWrapper)(nil)

func NewLoggerWrapper(l *gklogger.Zerolog) *LoggerWrapper {
	return &LoggerWrapper{logger: l}
}

// Debug implements [Logger].
func (l *LoggerWrapper) Debug(args ...any) {
	l.logger.Debug().Msg(fmt.Sprint(args...))
}

// Debugf implements [Logger].
func (l *LoggerWrapper) Debugf(format string, args ...any) {
	l.logger.Debug().Msgf(format, args...)
}

// Error implements [Logger].
func (l *LoggerWrapper) Error(args ...any) {
	l.logger.Error().Msg(fmt.Sprint(args...))
}

// Errorf implements [Logger].
func (l *LoggerWrapper) Errorf(format string, args ...any) {
	l.logger.Error().Msgf(format, args...)
}

// Fatal implements [Logger].
func (l *LoggerWrapper) Fatal(args ...any) {
	l.logger.Fatal().Msg(fmt.Sprint(args...))
}

// Fatalf implements [Logger].
func (l *LoggerWrapper) Fatalf(format string, args ...any) {
	l.logger.Fatal().Msgf(format, args...)
}

// Info implements [Logger].
func (l *LoggerWrapper) Info(args ...any) {
	l.logger.Info().Msg(fmt.Sprint(args...))
}

// Infof implements [Logger].
func (l *LoggerWrapper) Infof(format string, args ...any) {
	l.logger.Info().Msgf(format, args...)
}

// Panic implements [Logger].
func (l *LoggerWrapper) Panic(args ...any) {
	l.logger.Panic().Msg(fmt.Sprint(args...))
}

// Panicf implements [Logger].
func (l *LoggerWrapper) Panicf(format string, args ...any) {
	l.logger.Panic().Msgf(format, args...)
}

// Print implements [Logger].
func (l *LoggerWrapper) Print(args ...any) {
	l.logger.Print(args...)
}

// Printf implements [Logger].
func (l *LoggerWrapper) Printf(format string, args ...any) {
	l.logger.Printf(format, args...)
}

// Warn implements [Logger].
func (l *LoggerWrapper) Warn(args ...any) {
	l.logger.Warn().Msg(fmt.Sprint(args...))
}

// Warnf implements [Logger].
func (l *LoggerWrapper) Warnf(format string, args ...any) {
	l.logger.Warn().Msgf(format, args...)
}

// WithError implements [Logger].
func (l *LoggerWrapper) WithError(err error) Logger {
	newLogger := l.logger.With().Err(err).Logger()
	return &LoggerWrapper{&gklogger.Zerolog{Logger: &newLogger}}
}

// WithField implements [Logger].
func (l *LoggerWrapper) WithField(key string, value any) Logger {
	newLogger := l.logger.With().Interface(key, value).Logger()
	return &LoggerWrapper{&gklogger.Zerolog{Logger: &newLogger}}
}

// WithFields implements [Logger].
func (l *LoggerWrapper) WithFields(fields map[string]any) Logger {
	newLogger := l.logger.With().Fields(fields).Logger()
	return &LoggerWrapper{&gklogger.Zerolog{Logger: &newLogger}}
}
