package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RateLimiterRepo struct {
	client *redis.Client
	window time.Duration
}

func NewRateLimiterRepo(client *redis.Client) *RateLimiterRepo {
	return &RateLimiterRepo{
		client: client,
		window: 1 * time.Minute,
	}
}

func (r *RateLimiterRepo) Allow(ctx context.Context, tenantID string, limit int) (bool, error) {
	key := fmt.Sprintf("rl:%s", tenantID)
	now := time.Now()
	windowStart := now.Add(-r.window).UnixMilli()

	pipe := r.client.Pipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))
	cardCmd := pipe.ZCard(ctx, key)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("rate limit pipeline: %w", err)
	}

	if cardCmd.Val() >= int64(limit) {
		return false, nil
	}

	pipe = r.client.Pipeline()
	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(now.UnixMilli()),
		Member: uuid.New().String(),
	})
	pipe.Expire(ctx, key, r.window+10*time.Second)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("rate limit add: %w", err)
	}

	return true, nil
}
