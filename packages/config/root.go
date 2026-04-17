package config

import (
	"fmt"

	gokitconfig "github.com/raykavin/gokit/config"
)

// Root config

type appConfig struct {
	Application applicationConfig          `mapstructure:"application"`
	Databases   map[string]*databaseConfig `mapstructure:"databases"`
}

var _ ConfigProvider = (appConfig)(appConfig{})

func (c appConfig) GetApplication() ApplicationConfigProvider {
	return &c.Application
}

func (c appConfig) GetDatabase(name string) (DatabaseConfigProvider, error) {
	if db, ok := c.Databases[name]; ok {
		return db, nil
	}
	return nil, fmt.Errorf("database %q not found", name)
}

// Load reads the configuration file and returns a ConfigProvider.
func Load(path string) (ConfigProvider, error) {
	opts := gokitconfig.DefaultLoaderOptions[appConfig]()
	opts.ConfigPaths = []string{"./configs", "."}

	loader := gokitconfig.NewViper[appConfig](opts)

	if path != "" {
		loader.GetViper().SetConfigFile(path)
	}

	cfg, err := loader.Load()
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
