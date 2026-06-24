package handler

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/webhook-platform/internal/api/response"
)

type HealthHandler struct {
	pool  *pgxpool.Pool
	redis *redis.Client
}

func NewHealthHandler(pool *pgxpool.Pool, redis *redis.Client) *HealthHandler {
	return &HealthHandler{pool: pool, redis: redis}
}

func (h *HealthHandler) Live(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"status": "alive"})
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.pool.Ping(r.Context()); err != nil {
		response.Error(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}

	if err := h.redis.Ping(r.Context()).Err(); err != nil {
		response.Error(w, http.StatusServiceUnavailable, "redis unavailable")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
