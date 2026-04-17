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

	stdlogger "log"

	"github.com/raykavin/gokit/terminal"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/raykavin/gokit/logger"

	apiserver "github.com/raykavin/helix-acs/apps/api/internal"
	"github.com/raykavin/helix-acs/packages/auth"
	"github.com/raykavin/helix-acs/packages/config"
	"github.com/raykavin/helix-acs/packages/device"
	l "github.com/raykavin/helix-acs/packages/logger"
	"github.com/raykavin/helix-acs/packages/task"
)

const (
	storageDBName = "storage"
	cacheDBName   = "cache"
	queueTaskName = "queue"
)

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
	appLogger.Debug("Helix ACS API starting...")

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

	appLogger.Info("Helix ACS API stopped")
}

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

	routerCfg := apiserver.Config{
		CORS:        apiConfig.GetCORS(),
		MaxAttempts: tsk.GetMaxAttempts(),
		ACSUsername: acsConfig.GetUsername(),
		ACSPassword: acsConfig.GetPassword(),
	}

	apiRouter := apiserver.NewRouter(
		deviceSvc,
		taskQueue,
		jwtSvc,
		storageDB,
		cacheDB,
		appLogger,
		routerCfg,
	)

	return serveHTTP(ctx, apiConfig, apiRouter, appLogger)
}

func connectStorage(cfg config.ConfigProvider, appLogger l.Logger) (*mongo.Client, error) {
	stg, err := cfg.GetDatabase(storageDBName)
	if err != nil {
		return nil, fmt.Errorf("unable to find configuration for storage database %q", storageDBName)
	}
	return initStorageDB(stg, appLogger)
}

func connectCache(cfg config.ConfigProvider, appLogger l.Logger) (*redis.Client, error) {
	cc, err := cfg.GetDatabase(cacheDBName)
	if err != nil {
		return nil, fmt.Errorf("unable to find configuration for cache database %q", cacheDBName)
	}
	return initCacheDB(cc, appLogger)
}

func initStorageDB(cfg config.DatabaseConfigProvider, appLogger l.Logger) (*mongo.Client, error) {
	uri := cfg.GetURI()
	client, err := connectMongoDB(uri, appLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to storage database: %w", err)
	}
	appLogger.WithField("uri", uri).Debug("Connected to storage database")
	return client, nil
}

func disconnectStorageDB(client *mongo.Client, appLogger l.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Disconnect(ctx); err != nil {
		appLogger.WithError(err).Error("error disconnecting from MongoDB")
	}
}

func initCacheDB(cfg config.DatabaseConfigProvider, appLogger l.Logger) (*redis.Client, error) {
	uri := cfg.GetURI()
	client, err := connectRedis(uri, appLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}
	appLogger.WithField("uri", uri).Debug("Connected to cache database")
	return client, nil
}

func disconnectCacheDB(client *redis.Client, appLogger l.Logger) {
	if err := client.Close(); err != nil {
		appLogger.WithError(err).Error("error closing Redis connection")
	}
}

func initDeviceService(ctx context.Context, mongoClient *mongo.Client, appLogger l.Logger, cfg config.ConfigProvider) (device.Service, error) {
	stg, _ := cfg.GetDatabase(storageDBName)
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

func initTaskQueue(redisClient *redis.Client, appLogger l.Logger, ttl time.Duration, maxAttempts int) *task.RedisQueue {
	_ = appLogger
	return task.NewRedisQueue(redisClient, ttl, maxAttempts)
}

func initJWTService(cfg config.JWTConfigProvider) *auth.JWTService {
	return auth.NewJWTService(cfg.GetSecret(), cfg.GetExpiresIn(), cfg.GetRefreshExpiresIn())
}

func serveHTTP(
	ctx context.Context,
	apiCfg config.APIConfigProvider,
	apiRouter http.Handler,
	appLogger l.Logger,
) error {
	apiHTTP := newHTTPServer(fmt.Sprintf(":%d", apiCfg.GetListenPort()), apiRouter)

	serverErr := make(chan error, 1)
	go startServer(apiHTTP, "API", appLogger, serverErr)

	select {
	case <-ctx.Done():
		appLogger.Info("shutdown signal received")
	case err := <-serverErr:
		appLogger.WithError(err).Error("server error")
	}

	return shutdownServers(appLogger, apiHTTP)
}

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

func startServer(srv *http.Server, name string, appLogger l.Logger, errCh chan<- error) {
	appLogger.WithField("addr", srv.Addr).Infof("%s server listening", name)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		errCh <- fmt.Errorf("%s server: %w", name, err)
	}
}

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

func displayAppBanner(cfg config.ApplicationConfigProvider) {
	terminal.PrintBanner(cfg.GetName())
	terminal.PrintText(cfg.GetDescription())
	terminal.PrintText(fmt.Sprintf("Copyright (c) %d EchoSys, All rights reserved!", time.Now().Year()))
	terminal.PrintHeader(fmt.Sprintf("Version: %s", cfg.GetVersion()))
}
