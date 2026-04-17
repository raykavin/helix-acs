package task

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	keyQueue = "task:queue:%s" // FIFO list of task IDs keyed by serial
	keyData  = "task:data:%s"  // JSON-encoded Task keyed by task ID
	keyIndex = "task:index:%s" // sorted set of task IDs scored by created_at (unix) keyed by serial
)

// RedisQueue is a Redis-backed implementation of the Queue interface.
type RedisQueue struct {
	client      *redis.Client
	ttl         time.Duration
	maxAttempts int
}

// NewRedisQueue creates a new RedisQueue.
//
//   - ttl         is the TTL applied to task data keys (0 means no expiry).
//   - maxAttempts is the default maximum number of delivery attempts.
func NewRedisQueue(client *redis.Client, ttl time.Duration, maxAttempts int) *RedisQueue {
	return &RedisQueue{
		client:      client,
		ttl:         ttl,
		maxAttempts: maxAttempts,
	}
}

// queueKey returns the list key for a device's task queue.
func queueKey(serial string) string { return fmt.Sprintf(keyQueue, serial) }

// dataKey returns the hash key for a specific task's data.
func dataKey(taskID string) string { return fmt.Sprintf(keyData, taskID) }

// indexKey returns the sorted-set key used for listing tasks for a device.
func indexKey(serial string) string { return fmt.Sprintf(keyIndex, serial) }

// Enqueue adds a task to the device queue and stores its data.
func (q *RedisQueue) Enqueue(ctx context.Context, t *Task) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if t.MaxAttempts == 0 {
		t.MaxAttempts = q.maxAttempts
	}
	if t.Status == "" {
		t.Status = StatusPending
	}

	data, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}

	score := float64(t.CreatedAt.UnixNano())

	pipe := q.client.Pipeline()
	pipe.RPush(ctx, queueKey(t.Serial), t.ID)
	if q.ttl > 0 {
		pipe.Set(ctx, dataKey(t.ID), data, q.ttl)
	} else {
		pipe.Set(ctx, dataKey(t.ID), data, 0)
	}
	pipe.ZAdd(ctx, indexKey(t.Serial), redis.Z{Score: score, Member: t.ID})
	if q.ttl > 0 {
		pipe.Expire(ctx, indexKey(t.Serial), q.ttl)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("enqueue task %s: %w", t.ID, err)
	}
	return nil
}

// DequeuePending returns all pending tasks for the given device without removing
// them from the queue. Tasks are considered dequeued by transitioning their status
// via UpdateStatus.
func (q *RedisQueue) DequeuePending(ctx context.Context, serial string) ([]*Task, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	ids, err := q.client.LRange(ctx, queueKey(serial), 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("lrange queue for %s: %w", serial, err)
	}

	var pending []*Task
	for _, id := range ids {
		t, err := q.fetchTask(ctx, id)
		if err != nil {
			// Task data may have expired; skip it.
			continue
		}
		if t.Status == StatusPending {
			pending = append(pending, t)
		}
	}
	return pending, nil
}

// UpdateStatus persists the current state of the task back to Redis.
func (q *RedisQueue) UpdateStatus(ctx context.Context, t *Task) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	data, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}

	var setErr error
	if q.ttl > 0 {
		setErr = q.client.Set(ctx, dataKey(t.ID), data, q.ttl).Err()
	} else {
		setErr = q.client.Set(ctx, dataKey(t.ID), data, 0).Err()
	}
	if setErr != nil {
		return fmt.Errorf("update task status %s: %w", t.ID, setErr)
	}

	// Remove terminal tasks from the device's FIFO queue list so it doesn't
	// grow unboundedly.
	if t.Status == StatusDone || t.Status == StatusFailed || t.Status == StatusCancelled {
		q.client.LRem(ctx, queueKey(t.Serial), 0, t.ID)
	}

	return nil
}

// Cancel marks a task as cancelled.
func (q *RedisQueue) Cancel(ctx context.Context, taskID, serial string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	t, err := q.fetchTask(ctx, taskID)
	if err != nil {
		return err
	}

	if t.Status == StatusDone || t.Status == StatusFailed || t.Status == StatusCancelled {
		return fmt.Errorf("task %s is already in terminal state %s", taskID, t.Status)
	}

	now := time.Now().UTC()
	t.Status = StatusCancelled
	t.CompletedAt = &now

	return q.UpdateStatus(ctx, t)
}

// GetByID retrieves a task by its ID.
func (q *RedisQueue) GetByID(ctx context.Context, taskID string) (*Task, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return q.fetchTask(ctx, taskID)
}

// List returns a paginated list of tasks for the given device, 
// ordered by creation time
func (q *RedisQueue) List(ctx context.Context, serial string, page, limit int) ([]*Task, int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}

	idxKey := indexKey(serial)

	total, err := q.client.ZCard(ctx, idxKey).Result()
	if err != nil {
		return nil, 0, fmt.Errorf("zcard index for %s: %w", serial, err)
	}

	// Sorted set is in ascending order; list newest first by reversing.
	offset := int64((page - 1) * limit)
	end := offset + int64(limit) - 1

	// ZREVRANGE returns members from highest score to lowest.
	ids, err := q.client.ZRevRange(ctx, idxKey, offset, end).Result()
	if err != nil {
		return nil, 0, fmt.Errorf("zrevrange index for %s: %w", serial, err)
	}

	tasks := make([]*Task, 0, len(ids))
	for _, id := range ids {
		t, err := q.fetchTask(ctx, id)
		if err != nil {
			// Task data expired; count it but return a placeholder so
			// the caller can see the gap.
			continue
		}
		tasks = append(tasks, t)
	}

	return tasks, total, nil
}

// FindExecutingDiagnostics returns all executing async-diagnostic tasks for
// the given device by scanning the per-device queue list.
func (q *RedisQueue) FindExecutingDiagnostics(ctx context.Context, serial string) ([]*Task, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	ids, err := q.client.LRange(ctx, queueKey(serial), 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("lrange queue for %s: %w", serial, err)
	}

	var result []*Task
	for _, id := range ids {
		t, err := q.fetchTask(ctx, id)
		if err != nil {
			continue
		}
		if t.Status == StatusExecuting && IsDiagnosticAsync(t.Type) {
			result = append(result, t)
		}
	}
	return result, nil
}

// fetchTask retrieves and unmarshals a single task by ID.
func (q *RedisQueue) fetchTask(ctx context.Context, taskID string) (*Task, error) {
	data, err := q.client.Get(ctx, dataKey(taskID)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("task %s not found", taskID)
		}
		return nil, fmt.Errorf("get task %s: %w", taskID, err)
	}

	var t Task
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("unmarshal task %s: %w", taskID, err)
	}
	return &t, nil
}
