package wiring

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/raykavin/helix-acs/internal/config"
	"github.com/raykavin/helix-acs/internal/device"
	l "github.com/raykavin/helix-acs/internal/logger"
)

// StorageDatabase returns the *mongo.Database for the storage DB name from config.
func StorageDatabase(client *mongo.Client, cfg config.ConfigProvider) *mongo.Database {
	stg, _ := cfg.GetDatabase(StorageDBName) // already validated by ConnectStorage
	return client.Database(stg.GetName())
}

// ConnectStorage reads the storage DB config and connects to MongoDB with retries.
func ConnectStorage(cfg config.ConfigProvider, log l.Logger) (*mongo.Client, error) {
	stg, err := cfg.GetDatabase(StorageDBName)
	if err != nil {
		return nil, fmt.Errorf("unable to find configuration for storage database %q", StorageDBName)
	}
	uri := stg.GetURI()
	client, err := ConnectMongoDB(uri, log)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to storage database: %w", err)
	}
	log.WithField("uri", uri).Debug("Connected to storage database")
	return client, nil
}

// DisconnectStorage gracefully closes the MongoDB connection.
func DisconnectStorage(client *mongo.Client, log l.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Disconnect(ctx); err != nil {
		log.WithError(err).Error("error disconnecting from MongoDB")
	}
}

// ConnectMongoDB dials MongoDB with up to DBMaxRetries attempts.
func ConnectMongoDB(uri string, log l.Logger) (*mongo.Client, error) {
	var lastErr error
	for attempt := 1; attempt <= DBMaxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), DBAttemptTimeout)
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
		cancel()
		if err != nil {
			lastErr = err
			log.WithError(err).WithField("attempt", attempt).Warn("MongoDB connect failed, retrying")
			time.Sleep(DBRetryInterval)
			continue
		}

		pingCtx, pingCancel := context.WithTimeout(context.Background(), DBAttemptTimeout)
		err = client.Ping(pingCtx, nil)
		pingCancel()
		if err != nil {
			lastErr = err
			_ = client.Disconnect(context.Background())
			log.WithError(err).WithField("attempt", attempt).Warn("MongoDB ping failed, retrying")
			time.Sleep(DBRetryInterval)
			continue
		}

		return client, nil
	}
	return nil, fmt.Errorf("mongodb: failed after %d attempts: %w", DBMaxRetries, lastErr)
}

// NewDeviceRepository creates a MongoDB-backed device repository.
// A 30-second startup context bounds index creation and other one-off setup operations.
func NewDeviceRepository(ctx context.Context, db *mongo.Database) (device.Repository, error) {
	startupCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	repo, err := device.NewMongoRepository(startupCtx, db)
	if err != nil {
		return nil, fmt.Errorf("failed to create device repository: %w", err)
	}
	return repo, nil
}

// NewDeviceService creates the device service backed by the given repository.
func NewDeviceService(repo device.Repository, log l.Logger) device.Service {
	return device.NewService(repo, log)
}
