package config

import "time"

type acsConfig struct {
	ListenPort     uint16        `mapstructure:"listen_port"`
	InformInterval time.Duration `mapstructure:"inform_interval"`
	Username       string        `mapstructure:"username"`
	Password       string        `mapstructure:"password"`
	URL            string        `mapstructure:"url"`
	SchemasDir     string        `mapstructure:"schemas_dir"`
}

var _ ACSConfigProvider = (acsConfig)(acsConfig{})

// GetListenPort implements [ACSConfigProvider].
func (a acsConfig) GetListenPort() uint16 { return a.ListenPort }

// GetInformInterval implements [ACSConfigProvider].
func (a acsConfig) GetInformInterval() time.Duration { return a.InformInterval }

// GetUsername implements [ACSConfigProvider].
func (a acsConfig) GetUsername() string { return a.Username }

// GetPassword implements [ACSConfigProvider].
func (a acsConfig) GetPassword() string { return a.Password }

// GetURL implements [ACSConfigProvider].
func (a acsConfig) GetURL() string { return a.URL }

// GetSchemasDir implements [ACSConfigProvider].
// Returns the path to the schemas directory, defaulting to "./schemas".
func (a acsConfig) GetSchemasDir() string {
	if a.SchemasDir == "" {
		return "./schemas"
	}
	return a.SchemasDir
}
