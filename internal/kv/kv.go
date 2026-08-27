package kv

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrMissing = errors.New("missing")

type KV interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, val string, ttl time.Duration) error
	Del(ctx context.Context, key string) error
	TTL(ctx context.Context, key string) (time.Duration, error)
}

// Redis-backed (local dev / when REDIS_URL set)
type RedisKV struct{ c *redis.Client }

func NewRedis(addr string) *RedisKV {
	return &RedisKV{c: redis.NewClient(&redis.Options{Addr: addr, PoolSize: 20})}
}
func (r *RedisKV) Get(ctx context.Context, k string) (string, error) {
	v, err := r.c.Get(ctx, k).Result()
	if err == redis.Nil {
		return "", ErrMissing
	}
	return v, err
}
func (r *RedisKV) Set(ctx context.Context, k, v string, ttl time.Duration) error { return r.c.Set(ctx, k, v, ttl).Err() }
func (r *RedisKV) Del(ctx context.Context, k string) error                      { return r.c.Del(ctx, k).Err() }
func (r *RedisKV) TTL(ctx context.Context, k string) (time.Duration, error)     { return r.c.TTL(ctx, k).Result() }

// In-memory fallback (Render free tier, no Redis needed)
type memItem struct {
	val string
	exp time.Time
}
type MemKV struct {
	mu sync.RWMutex
	m  map[string]memItem
}

func NewMem() *MemKV { return &MemKV{m: map[string]memItem{}} }
func (m *MemKV) Get(_ context.Context, k string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	it, ok := m.m[k]
	if !ok || (!it.exp.IsZero() && time.Now().After(it.exp)) {
		return "", ErrMissing
	}
	return it.val, nil
}
func (m *MemKV) Set(_ context.Context, k, v string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	m.m[k] = memItem{val: v, exp: exp}
	return nil
}
func (m *MemKV) Del(_ context.Context, k string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.m, k)
	return nil
}
func (m *MemKV) TTL(_ context.Context, k string) (time.Duration, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	it, ok := m.m[k]
	if !ok || (!it.exp.IsZero() && time.Now().After(it.exp)) {
		return 0, ErrMissing
	}
	if it.exp.IsZero() {
		return 0, nil
	}
	return time.Until(it.exp), nil
}
