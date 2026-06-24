package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
)

type EventMessage struct {
	EventID    string          `json:"event_id"`
	TenantID   string          `json:"tenant_id"`
	EndpointID string          `json:"endpoint_id"`
	EventType  string          `json:"event_type"`
	Payload    json.RawMessage `json:"payload"`
	Signature  string          `json:"signature"`
	Timestamp  time.Time       `json:"timestamp"`
}

type Producer struct {
	writer *kafka.Writer
	logger *slog.Logger
}

func NewProducer(brokers []string, topic string, logger *slog.Logger) *Producer {
	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.Hash{},
		BatchTimeout: 10 * time.Millisecond,
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
		RequiredAcks: kafka.RequireOne,
	}

	return &Producer{
		writer: w,
		logger: logger,
	}
}

func (p *Producer) PublishEvent(ctx context.Context, eventID string, tenantID string, endpointID string, eventType string, payload []byte, signature string) error {
	msg := EventMessage{
		EventID:    eventID,
		TenantID:   tenantID,
		EndpointID: endpointID,
		EventType:  eventType,
		Payload:    payload,
		Signature:  signature,
		Timestamp:  time.Now().UTC(),
	}

	value, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal event message: %w", err)
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(tenantID),
		Value: value,
	})
	if err != nil {
		return fmt.Errorf("write kafka message: %w", err)
	}

	p.logger.Info("event published to kafka",
		slog.String("event_id", eventID),
		slog.String("tenant_id", tenantID),
		slog.String("endpoint_id", endpointID),
		slog.String("event_type", eventType),
	)

	return nil
}

func (p *Producer) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}
