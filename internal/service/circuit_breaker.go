package service

import (
	"context"
	"fmt"

	"github.com/webhook-platform/internal/repository/redis"
)

type CircuitBreakerService interface {
	Allow(ctx context.Context, endpointID string) (bool, string, error)
	RecordSuccess(ctx context.Context, endpointID string) error
	RecordFailure(ctx context.Context, endpointID string) error
	GetState(ctx context.Context, endpointID string) (string, error)
}

type circuitBreakerService struct {
	repo *redis.CircuitBreakerRepo
}

func NewCircuitBreakerService(repo *redis.CircuitBreakerRepo) CircuitBreakerService {
	return &circuitBreakerService{repo: repo}
}

func (s *circuitBreakerService) Allow(ctx context.Context, endpointID string) (bool, string, error) {
	allowed, state, err := s.repo.Allow(ctx, endpointID)
	if err != nil {
		return false, "", fmt.Errorf("circuit breaker allow: %w", err)
	}
	return allowed, state, nil
}

func (s *circuitBreakerService) RecordSuccess(ctx context.Context, endpointID string) error {
	return s.repo.RecordSuccess(ctx, endpointID)
}

func (s *circuitBreakerService) RecordFailure(ctx context.Context, endpointID string) error {
	return s.repo.RecordFailure(ctx, endpointID)
}

func (s *circuitBreakerService) GetState(ctx context.Context, endpointID string) (string, error) {
	return s.repo.GetState(ctx, endpointID)
}
