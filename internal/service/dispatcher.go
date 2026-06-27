package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/webhook-platform/internal/delivery"
	"github.com/webhook-platform/internal/domain"
	kafkapkg "github.com/webhook-platform/internal/kafka"
	"github.com/webhook-platform/internal/repository/redis"
	"github.com/webhook-platform/pkg/retry"
	"github.com/webhook-platform/pkg/telemetry"
)

type DispatcherService struct {
	eventRepo      EventRepository
	endpointRepo   EndpointLookup
	tenantRepo     TenantLookup
	deliveryRepo   DeliveryAttemptRepository
	deadLetterRepo DeadLetterEventRepository
	consumer       KafkaConsumer
	publisher      KafkaPublisher
	webhookClient  *delivery.WebhookClient
	circuitBreaker CircuitBreakerService
	lockRepo       *redis.LockRepo
	endpointCache  *redis.EndpointCache
	logger         *slog.Logger
}

func NewDispatcherService(
	eventRepo EventRepository,
	endpointRepo EndpointLookup,
	tenantRepo TenantLookup,
	deliveryRepo DeliveryAttemptRepository,
	deadLetterRepo DeadLetterEventRepository,
	consumer KafkaConsumer,
	publisher KafkaPublisher,
	circuitBreaker CircuitBreakerService,
	lockRepo *redis.LockRepo,
	endpointCache *redis.EndpointCache,
	logger *slog.Logger,
) *DispatcherService {
	return &DispatcherService{
		eventRepo:      eventRepo,
		endpointRepo:   endpointRepo,
		tenantRepo:     tenantRepo,
		deliveryRepo:   deliveryRepo,
		deadLetterRepo: deadLetterRepo,
		consumer:       consumer,
		publisher:      publisher,
		webhookClient:  delivery.NewWebhookClient(),
		circuitBreaker: circuitBreaker,
		lockRepo:       lockRepo,
		endpointCache:  endpointCache,
		logger:         logger,
	}
}

func (s *DispatcherService) Start(ctx context.Context) error {
	s.logger.Info("dispatcher started consuming events")

	for {
		msg, err := s.consumer.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				s.logger.Info("dispatcher stopping")
				return nil
			}
			s.logger.Error("fetch message error", slog.String("error", err.Error()))
			continue
		}

		if err := s.processMessage(ctx, msg); err != nil {
			s.logger.Error("process message error",
				slog.String("error", err.Error()),
				slog.String("topic", msg.Topic),
			)
		}

		if err := s.consumer.CommitMessages(ctx, msg); err != nil {
			s.logger.Error("commit message error", slog.String("error", err.Error()))
		}
	}
}

func (s *DispatcherService) processMessage(ctx context.Context, msg kafka.Message) error {
	var eventMsg kafkapkg.EventMessage
	if err := json.Unmarshal(msg.Value, &eventMsg); err != nil {
		s.logger.Error("unmarshal event message", slog.String("error", err.Error()))
		return nil
	}

	telemetry.KafkaMessagesConsumed.WithLabelValues(msg.Topic, "dispatcher-group").Inc()

	event, err := s.eventRepo.GetByID(ctx, eventMsg.EventID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.logger.Warn("event not found, skipping", slog.String("event_id", eventMsg.EventID))
			return nil
		}
		return fmt.Errorf("get event: %w", err)
	}

	if event.Status == domain.EventStatusDelivered || event.Status == domain.EventStatusFailed {
		s.logger.Info("event already resolved, skipping",
			slog.String("event_id", event.ID),
			slog.String("status", event.Status),
		)
		return nil
	}

	lockVal, locked := s.lockRepo.Acquire(ctx, "dispatch:"+event.ID, 5*time.Minute)
	if !locked {
		s.logger.Info("event already being processed, skipping", slog.String("event_id", event.ID))
		return nil
	}
	defer s.lockRepo.Release(ctx, "dispatch:"+event.ID, lockVal)

	allowed, cbState, err := s.circuitBreaker.Allow(ctx, event.EndpointID)
	if err != nil {
		s.logger.Error("circuit breaker check error", slog.String("error", err.Error()))
	}
	if !allowed {
		s.logger.Warn("circuit breaker open, skipping delivery",
			slog.String("event_id", event.ID),
			slog.String("endpoint_id", event.EndpointID),
			slog.String("cb_state", cbState),
		)
		return nil
	}

	endpoint, err := s.getEndpoint(ctx, event.EndpointID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.handleDeadLetter(ctx, event, "endpoint not found", 0)
			return nil
		}
		return fmt.Errorf("get endpoint: %w", err)
	}

	if !endpoint.Active {
		s.handleDeadLetter(ctx, event, "endpoint inactive", 0)
		return nil
	}

	headers := map[string]string{
		"X-Webhook-ID":        event.ID,
		"X-Webhook-Signature": event.Signature,
		"X-Webhook-Event":     event.EventType,
		"X-Webhook-Attempt":   strconv.Itoa(event.AttemptCount + 1),
	}

	result := s.webhookClient.Deliver(ctx, endpoint.URL, event.Payload, headers)

	telemetry.DeliveryDuration.WithLabelValues(event.EndpointID).Observe(result.Duration.Seconds())

	s.recordAttempt(ctx, event, result)

	if result.StatusCode >= 200 && result.StatusCode < 300 && result.Error == nil {
		s.logger.Info("webhook delivered successfully",
			slog.String("event_id", event.ID),
			slog.String("endpoint_id", event.EndpointID),
			slog.Int("status_code", result.StatusCode),
			slog.Duration("duration", result.Duration),
		)
		s.circuitBreaker.RecordSuccess(ctx, event.EndpointID)
		telemetry.EventsDelivered.WithLabelValues(event.EndpointID, strconv.Itoa(result.StatusCode)).Inc()
		if err := s.eventRepo.UpdateStatus(ctx, event.ID, domain.EventStatusDelivered); err != nil {
			s.logger.Error("update event status to delivered", slog.String("error", err.Error()))
		}
	} else {
		s.circuitBreaker.RecordFailure(ctx, event.EndpointID)
		telemetry.EventsFailed.WithLabelValues(event.EndpointID).Inc()
		s.handleFailure(ctx, event, result)
	}

	return nil
}

func (s *DispatcherService) getEndpoint(ctx context.Context, endpointID string) (*domain.Endpoint, error) {
	cached, err := s.endpointCache.Get(ctx, endpointID)
	if err != nil {
		s.logger.Warn("cache get error, falling back to db", slog.String("error", err.Error()))
	}
	if cached != nil {
		return cached, nil
	}

	endpoint, err := s.endpointRepo.GetByID(ctx, endpointID)
	if err != nil {
		return nil, err
	}

	if cacheErr := s.endpointCache.Set(ctx, endpoint); cacheErr != nil {
		s.logger.Warn("cache set error", slog.String("error", cacheErr.Error()))
	}

	return endpoint, nil
}

func (s *DispatcherService) handleFailure(ctx context.Context, event *domain.Event, result delivery.DeliveryResult) {
	newAttemptCount := event.AttemptCount + 1
	failureReason := failureMessage(result)

	if newAttemptCount >= event.MaxAttempts {
		s.handleDeadLetter(ctx, event, failureReason, result.StatusCode)
		return
	}

	tenant, _ := s.tenantRepo.GetByID(ctx, event.TenantID)
	retryBaseMs := 1000
	if tenant != nil {
		retryBaseMs = tenant.RetryBaseMs
	}

	nextRetryAt := retry.NextRetryAt(newAttemptCount, retryBaseMs)

	s.logger.Info("scheduling retry",
		slog.String("event_id", event.ID),
		slog.Int("attempt", newAttemptCount),
		slog.Int("max_attempts", event.MaxAttempts),
		slog.Time("next_retry_at", nextRetryAt),
	)

	if err := s.eventRepo.UpdateForRetry(ctx, event.ID, domain.EventStatusRetrying, newAttemptCount, nextRetryAt); err != nil {
		s.logger.Error("update event for retry", slog.String("error", err.Error()))
	}
}

func (s *DispatcherService) handleDeadLetter(ctx context.Context, event *domain.Event, reason string, lastStatusCode int) {
	s.logger.Warn("moving event to dead letter queue",
		slog.String("event_id", event.ID),
		slog.String("reason", reason),
	)

	dlqEvent := &domain.DeadLetterEvent{
		ID:              uuid.New().String(),
		TenantID:        event.TenantID,
		EndpointID:      event.EndpointID,
		OriginalEventID: event.ID,
		EventType:       event.EventType,
		Payload:         event.Payload,
		FailureReason:   reason,
		LastStatusCode:  lastStatusCode,
		TotalAttempts:   event.AttemptCount,
		CreatedAt:       time.Now().UTC(),
	}

	if err := s.deadLetterRepo.Create(ctx, dlqEvent); err != nil {
		s.logger.Error("create dead letter event", slog.String("error", err.Error()))
	}

	if err := s.eventRepo.UpdateStatus(ctx, event.ID, domain.EventStatusFailed); err != nil {
		s.logger.Error("update event status to failed", slog.String("error", err.Error()))
	}
}

func (s *DispatcherService) recordAttempt(ctx context.Context, event *domain.Event, result delivery.DeliveryResult) {
	headersJSON, _ := json.Marshal(result.ResponseHeaders)

	attempt := &domain.DeliveryAttempt{
		ID:              uuid.New().String(),
		EventID:         event.ID,
		EndpointID:      event.EndpointID,
		AttemptNumber:   event.AttemptCount + 1,
		StatusCode:      result.StatusCode,
		ResponseBody:    truncate(result.ResponseBody, 1024),
		ResponseHeaders: headersJSON,
		ErrorMessage:    errorMessage(result.Error),
		DurationMs:      int(result.Duration.Milliseconds()),
		CreatedAt:       time.Now().UTC(),
	}

	if err := s.deliveryRepo.Create(ctx, attempt); err != nil {
		s.logger.Error("record delivery attempt", slog.String("error", err.Error()))
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func errorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func failureMessage(result delivery.DeliveryResult) string {
	if result.Error != nil {
		return fmt.Sprintf("connection error: %s", result.Error.Error())
	}
	return fmt.Sprintf("HTTP %d", result.StatusCode)
}
