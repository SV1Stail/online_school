package redisstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"
)

var loginLimitScript = redis.NewScript(`
local current = redis.call('INCR', KEYS[1])
if current == 1 then
    redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return current
`)

type Client struct {
	client    *redis.Client
	limit     int64
	window    time.Duration
	keyPrefix string
}

func New(address, password string, database int, limit int64, window time.Duration) *Client {
	return &Client{
		client: redis.NewClient(&redis.Options{
			Addr:         address,
			Password:     password,
			DB:           database,
			PoolSize:     5,
			MinIdleConns: 1,
			DialTimeout:  2 * time.Second,
			ReadTimeout:  2 * time.Second,
			WriteTimeout: 2 * time.Second,
		}),
		limit:     limit,
		window:    window,
		keyPrefix: "github.com/SV1Stail/online_school:login:",
	}
}

func (c *Client) Ping(ctx context.Context) error {
	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}

	return nil
}

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) Allow(ctx context.Context, key string) (bool, error) {
	redisKey := c.keyPrefix + hashKey(key)
	value, err := loginLimitScript.Run(ctx, c.client, []string{redisKey}, c.window.Milliseconds()).Int64()
	if err != nil {
		return false, fmt.Errorf("check login limit: %w", err)
	}

	return value <= c.limit, nil
}

func (c *Client) Reset(ctx context.Context, key string) error {
	if err := c.client.Del(ctx, c.keyPrefix+hashKey(key)).Err(); err != nil {
		return fmt.Errorf("reset login limit: %w", err)
	}

	return nil
}

func hashKey(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}
