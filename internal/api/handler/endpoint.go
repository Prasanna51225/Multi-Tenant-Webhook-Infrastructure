package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/webhook-platform/internal/api/middleware"
	"github.com/webhook-platform/internal/api/response"
	"github.com/webhook-platform/internal/domain"
	"github.com/webhook-platform/internal/service"
)

type EndpointHandler struct {
	svc service.EndpointService
}

func NewEndpointHandler(svc service.EndpointService) *EndpointHandler {
	return &EndpointHandler{svc: svc}
}

type createEndpointRequest struct {
	URL         string   `json:"url"`
	Description string   `json:"description"`
	EventTypes  []string `json:"event_types"`
	Active      *bool    `json:"active"`
}

type createEndpointResponse struct {
	ID          string   `json:"id"`
	URL         string   `json:"url"`
	Description string   `json:"description,omitempty"`
	EventTypes  []string `json:"event_types"`
	Secret      string   `json:"secret"`
	Active      bool     `json:"active"`
	CreatedAt   string   `json:"created_at"`
}

type endpointResponse struct {
	ID          string   `json:"id"`
	URL         string   `json:"url"`
	Description string   `json:"description,omitempty"`
	EventTypes  []string `json:"event_types"`
	Active      bool     `json:"active"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

type endpointListResponse struct {
	Data   []endpointResponse `json:"data"`
	Total  int                `json:"total"`
	Limit  int                `json:"limit"`
	Offset int                `json:"offset"`
}

type updateEndpointRequest struct {
	URL         *string  `json:"url"`
	Description *string  `json:"description"`
	EventTypes  []string `json:"event_types"`
	Active      *bool    `json:"active"`
}

func (h *EndpointHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "tenant not found in context")
		return
	}

	var req createEndpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	input := domain.CreateEndpointInput{
		URL:         req.URL,
		Description: req.Description,
		EventTypes:  req.EventTypes,
		Active:      req.Active,
	}

	endpoint, secret, err := h.svc.Create(r.Context(), tenant.ID, input)
	if err != nil {
		response.HandleError(w, err)
		return
	}

	eventTypes := endpoint.EventTypes
	if eventTypes == nil {
		eventTypes = []string{}
	}

	response.JSON(w, http.StatusCreated, createEndpointResponse{
		ID:          endpoint.ID,
		URL:         endpoint.URL,
		Description: endpoint.Description,
		EventTypes:  eventTypes,
		Secret:      secret,
		Active:      endpoint.Active,
		CreatedAt:   endpoint.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (h *EndpointHandler) List(w http.ResponseWriter, r *http.Request) {
	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "tenant not found in context")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	endpoints, total, err := h.svc.ListByTenantID(r.Context(), tenant.ID, limit, offset)
	if err != nil {
		response.HandleError(w, err)
		return
	}

	data := make([]endpointResponse, 0, len(endpoints))
	for _, ep := range endpoints {
		eventTypes := ep.EventTypes
		if eventTypes == nil {
			eventTypes = []string{}
		}
		data = append(data, endpointResponse{
			ID:          ep.ID,
			URL:         ep.URL,
			Description: ep.Description,
			EventTypes:  eventTypes,
			Active:      ep.Active,
			CreatedAt:   ep.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:   ep.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	response.JSON(w, http.StatusOK, endpointListResponse{
		Data:   data,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

func (h *EndpointHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	endpoint, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		response.HandleError(w, err)
		return
	}

	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok || endpoint.TenantID != tenant.ID {
		response.Error(w, http.StatusNotFound, "endpoint not found")
		return
	}

	eventTypes := endpoint.EventTypes
	if eventTypes == nil {
		eventTypes = []string{}
	}

	response.JSON(w, http.StatusOK, endpointResponse{
		ID:          endpoint.ID,
		URL:         endpoint.URL,
		Description: endpoint.Description,
		EventTypes:  eventTypes,
		Active:      endpoint.Active,
		CreatedAt:   endpoint.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   endpoint.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (h *EndpointHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "tenant not found in context")
		return
	}

	var req updateEndpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	input := domain.UpdateEndpointInput{
		URL:         req.URL,
		Description: req.Description,
		EventTypes:  req.EventTypes,
		Active:      req.Active,
	}

	endpoint, err := h.svc.Update(r.Context(), id, tenant.ID, input)
	if err != nil {
		response.HandleError(w, err)
		return
	}

	eventTypes := endpoint.EventTypes
	if eventTypes == nil {
		eventTypes = []string{}
	}

	response.JSON(w, http.StatusOK, endpointResponse{
		ID:          endpoint.ID,
		URL:         endpoint.URL,
		Description: endpoint.Description,
		EventTypes:  eventTypes,
		Active:      endpoint.Active,
		CreatedAt:   endpoint.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   endpoint.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (h *EndpointHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	tenant, ok := middleware.GetTenantFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "tenant not found in context")
		return
	}

	if err := h.svc.Delete(r.Context(), id, tenant.ID); err != nil {
		response.HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *EndpointHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Route("/{id}", func(r chi.Router) {
		r.Get("/", h.Get)
		r.Put("/", h.Update)
		r.Delete("/", h.Delete)
	})
	return r
}
