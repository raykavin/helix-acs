package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Init(level string, pretty bool) zerolog.Logger {
	var w io.Writer = os.Stdout
	if pretty {
		w = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	}

	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(lvl)
	logger := zerolog.New(w).With().Timestamp().Logger()
	log.Logger = logger
	return logger
}

func With(logger zerolog.Logger, key string, val any) zerolog.Logger {
	return logger.With().Interface(key, val).Logger()
}
