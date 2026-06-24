package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/webhook-platform/internal/domain"
)

type TenantRepository interface {
	Create(ctx context.Context, tenant *domain.Tenant) error
	GetByID(ctx context.Context, id string) (*domain.Tenant, error)
	GetByAPIKey(ctx context.Context, apiKey string) (*domain.Tenant, error)
	Update(ctx context.Context, tenant *domain.Tenant) error
}

type TenantService interface {
	Create(ctx context.Context, input domain.CreateTenantInput) (*domain.Tenant, string, error)
	GetByID(ctx context.Context, id string) (*domain.Tenant, error)
	GetByAPIKey(ctx context.Context, apiKey string) (*domain.Tenant, error)
	Update(ctx context.Context, id string, input domain.UpdateTenantInput) (*domain.Tenant, error)
}

type tenantService struct {
	repo TenantRepository
}

func NewTenantService(repo TenantRepository) TenantService {
	return &tenantService{repo: repo}
}

func (s *tenantService) Create(ctx context.Context, input domain.CreateTenantInput) (*domain.Tenant, string, error) {
	if errs := input.Validate(); len(errs) > 0 {
		return nil, "", fmt.Errorf("%w: %v", domain.ErrValidation, errs)
	}

	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, "", fmt.Errorf("generate api key: %w", err)
	}

	rateLimit := input.RateLimitPerMinute
	if rateLimit == 0 {
		rateLimit = 1000
	}

	maxRetries := input.MaxRetries
	if maxRetries == 0 {
		maxRetries = 5
	}

	retryBaseMs := input.RetryBaseMs
	if retryBaseMs == 0 {
		retryBaseMs = 1000
	}

	now := time.Now().UTC()
	tenant := &domain.Tenant{
		ID:                 uuid.New().String(),
		Name:               input.Name,
		APIKey:             apiKey,
		RateLimitPerMinute: rateLimit,
		MaxRetries:         maxRetries,
		RetryBaseMs:        retryBaseMs,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := s.repo.Create(ctx, tenant); err != nil {
		return nil, "", fmt.Errorf("create tenant: %w", err)
	}

	return tenant, apiKey, nil
}

func (s *tenantService) GetByID(ctx context.Context, id string) (*domain.Tenant, error) {
	tenant, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get tenant by id: %w", err)
	}
	return tenant, nil
}

func (s *tenantService) GetByAPIKey(ctx context.Context, apiKey string) (*domain.Tenant, error) {
	tenant, err := s.repo.GetByAPIKey(ctx, apiKey)
	if err != nil {
		return nil, fmt.Errorf("get tenant by api key: %w", err)
	}
	return tenant, nil
}

func (s *tenantService) Update(ctx context.Context, id string, input domain.UpdateTenantInput) (*domain.Tenant, error) {
	tenant, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get tenant: %w", err)
	}

	if input.Name != nil {
		if *input.Name == "" || len(*input.Name) > 255 {
			return nil, fmt.Errorf("%w: name must be 1-255 characters", domain.ErrValidation)
		}
		tenant.Name = *input.Name
	}
	if input.RateLimitPerMinute != nil {
		if *input.RateLimitPerMinute < 0 {
			return nil, fmt.Errorf("%w: rate_limit_per_minute must be non-negative", domain.ErrValidation)
		}
		tenant.RateLimitPerMinute = *input.RateLimitPerMinute
	}
	if input.MaxRetries != nil {
		if *input.MaxRetries < 0 || *input.MaxRetries > 10 {
			return nil, fmt.Errorf("%w: max_retries must be between 0 and 10", domain.ErrValidation)
		}
		tenant.MaxRetries = *input.MaxRetries
	}
	if input.RetryBaseMs != nil {
		if *input.RetryBaseMs < 100 {
			return nil, fmt.Errorf("%w: retry_base_ms must be at least 100", domain.ErrValidation)
		}
		tenant.RetryBaseMs = *input.RetryBaseMs
	}

	tenant.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, tenant); err != nil {
		return nil, fmt.Errorf("update tenant: %w", err)
	}

	return tenant, nil
}

func generateAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "whpk_live_" + base64.RawURLEncoding.EncodeToString(b), nil
}
