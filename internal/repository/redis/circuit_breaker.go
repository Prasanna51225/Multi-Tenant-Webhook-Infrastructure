package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	CBClosed   = "closed"
	CBOpen     = "open"
	CBHalfOpen = "half_open"
)

type CircuitBreakerRepo struct {
	client           *redis.Client
	failureThreshold int64
	timeout          time.Duration
}

func NewCircuitBreakerRepo(client *redis.Client, failureThreshold int, timeout time.Duration) *CircuitBreakerRepo {
	return &CircuitBreakerRepo{
		client:           client,
		failureThreshold: int64(failureThreshold),
		timeout:          timeout,
	}
}

func (r *CircuitBreakerRepo) Allow(ctx context.Context, endpointID string) (bool, string, error) {
	stateKey := fmt.Sprintf("cb:%s:state", endpointID)

	state, err := r.client.Get(ctx, stateKey).Result()
	if err == redis.Nil {
		return true, CBClosed, nil
	}
	if err != nil {
		return false, "", fmt.Errorf("get circuit breaker state: %w", err)
	}

	if state == CBClosed {
		return true, CBClosed, nil
	}

	if state == CBHalfOpen {
		return true, CBHalfOpen, nil
	}

	if state == CBOpen {
		lastFailureKey := fmt.Sprintf("cb:%s:last_failure", endpointID)
		lastFailure, err := r.client.Get(ctx, lastFailureKey).Int64()
		if err != nil && err != redis.Nil {
			return false, CBOpen, fmt.Errorf("get last failure time: %w", err)
		}

		if time.Now().Unix()-lastFailure > int64(r.timeout.Seconds()) {
			r.client.Set(ctx, stateKey, CBHalfOpen, 0)
			return true, CBHalfOpen, nil
		}

		return false, CBOpen, nil
	}

	return true, state, nil
}

func (r *CircuitBreakerRepo) RecordSuccess(ctx context.Context, endpointID string) error {
	stateKey := fmt.Sprintf("cb:%s:state", endpointID)
	failuresKey := fmt.Sprintf("cb:%s:failures", endpointID)

	pipe := r.client.Pipeline()
	pipe.Set(ctx, stateKey, CBClosed, 0)
	pipe.Del(ctx, failuresKey)
	_, err := pipe.Exec(ctx)

	return err
}

func (r *CircuitBreakerRepo) RecordFailure(ctx context.Context, endpointID string) error {
	stateKey := fmt.Sprintf("cb:%s:state", endpointID)
	failuresKey := fmt.Sprintf("cb:%s:failures", endpointID)
	lastFailureKey := fmt.Sprintf("cb:%s:last_failure", endpointID)

	state, _ := r.client.Get(ctx, stateKey).Result()

	if state == CBHalfOpen {
		pipe := r.client.Pipeline()
		pipe.Set(ctx, stateKey, CBOpen, 0)
		pipe.Set(ctx, lastFailureKey, time.Now().Unix(), 0)
		_, err := pipe.Exec(ctx)
		return err
	}

	count, err := r.client.Incr(ctx, failuresKey).Result()
	if err != nil {
		return err
	}

	if count >= r.failureThreshold {
		pipe := r.client.Pipeline()
		pipe.Set(ctx, stateKey, CBOpen, 0)
		pipe.Set(ctx, lastFailureKey, time.Now().Unix(), 0)
		_, err := pipe.Exec(ctx)
		return err
	}

	return nil
}

func (r *CircuitBreakerRepo) GetState(ctx context.Context, endpointID string) (string, error) {
	stateKey := fmt.Sprintf("cb:%s:state", endpointID)
	state, err := r.client.Get(ctx, stateKey).Result()
	if err == redis.Nil {
		return CBClosed, nil
	}
	if err != nil {
		return "", err
	}
	return state, nil
}
