package config

import (
	"time"
)

type databaseConfig struct {
	Name           string            `mapstructure:"name"`
	Dialector      string            `mapstructure:"dialector"`
	DSN            string            `mapstructure:"dsn"`
	URI            string            `mapstructure:"uri"`
	Database       string            `mapstructure:"database"`
	LogLevel       string            `mapstructure:"log_level"`
	IdleMaxConns   int               `mapstructure:"idle_max_conns"`
	MaxOpenConns   int               `mapstructure:"max_open_conns"`
	IdleConnsTime  time.Duration     `mapstructure:"idle_conns_time"`
	TTL            time.Duration     `mapstructure:"ttl"`
	MigrationsPath string            `mapstructure:"migrations_path"`
	PopulationPath string            `mapstructure:"population_path"`
	QueriesPath    map[string]string `mapstructure:"queries_path"`
}

var _ DatabaseConfigProvider = (databaseConfig)(databaseConfig{})

func (d databaseConfig) GetName() string                   { return d.Name }
func (d databaseConfig) GetDialector() string              { return d.Dialector }
func (d databaseConfig) GetLogLevel() string               { return d.LogLevel }
func (d databaseConfig) GetIdleConnsTime() time.Duration   { return d.IdleConnsTime }
func (d databaseConfig) GetIdleMaxConns() int              { return d.IdleMaxConns }
func (d databaseConfig) GetMaxOpenConns() int              { return d.MaxOpenConns }
func (d databaseConfig) GetMigrationsPath() string         { return d.MigrationsPath }
func (d databaseConfig) GetPopulationPath() string         { return d.PopulationPath }
func (d databaseConfig) GetQueriesPath() map[string]string { return d.QueriesPath }
func (d databaseConfig) GetTTL() time.Duration             { return d.TTL }
func (d databaseConfig) GetDSN() string                    { return d.DSN }
func (d databaseConfig) GetURI() string                    { return d.URI }
