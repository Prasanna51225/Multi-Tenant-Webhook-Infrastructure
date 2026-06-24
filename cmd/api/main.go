package main

import (
    "context"
    "errors"
    "fmt"
    "log/slog"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/webhook-platform/internal/api"
    redisRepo "github.com/webhook-platform/internal/repository/redis"
    "github.com/webhook-platform/internal/repository/postgres"
    "github.com/webhook-platform/internal/service"
    "github.com/webhook-platform/pkg/config"
    "github.com/webhook-platform/pkg/logger"
)

func main() {
    ctx := context.Background()

    cfg, err := config.Load()
    if err != nil {
        slog.Error("failed to load config", slog.String("error", err.Error()))
        os.Exit(1)
    }

    log := logger.New(cfg)

    pool, err := postgres.NewPool(ctx, cfg.DBURL)
    if err != nil {
        log.Error("failed to connect to database", slog.String("error", err.Error()))
        os.Exit(1)
    }
    defer pool.Close()

    rdb, err := redisRepo.NewClient(ctx, cfg.RedisURL)
    if err != nil {
        log.Error("failed to connect to redis", slog.String("error", err.Error()))
        os.Exit(1)
    }
    defer rdb.Close()

    tenantRepo := postgres.NewTenantRepo(pool)
    endpointRepo := postgres.NewEndpointRepo(pool)

    tenantSvc := service.NewTenantService(tenantRepo)
    endpointSvc := service.NewEndpointService(endpointRepo)

    srv := api.NewServer(log, tenantSvc, endpointSvc, pool, rdb)

    httpServer := &http.Server{
        Addr:         fmt.Sprintf(":%s", cfg.HTTPPort),
        Handler:      srv,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    go func() {
        log.Info("starting api server", slog.String("port", cfg.HTTPPort))
        if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
            log.Error("server error", slog.String("error", err.Error()))
            os.Exit(1)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Info("shutting down server")

    shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := httpServer.Shutdown(shutdownCtx); err != nil {
        log.Error("server forced to shutdown", slog.String("error", err.Error()))
        os.Exit(1)
    }

    log.Info("server stopped gracefully")
}