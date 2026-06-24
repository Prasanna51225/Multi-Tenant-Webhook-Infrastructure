package service

import (
    "context"
    "crypto/rand"
    "encoding/base64"
    "fmt"
    "net/url"
    "time"

    "github.com/google/uuid"

    "github.com/webhook-platform/internal/domain"
)

type EndpointRepository interface {
    Create(ctx context.Context, endpoint *domain.Endpoint) error
    GetByID(ctx context.Context, id string) (*domain.Endpoint, error)
    ListByTenantID(ctx context.Context, tenantID string, limit, offset int) ([]*domain.Endpoint, error)
    CountByTenantID(ctx context.Context, tenantID string) (int, error)
    Update(ctx context.Context, endpoint *domain.Endpoint) error
    Delete(ctx context.Context, id string) error
}

type EndpointService interface {
    Create(ctx context.Context, tenantID string, input domain.CreateEndpointInput) (*domain.Endpoint, string, error)
    GetByID(ctx context.Context, id string) (*domain.Endpoint, error)
    ListByTenantID(ctx context.Context, tenantID string, limit, offset int) ([]*domain.Endpoint, int, error)
    Update(ctx context.Context, id string, tenantID string, input domain.UpdateEndpointInput) (*domain.Endpoint, error)
    Delete(ctx context.Context, id string, tenantID string) error
}

type endpointService struct {
    repo EndpointRepository
}

func NewEndpointService(repo EndpointRepository) EndpointService {
    return &endpointService{repo: repo}
}

func (s *endpointService) Create(ctx context.Context, tenantID string, input domain.CreateEndpointInput) (*domain.Endpoint, string, error) {
    if errs := input.Validate(); len(errs) > 0 {
        return nil, "", fmt.Errorf("%w: %v", domain.ErrValidation, errs)
    }

    if err := validateURL(input.URL); err != nil {
        return nil, "", fmt.Errorf("%w: url: %s", domain.ErrValidation, err.Error())
    }

    secret, err := generateSecret()
    if err != nil {
        return nil, "", fmt.Errorf("generate secret: %w", err)
    }

    active := true
    if input.Active != nil {
        active = *input.Active
    }

    eventTypes := input.EventTypes
    if eventTypes == nil {
        eventTypes = []string{}
    }

    now := time.Now().UTC()
    endpoint := &domain.Endpoint{
        ID:          uuid.New().String(),
        TenantID:    tenantID,
        URL:         input.URL,
        Description: input.Description,
        EventTypes:  eventTypes,
        Secret:      secret,
        Active:      active,
        CreatedAt:   now,
        UpdatedAt:   now,
    }

    if err := s.repo.Create(ctx, endpoint); err != nil {
        return nil, "", fmt.Errorf("create endpoint: %w", err)
    }

    return endpoint, secret, nil
}

func (s *endpointService) GetByID(ctx context.Context, id string) (*domain.Endpoint, error) {
    endpoint, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("get endpoint: %w", err)
    }
    return endpoint, nil
}

func (s *endpointService) ListByTenantID(ctx context.Context, tenantID string, limit, offset int) ([]*domain.Endpoint, int, error) {
    if limit <= 0 {
        limit = 50
    }
    if limit > 100 {
        limit = 100
    }
    if offset < 0 {
        offset = 0
    }

    endpoints, err := s.repo.ListByTenantID(ctx, tenantID, limit, offset)
    if err != nil {
        return nil, 0, fmt.Errorf("list endpoints: %w", err)
    }

    count, err := s.repo.CountByTenantID(ctx, tenantID)
    if err != nil {
        return nil, 0, fmt.Errorf("count endpoints: %w", err)
    }

    return endpoints, count, nil
}

func (s *endpointService) Update(ctx context.Context, id string, tenantID string, input domain.UpdateEndpointInput) (*domain.Endpoint, error) {
    endpoint, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("get endpoint: %w", err)
    }

    if endpoint.TenantID != tenantID {
        return nil, domain.ErrForbidden
    }

    if input.URL != nil {
        if err := validateURL(*input.URL); err != nil {
            return nil, fmt.Errorf("%w: url: %s", domain.ErrValidation, err.Error())
        }
        endpoint.URL = *input.URL
    }
    if input.Description != nil {
        endpoint.Description = *input.Description
    }
    if input.EventTypes != nil {
        endpoint.EventTypes = input.EventTypes
    }
    if input.Active != nil {
        endpoint.Active = *input.Active
    }

    endpoint.UpdatedAt = time.Now().UTC()

    if err := s.repo.Update(ctx, endpoint); err != nil {
        return nil, fmt.Errorf("update endpoint: %w", err)
    }

    return endpoint, nil
}

func (s *endpointService) Delete(ctx context.Context, id string, tenantID string) error {
    endpoint, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return fmt.Errorf("get endpoint: %w", err)
    }

    if endpoint.TenantID != tenantID {
        return domain.ErrForbidden
    }

    if err := s.repo.Delete(ctx, id); err != nil {
        return fmt.Errorf("delete endpoint: %w", err)
    }

    return nil
}

func validateURL(raw string) error {
    u, err := url.Parse(raw)
    if err != nil {
        return fmt.Errorf("invalid url: %w", err)
    }
    if u.Scheme != "http" && u.Scheme != "https" {
        return fmt.Errorf("url must use http or https scheme")
    }
    if u.Host == "" {
        return fmt.Errorf("url must have a host")
    }
    return nil
}

func generateSecret() (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return "whsec_" + base64.RawURLEncoding.EncodeToString(b), nil
}