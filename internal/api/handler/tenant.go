package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/webhook-platform/internal/api/middleware"
	"github.com/webhook-platform/internal/domain"
	"github.com/webhook-platform/internal/service"
)

type TenantHandler struct {
	svc service.TenantService
}

func NewTenantHandler(svc service.TenantService) *TenantHandler {
	return &TenantHandler{svc: svc}
}

type createTenantRequest struct {
	Name               string `json:"name"`
	RateLimitPerMinute int    `json:"rate_limit_per_minute"`
	MaxRetries         int    `json:"max_retries"`
	RetryBaseMs        int    `json:"retry_base_ms"`
}

type createTenantResponse struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	APIKey             string `json:"api_key"`
	RateLimitPerMinute int    `json:"rate_limit_per_minute"`
	MaxRetries         int    `json:"max_retries"`
	RetryBaseMs        int    `json:"retry_base_ms"`
	CreatedAt          string `json:"created_at"`
}

type tenantResponse struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	RateLimitPerMinute int    `json:"rate_limit_per_minute"`
	MaxRetries         int    `json:"max_retries"`
	RetryBaseMs        int    `json:"retry_base_ms"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

type updateTenantRequest struct {
	Name               *string `json:"name"`
	RateLimitPerMinute *int    `json:"rate_limit_per_minute"`
	MaxRetries         *int    `json:"max_retries"`
	RetryBaseMs        *int    `json:"retry_base_ms"`
}

func (h *TenantHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("DEBUG: JSON decode error: %v", err)
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	log.Printf("DEBUG: Create tenant request: name=%s rate_limit=%d max_retries=%d retry_base=%d",
		req.Name, req.RateLimitPerMinute, req.MaxRetries, req.RetryBaseMs)

	input := domain.CreateTenantInput{
		Name:               req.Name,
		RateLimitPerMinute: req.RateLimitPerMinute,
		MaxRetries:         req.MaxRetries,
		RetryBaseMs:        req.RetryBaseMs,
	}

	tenant, apiKey, err := h.svc.Create(r.Context(), input)
	if err != nil {
		log.Printf("DEBUG: Service create error: %v", err)
		HandleError(w, err)
		return
	}

	RespondJSON(w, http.StatusCreated, createTenantResponse{
		ID:                 tenant.ID,
		Name:               tenant.Name,
		APIKey:             apiKey,
		RateLimitPerMinute: tenant.RateLimitPerMinute,
		MaxRetries:         tenant.MaxRetries,
		RetryBaseMs:        tenant.RetryBaseMs,
		CreatedAt:          tenant.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (h *TenantHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		RespondError(w, http.StatusUnauthorized, "tenant not found in context")
		return
	}

	RespondJSON(w, http.StatusOK, tenantResponse{
		ID:                 tenant.ID,
		Name:               tenant.Name,
		RateLimitPerMinute: tenant.RateLimitPerMinute,
		MaxRetries:         tenant.MaxRetries,
		RetryBaseMs:        tenant.RetryBaseMs,
		CreatedAt:          tenant.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:          tenant.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (h *TenantHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		RespondError(w, http.StatusUnauthorized, "tenant not found in context")
		return
	}

	var req updateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	input := domain.UpdateTenantInput{
		Name:               req.Name,
		RateLimitPerMinute: req.RateLimitPerMinute,
		MaxRetries:         req.MaxRetries,
		RetryBaseMs:        req.RetryBaseMs,
	}

	updated, err := h.svc.Update(r.Context(), tenant.ID, input)
	if err != nil {
		HandleError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, tenantResponse{
		ID:                 updated.ID,
		Name:               updated.Name,
		RateLimitPerMinute: updated.RateLimitPerMinute,
		MaxRetries:         updated.MaxRetries,
		RetryBaseMs:        updated.RetryBaseMs,
		CreatedAt:          updated.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:          updated.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (h *TenantHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/me", h.GetMe)
	r.Put("/me", h.UpdateMe)
	return r
}
