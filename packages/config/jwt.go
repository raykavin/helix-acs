package config

import "time"

type jwtConfig struct {
	Secret           string        `mapstructure:"secret"`
	ExpiresIn        time.Duration `mapstructure:"expires_in"`
	RefreshExpiresIn time.Duration `mapstructure:"refresh_expires_in"`
}

var _ JWTConfigProvider = (jwtConfig)(jwtConfig{})

// GetExpiresIn implements [JWTConfigProvider].
func (j jwtConfig) GetExpiresIn() time.Duration { return j.ExpiresIn }

// GetRefreshExpiresIn implements [JWTConfigProvider].
func (j jwtConfig) GetRefreshExpiresIn() time.Duration { return j.RefreshExpiresIn }

// GetSecret implements [JWTConfigProvider].
func (j jwtConfig) GetSecret() string { return j.Secret }
