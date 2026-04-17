package config

import (
	"fmt"
)

type applicationConfig struct {
	Name        string                 `mapstructure:"name"`
	Description string                 `mapstructure:"description"`
	Version     string                 `mapstructure:"version"`
	LogLevel    string                 `mapstructure:"log_level"`
	JWT         jwtConfig              `mapstructure:"jwt"`
	ACS         acsConfig              `mapstructure:"acs"`
	API         apiConfig              `mapstructure:"web"`
	Tasks       map[string]*taskConfig `mapstructure:"tasks"`
}

var _ ApplicationConfigProvider = (applicationConfig)(applicationConfig{})

func (a applicationConfig) GetName() string           { return a.Name }
func (a applicationConfig) GetDescription() string    { return a.Description }
func (a applicationConfig) GetVersion() string        { return a.Version }
func (a applicationConfig) GetLogLevel() string       { return a.LogLevel }
func (a applicationConfig) GetJWT() JWTConfigProvider { return a.JWT }
func (a applicationConfig) GetAPI() APIConfigProvider { return a.API }
func (a applicationConfig) GetACS() ACSConfigProvider { return a.ACS }

func (a applicationConfig) GetTask(taskName string) (TaskConfigProvider, error) {
	if t, ok := a.Tasks[taskName]; ok {
		return t, nil
	}
	return nil, fmt.Errorf("task %q not found", taskName)
}
