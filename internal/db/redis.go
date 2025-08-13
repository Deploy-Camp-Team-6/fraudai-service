package db

import (
	redis "github.com/redis/go-redis/v9"
)

// NewRedisClient creates a new Redis client.
func NewRedisClient(addr, password string, db int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}
