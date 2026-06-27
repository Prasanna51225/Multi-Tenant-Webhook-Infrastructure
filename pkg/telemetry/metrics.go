package telemetry

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	EventsPublished = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "webhook_events_published_total",
			Help: "Total number of events published",
		},
		[]string{"tenant_id", "event_type"},
	)

	EventsDelivered = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "webhook_events_delivered_total",
			Help: "Total number of events delivered successfully",
		},
		[]string{"endpoint_id", "status_code"},
	)

	EventsFailed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "webhook_events_failed_total",
			Help: "Total number of events that failed delivery",
		},
		[]string{"endpoint_id"},
	)

	DeliveryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "webhook_delivery_duration_seconds",
			Help:    "Duration of webhook delivery attempts in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"endpoint_id"},
	)

	CircuitBreakerState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "webhook_circuit_breaker_state",
			Help: "Circuit breaker state: 0=closed, 1=open, 2=half_open",
		},
		[]string{"endpoint_id"},
	)

	KafkaMessagesConsumed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "webhook_kafka_messages_consumed_total",
			Help: "Total number of Kafka messages consumed",
		},
		[]string{"topic", "group"},
	)

	RateLimitExceeded = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "webhook_rate_limit_exceeded_total",
			Help: "Total number of rate limit exceeded events",
		},
		[]string{"tenant_id"},
	)
)

func init() {
	prometheus.MustRegister(EventsPublished)
	prometheus.MustRegister(EventsDelivered)
	prometheus.MustRegister(EventsFailed)
	prometheus.MustRegister(DeliveryDuration)
	prometheus.MustRegister(CircuitBreakerState)
	prometheus.MustRegister(KafkaMessagesConsumed)
	prometheus.MustRegister(RateLimitExceeded)
}

func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
