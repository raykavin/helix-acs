package wiring

import "time"

const (
	StorageDBName = "storage"
	CacheDBName   = "cache"
	QueueTaskName = "queue"
)

const (
	DBMaxRetries     = 3
	DBRetryInterval  = 2 * time.Second
	DBAttemptTimeout = 10 * time.Second
)
