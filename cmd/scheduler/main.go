package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/webhook-platform/internal/kafka"
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

	brokers := []string{cfg.KafkaBrokers}

	publisher := kafka.NewProducer(brokers, kafka.TopicEvents, log)
	defer publisher.Close()

	eventRepo := postgres.NewEventRepo(pool)

	scheduler := service.NewSchedulerService(eventRepo, publisher, log)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Info("shutting down scheduler")
		cancel()
	}()

	if err := scheduler.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Error("scheduler error", slog.String("error", err.Error()))
		os.Exit(1)
	}

	log.Info("scheduler stopped gracefully")
}
