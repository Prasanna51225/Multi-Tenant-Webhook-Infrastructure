package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/webhook-platform/internal/api/middleware"
	"github.com/webhook-platform/internal/domain"
	"github.com/webhook-platform/internal/service"
)

type EventHandler struct {
	svc service.EventService
}

func NewEventHandler(svc service.EventService) *EventHandler {
	return &EventHandler{svc: svc}
}

type createEventRequest struct {
	EndpointID string          `json:"endpoint_id"`
	EventType  string          `json:"event_type"`
	Payload    json.RawMessage `json:"payload"`
}

type createEventResponse struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	EventType string `json:"event_type"`
	CreatedAt string `json:"created_at"`
}

type eventResponse struct {
	ID           string          `json:"id"`
	EndpointID   string          `json:"endpoint_id"`
	EventType    string          `json:"event_type"`
	Payload      json.RawMessage `json:"payload"`
	Status       string          `json:"status"`
	AttemptCount int             `json:"attempt_count"`
	MaxAttempts  int             `json:"max_attempts"`
	CreatedAt    string          `json:"created_at"`
	UpdatedAt    string          `json:"updated_at"`
}

type eventListResponse struct {
	Data   []eventResponse `json:"data"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

func (h *EventHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		RespondError(w, http.StatusUnauthorized, "tenant not found in context")
		return
	}

	var req createEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	input := domain.CreateEventInput{
		EndpointID: req.EndpointID,
		EventType:  req.EventType,
		Payload:    req.Payload,
	}

	event, err := h.svc.Create(r.Context(), tenant.ID, input)
	if err != nil {
		HandleError(w, err)
		return
	}

	RespondJSON(w, http.StatusAccepted, createEventResponse{
		ID:        event.ID,
		Status:    event.Status,
		EventType: event.EventType,
		CreatedAt: event.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (h *EventHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	event, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		HandleError(w, err)
		return
	}

	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok || event.TenantID != tenant.ID {
		RespondError(w, http.StatusNotFound, "event not found")
		return
	}

	RespondJSON(w, http.StatusOK, eventResponse{
		ID:           event.ID,
		EndpointID:   event.EndpointID,
		EventType:    event.EventType,
		Payload:      event.Payload,
		Status:       event.Status,
		AttemptCount: event.AttemptCount,
		MaxAttempts:  event.MaxAttempts,
		CreatedAt:    event.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    event.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (h *EventHandler) List(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		RespondError(w, http.StatusUnauthorized, "tenant not found in context")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	events, total, err := h.svc.ListByTenantID(r.Context(), tenant.ID, limit, offset)
	if err != nil {
		HandleError(w, err)
		return
	}

	data := make([]eventResponse, 0, len(events))
	for _, e := range events {
		data = append(data, eventResponse{
			ID:           e.ID,
			EndpointID:   e.EndpointID,
			EventType:    e.EventType,
			Payload:      e.Payload,
			Status:       e.Status,
			AttemptCount: e.AttemptCount,
			MaxAttempts:  e.MaxAttempts,
			CreatedAt:    e.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:    e.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	RespondJSON(w, http.StatusOK, eventListResponse{
		Data:   data,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

func (h *EventHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Route("/{id}", func(r chi.Router) {
		r.Get("/", h.Get)
	})
	return r
}
