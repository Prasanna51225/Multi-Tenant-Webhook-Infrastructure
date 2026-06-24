package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/webhook-platform/internal/api/response"
	"github.com/webhook-platform/internal/domain"
)

type contextKey string

const tenantCtxKey contextKey = "tenant"

type TenantLookupFunc func(ctx context.Context, apiKey string) (*domain.Tenant, error)

func Auth(lookup TenantLookupFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := extractAPIKey(r)
			if apiKey == "" {
				response.Error(w, http.StatusUnauthorized, "missing or invalid authorization header")
				return
			}

			tenant, err := lookup(r.Context(), apiKey)
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "invalid api key")
				return
			}

			ctx := context.WithValue(r.Context(), tenantCtxKey, tenant)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetTenantFromContext(ctx context.Context) (*domain.Tenant, bool) {
	tenant, ok := ctx.Value(tenantCtxKey).(*domain.Tenant)
	return tenant, ok
}

func extractAPIKey(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}

	return r.Header.Get("X-API-Key")
}
