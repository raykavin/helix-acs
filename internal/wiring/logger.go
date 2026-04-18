package wiring

import (
	"fmt"
	stdlogger "log"
	"time"

	gklogger "github.com/raykavin/gokit/logger"
	"github.com/raykavin/gokit/terminal"

	"github.com/raykavin/helix-acs/internal/config"
	l "github.com/raykavin/helix-acs/internal/logger"
)

// NewLogger initializes the application logger from config.
func NewLogger(cfg config.ApplicationConfigProvider) *l.LoggerWrapper {
	gkLogger, err := gklogger.New(&gklogger.Config{
		Level:          cfg.GetLogLevel(),
		DateTimeLayout: "2006-01-02 15:04:05",
		Colored:        true,
		JSONFormat:     false,
		UseEmoji:       false,
	})
	if err != nil {
		stdlogger.Fatalf("failed to initialize logger: %v\n", err)
	}
	return l.NewLoggerWrapper(gkLogger)
}

// DisplayBanner prints the application banner and metadata to the terminal.
func DisplayBanner(cfg config.ApplicationConfigProvider) {
	if err := terminal.PrintBanner(cfg.GetName()); err != nil {
		stdlogger.Printf("warning: could not print banner: %v\n", err)
	}
	terminal.PrintText(cfg.GetDescription())
	terminal.PrintText(fmt.Sprintf("Copyright (c) %d EchoSys, All rights reserved!", time.Now().Year()))
	terminal.PrintHeader(fmt.Sprintf("Version: %s", cfg.GetVersion()))
}
