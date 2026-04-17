package config

import (
	"time"
)

type apiConfig struct {
	ListenPort     uint16            `mapstructure:"listen_port"`
	UseSSL         bool              `mapstructure:"use_ssl"`
	Cert           string            `mapstructure:"crt"`
	Key            string            `mapstructure:"key"`
	MaxPayloadSize int64             `mapstructure:"max_payload_size"`
	WriteTimeout   time.Duration     `mapstructure:"write_timeout"`
	ReadTimeout    time.Duration     `mapstructure:"read_timeout"`
	IdleTimeout    time.Duration     `mapstructure:"idle_timeout"`
	NoRouteTo      string            `mapstructure:"no_router"`
	AllowedOrigins []string          `mapstructure:"allowed_origins"`
	AllowedMethods []string          `mapstructure:"allowed_methods"`
	CORS           map[string]string `mapstructure:"cors"`
}

var _ APIConfigProvider = (apiConfig)(apiConfig{})

func (w apiConfig) GetListenPort() uint16          { return w.ListenPort }
func (w apiConfig) GetUseSSL() bool                { return w.UseSSL }
func (w apiConfig) GetSSLCert() string             { return w.Cert }
func (w apiConfig) GetSSLKey() string              { return w.Key }
func (w apiConfig) GetMaxPayloadSize() int64       { return w.MaxPayloadSize }
func (w apiConfig) GetWriteTimeout() time.Duration { return w.WriteTimeout }
func (w apiConfig) GetReadTimeout() time.Duration  { return w.ReadTimeout }
func (w apiConfig) GetIdleTimeout() time.Duration  { return w.IdleTimeout }
func (w apiConfig) GetNoRouteTo() string           { return w.NoRouteTo }
func (w apiConfig) GetCORS() map[string]string     { return w.CORS }
func (w apiConfig) GetAllowedMethods() []string    { return w.AllowedMethods }
func (w apiConfig) GetAllowedOrigins() []string    { return w.AllowedOrigins }
