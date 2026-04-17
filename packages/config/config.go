package config

import "time"

// ConfigProvider is the main interface for accessing application configuration.
type ConfigProvider interface {
	GetApplication() ApplicationConfigProvider               // Returns the main application configuration
	GetDatabase(name string) (DatabaseConfigProvider, error) // Returns a database configuration by name
}

// ApplicationConfigProvider represents the main application configuration.
type ApplicationConfigProvider interface {
	GetName() string                                     // Returns the application name
	GetDescription() string                              // Returns the application description
	GetVersion() string                                  // Returns the application version
	GetLogLevel() string                                 // Returns the logging level (debug, info, warn, error)
	GetAPI() APIConfigProvider                           // Returns the api web server configuration
	GetACS() ACSConfigProvider                           // Returns the acs server configuration
	GetJWT() JWTConfigProvider                           // Returns the jwt configuration
	GetTask(taskName string) (TaskConfigProvider, error) // Returns a task configuration by name or error if not found
}

// APIConfigProvider defines the configuration for the web server.
type APIConfigProvider interface {
	GetListenPort() uint16          // Returns the port number the server listens on
	GetReadTimeout() time.Duration  // Returns the maximum duration for reading the entire request
	GetWriteTimeout() time.Duration // Returns the maximum duration for writing the response
	GetIdleTimeout() time.Duration  // Returns the maximum duration to keep idle connections open
	GetUseSSL() bool                // Indicates whether SSL/TLS is enabled
	GetSSLCert() string             // Returns the path to the SSL certificate file
	GetSSLKey() string              // Returns the path to the SSL private key file
	GetMaxPayloadSize() int64       // Returns the maximum allowed request payload size in bytes
	GetNoRouteTo() string           // Returns the URL to redirect when no route matches (404)
	GetCORS() map[string]string     // Returns the CORS headers configuration as key-value pairs
}

// ACSConfigProvider defines the configuration for acs
type ACSConfigProvider interface {
	GetListenPort() uint16            // Returns the CWMP server listen port
	GetUsername() string              // Returns the ACS username
	GetPassword() string              // Returns the ACS password
	GetURL() string                   // Returns the ACS configuration URL
	GetInformInterval() time.Duration // Returns the ACS inform interval
	GetSchemasDir() string            // Returns the path to the TR-069 parameter schema files directory
}

// JWTConfigProvider defines the configuration for the jwt
type JWTConfigProvider interface {
	GetSecret() string                  // Returns the secret value
	GetExpiresIn() time.Duration        // Returns the time expiration of token
	GetRefreshExpiresIn() time.Duration // Returns the time of expiration refresh token
}

// OIDCConfigProvider defines the configuration interface for OpenID Connect authentication.
type OIDCConfigProvider interface {
	GetRealmURL() string             // Returns the complete URL of the OIDC realm
	GetClientID() string             // Returns the client identifier registered with the OIDC provider
	GetClientSecret() string         // Returns the client secret with the OIDC provider
	GetSkipIssuerCheck() bool        // Indicates whether to skip validation of the token issuer
	GetSkipClientIDCheck() bool      // Indicates whether to skip validation of the client ID in token audience
	GetSkipExpiryCheck() bool        // Indicates whether to skip validation of token expiration
	GetCacheDuration() time.Duration // Returns the duration for caching OIDC configuration and keys
	GetHTTPTimeout() time.Duration   // Returns the timeout for HTTP requests to the OIDC provider
}

// TaskConfigProvider defines the configuration interface for background jobs.
type TaskConfigProvider interface {
	IsRunOnce() bool          // Indicates whether the job should execute only once
	IsFirstRun() bool         // Indicates whether the job should run immediately on application startup
	IsEnabled() bool          // Indicates whether the job is active and should be scheduled
	GetQueueSize() uint32     // Returns the maximum number of jobs that can be queued
	GetMaxAttempts() int      // Returns the max attempts of task
	GetInterval() string      // Returns the execution interval as a duration string (e.g., "5m", "1h")
	GetStartAfter() time.Time // Returns the time after which the job should start running
}

// DatabaseConfigProvider defines the interface for database configuration.
type DatabaseConfigProvider interface {
	GetName() string                   // Returns the name of database
	GetDialector() string              // Returns the database driver name (mysql, mariadb, postgres, sqlite, redis, mongodb)
	GetDSN() string                    // Returns the Data Source Name for the database connection
	GetURI() string                    // Returns the Uniform Resource Identifier for the database connection
	GetTTL() time.Duration             // Returns the TTL time
	GetLogLevel() string               // Returns the database log level (silent, info, warn, error)
	GetIdleConnsTime() time.Duration   // Returns the maximum amount of time a connection may be idle
	GetIdleMaxConns() int              // Returns the maximum number of idle connections in the pool
	GetMaxOpenConns() int              // Returns the maximum number of open connections to the database
	GetMigrationsPath() string         // Returns the file system path to database migration files
	GetPopulationPath() string         // Returns the file system path to database population files
	GetQueriesPath() map[string]string // Returns the file system paths to SQL query files
}
