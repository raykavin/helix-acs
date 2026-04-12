package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/raykavin/gokit/terminal"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	stdlogger "log"

	"github.com/raykavin/helix-acs/internal/api"
	"github.com/raykavin/helix-acs/internal/auth"
	"github.com/raykavin/helix-acs/internal/config"
	cwmpserver "github.com/raykavin/helix-acs/internal/cwmp"
	"github.com/raykavin/helix-acs/internal/device"
	"github.com/raykavin/helix-acs/internal/schema"
	"github.com/raykavin/helix-acs/internal/task"

	l "github.com/raykavin/helix-acs/internal/logger"

	"github.com/raykavin/gokit/logger"
)

// Database and queue identifiers used to look up named configurations.
const (
	storageDBName = "storage"
	cacheDBName   = "cache"
	queueTaskName = "queue"
)

// Connection retry policy applied to both MongoDB and Redis.
const (
	dbMaxRetries     = 3
	dbRetryInterval  = 2 * time.Second
	dbAttemptTimeout = 10 * time.Second
)

var configPath = flag.String("config", "./configs/config.yml", "path to config file (default: ./configs/config.yml)")

func main() {
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		stdlogger.Fatalf("failed to load config: %v\n", err)
	}

	appCfg := cfg.GetApplication()
	appLogger := initLogger(appCfg)

	displayAppBanner(appCfg)
	appLogger.Debug("Helix ACS starting...")

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGINT,
	)
	defer stop()

	if err := run(ctx, cfg, appLogger); err != nil {
		appLogger.WithError(err).Error("application finished with error")
		os.Exit(1)
	}

	appLogger.Info("Helix ACS stopped")
}

// run is the composition root: wires all dependencies and starts the servers.
func run(ctx context.Context, cfg config.ConfigProvider, appLogger l.Logger) error {
	storageDB, err := connectStorage(cfg, appLogger)
	if err != nil {
		return err
	}
	defer disconnectStorageDB(storageDB, appLogger)

	cacheDB, err := connectCache(cfg, appLogger)
	if err != nil {
		return err
	}
	defer disconnectCacheDB(cacheDB, appLogger)

	deviceSvc, err := initDeviceService(ctx, storageDB, appLogger, cfg)
	if err != nil {
		return err
	}

	tsk, err := cfg.GetApplication().GetTask(queueTaskName)
	if err != nil {
		return fmt.Errorf("unable to find configuration for queue task %q", queueTaskName)
	}

	appCfg := cfg.GetApplication()
	acsConfig := appCfg.GetACS()
	apiConfig := appCfg.GetAPI()
	cacheCC, _ := cfg.GetDatabase(cacheDBName)

	jwtSvc := initJWTService(appCfg.GetJWT())
	taskQueue := initTaskQueue(cacheDB, appLogger, cacheCC.GetTTL(), tsk.GetMaxAttempts())
	schemaReg := initSchemaRegistry(acsConfig.GetSchemasDir(), appLogger)
	cwmpSrv := initCWMPServer(deviceSvc, taskQueue, acsConfig, appLogger, schemaReg)

	routerCfg := api.Config{
		CORS:        apiConfig.GetCORS(),
		MaxAttempts: tsk.GetMaxAttempts(),
		ACSUsername: acsConfig.GetUsername(),
		ACSPassword: acsConfig.GetPassword(),
	}

	apiRouter := api.NewRouter(
		deviceSvc,
		taskQueue,
		jwtSvc,
		storageDB,
		cacheDB,
		appLogger,
		routerCfg,
	)

	return serveHTTP(ctx, apiConfig, acsConfig, cwmpSrv, apiRouter, appLogger)
}

// connectStorage validates config and connects to MongoDB.
func connectStorage(cfg config.ConfigProvider, appLogger l.Logger) (*mongo.Client, error) {
	stg, err := cfg.GetDatabase(storageDBName)
	if err != nil {
		return nil, fmt.Errorf("unable to find configuration for storage database %q", storageDBName)
	}
	return initStorageDB(stg, appLogger)
}

// connectCache validates config and connects to Redis.
func connectCache(cfg config.ConfigProvider, appLogger l.Logger) (*redis.Client, error) {
	cc, err := cfg.GetDatabase(cacheDBName)
	if err != nil {
		return nil, fmt.Errorf("unable to find configuration for cache database %q", cacheDBName)
	}
	return initCacheDB(cc, appLogger)
}

// initStorageDB connects to MongoDB and returns the client.
func initStorageDB(cfg config.DatabaseConfigProvider, appLogger l.Logger) (*mongo.Client, error) {
	uri := cfg.GetURI()
	client, err := connectMongoDB(uri, appLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to storage database: %w", err)
	}
	appLogger.WithField("uri", uri).Debug("Connected to storage database")
	return client, nil
}

// disconnectStorageDB gracefully closes the MongoDB connection.
func disconnectStorageDB(client *mongo.Client, appLogger l.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Disconnect(ctx); err != nil {
		appLogger.WithError(err).Error("error disconnecting from MongoDB")
	}
}

// initCacheDB connects to Redis and returns the client.
func initCacheDB(cfg config.DatabaseConfigProvider, appLogger l.Logger) (*redis.Client, error) {
	uri := cfg.GetURI()
	client, err := connectRedis(uri, appLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}
	appLogger.WithField("uri", uri).Debug("Connected to cache database")
	return client, nil
}

// disconnectCacheDB gracefully closes the Redis connection.
func disconnectCacheDB(client *redis.Client, appLogger l.Logger) {
	if err := client.Close(); err != nil {
		appLogger.WithError(err).Error("error closing Redis connection")
	}
}

// initDeviceService creates the device repository and service.
// A 30-second startup context is used to bound index creation and other
// one-off setup operations.
func initDeviceService(ctx context.Context, mongoClient *mongo.Client, appLogger l.Logger, cfg config.ConfigProvider) (device.Service, error) {
	stg, _ := cfg.GetDatabase(storageDBName) // already validated by caller
	dbName := stg.GetName()

	startupCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	db := mongoClient.Database(dbName)
	repo, err := device.NewMongoRepository(startupCtx, db)
	if err != nil {
		return nil, fmt.Errorf("failed to create device repository: %w", err)
	}
	return device.NewService(repo, appLogger), nil
}

// initTaskQueue creates the Redis-backed task queue.
func initTaskQueue(redisClient *redis.Client, appLogger l.Logger, ttl time.Duration, maxAttempts int) *task.RedisQueue {
	return task.NewRedisQueue(redisClient, ttl, maxAttempts)
}

// initJWTService creates the JWT authentication service.
func initJWTService(cfg config.JWTConfigProvider) *auth.JWTService {
	return auth.NewJWTService(cfg.GetSecret(), cfg.GetExpiresIn(), cfg.GetRefreshExpiresIn())
}

// initSchemaRegistry loads TR-069 parameter schemas from the ./schemas directory.
// On failure it logs a warning and returns an empty registry so the system falls
// back to the built-in Go mappers.
func initSchemaRegistry(schemasDir string, appLogger l.Logger) *schema.Registry {
	reg := schema.NewRegistry()
	if err := reg.LoadDir(schemasDir); err != nil {
		appLogger.WithError(err).
			WithField("dir", schemasDir).
			Warn("Failed to load schemas falling back to built-in mappers")
		return reg
	}
	appLogger.WithField("dir", schemasDir).Info("Loaded TR-069 parameter schemas")
	return reg
}

// initCWMPServer builds the CWMP handler and returns the server.
func initCWMPServer(deviceSvc device.Service, taskQueue *task.RedisQueue, acs config.ACSConfigProvider, appLogger l.Logger, schemaReg *schema.Registry) *cwmpserver.Server {
	handler := cwmpserver.NewHandler(
		deviceSvc,
		taskQueue,
		appLogger,
		acs.GetUsername(),
		acs.GetPassword(),
		acs.GetURL(),
		acs.GetInformInterval(),
		schemaReg,
	)
	digestAuth := auth.NewDigestAuth(appLogger, "ACS", acs.GetUsername(), acs.GetPassword())
	return cwmpserver.NewServer(handler, digestAuth, appLogger)
}

// serveHTTP starts both HTTP servers and blocks until ctx is cancelled or a
// server error occurs. Both servers are shut down gracefully on exit.
func serveHTTP(
	ctx context.Context,
	apiCfg config.APIConfigProvider,
	acsCfg config.ACSConfigProvider,
	cwmpSrv *cwmpserver.Server,
	apiRouter http.Handler,
	appLogger l.Logger,
) error {
	cwmpHTTP := newHTTPServer(fmt.Sprintf(":%d", acsCfg.GetListenPort()), cwmpSrv.Router())
	apiHTTP := newHTTPServer(fmt.Sprintf(":%d", apiCfg.GetListenPort()), apiRouter)

	serverErr := make(chan error, 2)

	go startServer(cwmpHTTP, "CWMP", appLogger, serverErr)
	go startServer(apiHTTP, "API", appLogger, serverErr)

	select {
	case <-ctx.Done():
		appLogger.Info("shutdown signal received")
	case err := <-serverErr:
		appLogger.WithError(err).Error("server error")
	}

	return shutdownServers(appLogger, cwmpHTTP, apiHTTP)
}

// newHTTPServer returns an http.Server with conservative production timeouts.
func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// startServer runs ListenAndServe in a goroutine and forwards non-closed errors
// to errCh.
func startServer(srv *http.Server, name string, appLogger l.Logger, errCh chan<- error) {
	appLogger.WithField("addr", srv.Addr).Infof("%s server listening", name)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		errCh <- fmt.Errorf("%s server: %w", name, err)
	}
}

// shutdownServers attempts a graceful shutdown of all provided servers within a
// 30-second window. Shutdown errors are logged but not returned; the function
// always returns nil so the caller can exit cleanly.
func shutdownServers(appLogger l.Logger, servers ...*http.Server) error {
	appLogger.Info("shutting down servers (30s timeout)")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, srv := range servers {
		if err := srv.Shutdown(ctx); err != nil {
			appLogger.WithError(err).Errorf("error shutting down server on %s", srv.Addr)
		}
	}
	return nil
}

// connectMongoDB dials MongoDB with up to dbMaxRetries attempts.
func connectMongoDB(uri string, appLogger l.Logger) (*mongo.Client, error) {
	var lastErr error
	for attempt := 1; attempt <= dbMaxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), dbAttemptTimeout)
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
		cancel()
		if err != nil {
			lastErr = err
			appLogger.WithError(err).WithField("attempt", attempt).Warn("MongoDB connect failed, retrying")
			time.Sleep(dbRetryInterval)
			continue
		}

		pingCtx, pingCancel := context.WithTimeout(context.Background(), dbAttemptTimeout)
		err = client.Ping(pingCtx, nil)
		pingCancel()
		if err != nil {
			lastErr = err
			_ = client.Disconnect(context.Background())
			appLogger.WithError(err).WithField("attempt", attempt).Warn("MongoDB ping failed, retrying")
			time.Sleep(dbRetryInterval)
			continue
		}

		return client, nil
	}
	return nil, fmt.Errorf("mongodb: failed after %d attempts: %w", dbMaxRetries, lastErr)
}

// connectRedis parses the Redis URL and verifies connectivity with retries.
func connectRedis(uri string, appLogger l.Logger) (*redis.Client, error) {
	opts, err := redis.ParseURL(uri)
	if err != nil {
		return nil, fmt.Errorf("redis: invalid URI %q: %w", uri, err)
	}

	client := redis.NewClient(opts)

	var lastErr error
	for attempt := 1; attempt <= dbMaxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), dbAttemptTimeout)
		err := client.Ping(ctx).Err()
		cancel()
		if err == nil {
			return client, nil
		}
		lastErr = err
		appLogger.WithError(err).WithField("attempt", attempt).Warn("Redis ping failed, retrying")
		time.Sleep(dbRetryInterval)
	}

	_ = client.Close()
	return nil, fmt.Errorf("redis: failed after %d attempts: %w", dbMaxRetries, lastErr)
}

// initLogger initializes the application logger from config.
func initLogger(cfg config.ApplicationConfigProvider) *l.LoggerWrapper {
	gkLogger, err := logger.New(&logger.Config{
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

// displayAppBanner prints the application banner and metadata to the terminal.
func displayAppBanner(cfg config.ApplicationConfigProvider) {
	terminal.PrintBanner(cfg.GetName())
	terminal.PrintText(cfg.GetDescription())
	// terminal.PrintText("-> EchoSys")
	terminal.PrintText(fmt.Sprintf("Copyright (c) %d EchoSys, All rights reserved!", time.Now().Year()))
	terminal.PrintHeader(fmt.Sprintf("Version: %s", cfg.GetVersion()))
}
