package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
)

type RateLimiter interface {
	Allow(ctx context.Context, tenantID string, limit int) (bool, error)
}

func RateLimit(limiter RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenant, ok := GetTenantFromContext(r.Context())
			if !ok {
				writeError(w, http.StatusUnauthorized, "tenant not found")
				return
			}

			allowed, err := limiter.Allow(r.Context(), tenant.ID, tenant.RateLimitPerMinute)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "rate limit check failed")
				return
			}

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(tenant.RateLimitPerMinute))

			if !allowed {
				w.Header().Set("Retry-After", "60")
				writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
		"code":  http.StatusText(status),
	})
}
