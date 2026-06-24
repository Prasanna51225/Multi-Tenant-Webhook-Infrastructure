package api

import (
	"net/http"

	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/webhook-platform/internal/api/handler"
	apimw "github.com/webhook-platform/internal/api/middleware"
	"github.com/webhook-platform/internal/service"
)

type Server struct {
	router      chi.Router
	logger      *slog.Logger
	tenantSvc   service.TenantService
	endpointSvc service.EndpointService
	eventSvc    service.EventService
	pool        *pgxpool.Pool
	redis       *redis.Client
}

func NewServer(
	logger *slog.Logger,
	tenantSvc service.TenantService,
	endpointSvc service.EndpointService,
	eventSvc service.EventService,
	pool *pgxpool.Pool,
	redis *redis.Client,
) *Server {
	s := &Server{
		logger:      logger,
		tenantSvc:   tenantSvc,
		endpointSvc: endpointSvc,
		eventSvc:    eventSvc,
		pool:        pool,
		redis:       redis,
	}

	s.setupRouter()
	return s
}

func (s *Server) setupRouter() {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(apimw.CORS)
	r.Use(apimw.Logging(s.logger))
	r.Use(middleware.Recoverer)

	tenantHandler := handler.NewTenantHandler(s.tenantSvc)
	endpointHandler := handler.NewEndpointHandler(s.endpointSvc)
	eventHandler := handler.NewEventHandler(s.eventSvc)
	healthHandler := handler.NewHealthHandler(s.pool, s.redis)

	r.Get("/healthz", healthHandler.Live)
	r.Get("/readyz", healthHandler.Ready)

	r.Route("/api/v1", func(r chi.Router) {
		r.Mount("/tenants", tenantHandler.Routes())

		r.Group(func(r chi.Router) {
			r.Use(apimw.Auth(s.tenantSvc.GetByAPIKey))
			r.Mount("/endpoints", endpointHandler.Routes())
			r.Mount("/events", eventHandler.Routes())
		})
	})

	s.router = r
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
