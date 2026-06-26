package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/webhook-platform/internal/kafka"
	"github.com/webhook-platform/internal/repository/postgres"
	redisRepo "github.com/webhook-platform/internal/repository/redis"
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

	brokers := []string{cfg.KafkaBrokers}

	consumer := kafka.NewConsumer(brokers, kafka.TopicEvents, "dispatcher-group", log)
	defer consumer.Close()

	publisher := kafka.NewProducer(brokers, kafka.TopicRetries, log)
	defer publisher.Close()

	eventRepo := postgres.NewEventRepo(pool)
	endpointRepo := postgres.NewEndpointRepo(pool)
	tenantRepo := postgres.NewTenantRepo(pool)
	deliveryRepo := postgres.NewDeliveryAttemptRepo(pool)
	deadLetterRepo := postgres.NewDeadLetterEventRepo(pool)

	circuitBreakerRepo := redisRepo.NewCircuitBreakerRepo(rdb, 5, 60*time.Second)
	circuitBreakerSvc := service.NewCircuitBreakerService(circuitBreakerRepo)

	lockRepo := redisRepo.NewLockRepo(rdb)
	endpointCache := redisRepo.NewEndpointCache(rdb)

	dispatcher := service.NewDispatcherService(
		eventRepo,
		endpointRepo,
		tenantRepo,
		deliveryRepo,
		deadLetterRepo,
		consumer,
		publisher,
		circuitBreakerSvc,
		lockRepo,
		endpointCache,
		log,
	)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Info("shutting down dispatcher")
		cancel()
	}()

	if err := dispatcher.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Error("dispatcher error", slog.String("error", err.Error()))
		os.Exit(1)
	}

	log.Info("dispatcher stopped gracefully")
}
