package config

import (
	"time"
)

type taskConfig struct {
	Enabled     bool      `mapstructure:"enabled"`
	Interval    string    `mapstructure:"interval"`
	FirstRun    bool      `mapstructure:"first_run"`
	RunOnce     bool      `mapstructure:"run_once"`
	QueueSize   uint32    `mapstructure:"queue_size"`
	StartAfter  time.Time `mapstructure:"start_after"`
	MaxAttempts int       `mapstructure:"max_attempts"`
}

var _ TaskConfigProvider = (taskConfig)(taskConfig{})

func (t taskConfig) IsEnabled() bool          { return t.Enabled }
func (t taskConfig) IsFirstRun() bool         { return t.FirstRun }
func (t taskConfig) IsRunOnce() bool          { return t.RunOnce }
func (t taskConfig) GetQueueSize() uint32     { return t.QueueSize }
func (t taskConfig) GetInterval() string      { return t.Interval }
func (t taskConfig) GetMaxAttempts() int      { return t.MaxAttempts }
func (t taskConfig) GetStartAfter() time.Time { return t.StartAfter }
