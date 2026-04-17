package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	StatusOk       = "OK"
	StatusDegraded = "DEGRADED"
)

// HealthHandler performs liveness/readiness checks on backing services.
type HealthHandler struct {
	mongoClient *mongo.Client
	redisClient *redis.Client
}

// NewHealthHandler creates a HealthHandler.
func NewHealthHandler(mongoClient *mongo.Client, redisClient *redis.Client) *HealthHandler {
	return &HealthHandler{
		mongoClient: mongoClient,
		redisClient: redisClient,
	}
}

// healthResponse is the JSON body returned by Check.
type healthResponse struct {
	Status  string `json:"status"`
	MongoDB string `json:"mongodb"`
	Redis   string `json:"redis"`
}

// Check handles GET /health.
// It pings both MongoDB and Redis. Returns 200 when both are healthy,
// or 503 with status "degraded" when either is unreachable.
func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	mongoStatus := StatusOk
	if err := h.mongoClient.Ping(ctx, nil); err != nil {
		mongoStatus = "error"
	}

	redisStatus := StatusOk
	if err := h.redisClient.Ping(ctx).Err(); err != nil {
		redisStatus = "error"
	}

	resp := healthResponse{
		Status:  StatusOk,
		MongoDB: mongoStatus,
		Redis:   redisStatus,
	}

	statusCode := http.StatusOK
	if mongoStatus != StatusOk || redisStatus != StatusOk {
		resp.Status = StatusDegraded
		statusCode = http.StatusServiceUnavailable
	}

	writeJSON(w, statusCode, resp)
}
