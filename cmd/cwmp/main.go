package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	stdlogger "log"

	"github.com/raykavin/helix-acs/internal/auth"
	"github.com/raykavin/helix-acs/internal/config"
	cwmpserver "github.com/raykavin/helix-acs/internal/cwmp"
	"github.com/raykavin/helix-acs/internal/device"
	l "github.com/raykavin/helix-acs/internal/logger"
	"github.com/raykavin/helix-acs/internal/schema"
	"github.com/raykavin/helix-acs/internal/task"
	"github.com/raykavin/helix-acs/internal/wiring"
)

var configPath = flag.String("config", "./configs/config.yml", "path to config file (default: ./configs/config.yml)")

func main() {
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		stdlogger.Fatalf("failed to load config: %v\n", err)
	}

	appCfg := cfg.GetApplication()
	appLogger := wiring.NewLogger(appCfg)

	wiring.DisplayBanner(appCfg)
	appLogger.Debug("Helix ACS CWMP starting...")

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

	appLogger.Info("Helix ACS CWMP stopped")
}

func run(ctx context.Context, cfg config.ConfigProvider, appLogger l.Logger) error {
	storageDB, err := wiring.ConnectStorage(cfg, appLogger)
	if err != nil {
		return err
	}
	defer wiring.DisconnectStorage(storageDB, appLogger)

	cacheDB, err := wiring.ConnectCache(cfg, appLogger)
	if err != nil {
		return err
	}
	defer wiring.DisconnectCache(cacheDB, appLogger)

	repo, err := wiring.NewDeviceRepository(ctx, wiring.StorageDatabase(storageDB, cfg))
	if err != nil {
		return err
	}
	deviceSvc := wiring.NewDeviceService(repo, appLogger)

	tsk, err := cfg.GetApplication().GetTask(wiring.QueueTaskName)
	if err != nil {
		return fmt.Errorf("unable to find configuration for queue task %q", wiring.QueueTaskName)
	}

	appCfg := cfg.GetApplication()
	acsConfig := appCfg.GetACS()
	cacheCC, _ := cfg.GetDatabase(wiring.CacheDBName)

	taskQueue := wiring.NewTaskQueue(cacheDB, cacheCC.GetTTL(), tsk.GetMaxAttempts())
	schemaReg := initSchemaRegistry(acsConfig.GetSchemasDir(), appLogger)
	cwmpSrv := initCWMPServer(deviceSvc, taskQueue, acsConfig, appLogger, schemaReg)

	return wiring.ServeHTTP(ctx, fmt.Sprintf(":%d", acsConfig.GetListenPort()), "CWMP", cwmpSrv.Router(), appLogger)
}

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

func initCWMPServer(
	deviceSvc device.Service,
	taskQueue *task.RedisQueue,
	acs config.ACSConfigProvider,
	appLogger l.Logger,
	schemaReg *schema.Registry,
) *cwmpserver.Server {
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
