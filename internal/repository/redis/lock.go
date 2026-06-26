package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type LockRepo struct {
	client *redis.Client
}

func NewLockRepo(client *redis.Client) *LockRepo {
	return &LockRepo{client: client}
}

func (l *LockRepo) Acquire(ctx context.Context, resourceID string, ttl time.Duration) (string, bool) {
	key := fmt.Sprintf("lock:%s", resourceID)
	val := uuid.New().String()
	ok, err := l.client.SetNX(ctx, key, val, ttl).Result()
	if err != nil {
		return "", false
	}
	return val, ok
}

func (l *LockRepo) Release(ctx context.Context, resourceID string, val string) bool {
	key := fmt.Sprintf("lock:%s", resourceID)
	curr, err := l.client.Get(ctx, key).Result()
	if err != nil {
		return false
	}
	if curr == val {
		l.client.Del(ctx, key)
		return true
	}
	return false
}
