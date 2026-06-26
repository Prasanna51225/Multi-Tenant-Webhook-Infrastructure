package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/webhook-platform/internal/domain"
)

type SchedulerService struct {
	eventRepo EventRepository
	publisher KafkaPublisher
	logger    *slog.Logger
	interval  time.Duration
	batchSize int
}

func NewSchedulerService(
	eventRepo EventRepository,
	publisher KafkaPublisher,
	logger *slog.Logger,
) *SchedulerService {
	return &SchedulerService{
		eventRepo: eventRepo,
		publisher: publisher,
		logger:    logger,
		interval:  5 * time.Second,
		batchSize: 100,
	}
}

func (s *SchedulerService) Start(ctx context.Context) error {
	s.logger.Info("scheduler started", slog.Duration("interval", s.interval))

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("scheduler stopping")
			return nil
		case <-ticker.C:
			if err := s.processRetryableEvents(ctx); err != nil {
				s.logger.Error("process retryable events", slog.String("error", err.Error()))
			}
		}
	}
}

func (s *SchedulerService) processRetryableEvents(ctx context.Context) error {
	events, err := s.eventRepo.FindRetryable(ctx, s.batchSize)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		return nil
	}

	s.logger.Info("found retryable events", slog.Int("count", len(events)))

	for _, event := range events {
		if err := s.publisher.PublishEvent(
			ctx,
			event.ID,
			event.TenantID,
			event.EndpointID,
			event.EventType,
			event.Payload,
			event.Signature,
		); err != nil {
			s.logger.Error("republish event",
				slog.String("error", err.Error()),
				slog.String("event_id", event.ID),
			)
			continue
		}

		if err := s.eventRepo.UpdateStatus(ctx, event.ID, domain.EventStatusQueued); err != nil {
			s.logger.Error("update event status to queued",
				slog.String("error", err.Error()),
				slog.String("event_id", event.ID),
			)
		}
	}

	return nil
}
