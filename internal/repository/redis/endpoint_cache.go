package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/webhook-platform/internal/domain"
)

type EndpointCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewEndpointCache(client *redis.Client) *EndpointCache {
	return &EndpointCache{
		client: client,
		ttl:    60 * time.Second,
	}
}

func (c *EndpointCache) Get(ctx context.Context, endpointID string) (*domain.Endpoint, error) {
	key := fmt.Sprintf("cache:endpoint:%s", endpointID)
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cache get: %w", err)
	}

	var ep domain.Endpoint
	if err := json.Unmarshal([]byte(val), &ep); err != nil {
		return nil, fmt.Errorf("cache unmarshal: %w", err)
	}
	return &ep, nil
}

func (c *EndpointCache) Set(ctx context.Context, endpoint *domain.Endpoint) error {
	key := fmt.Sprintf("cache:endpoint:%s", endpoint.ID)
	val, err := json.Marshal(endpoint)
	if err != nil {
		return fmt.Errorf("cache marshal: %w", err)
	}
	return c.client.Set(ctx, key, val, c.ttl).Err()
}

func (c *EndpointCache) Invalidate(ctx context.Context, endpointID string) error {
	key := fmt.Sprintf("cache:endpoint:%s", endpointID)
	return c.client.Del(ctx, key).Err()
}
