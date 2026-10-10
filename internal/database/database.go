package database

import (
	"context"
	"time"

	"github.com/dpcamargo/redis-rate-limiter-go/internal/database/redisdb"
	"github.com/go-redis/redis/v8"
)

type DatabaseInterface interface {
	GetCount(ctx context.Context, key string) (int, error)
	IncrementCount(ctx context.Context, key string, expiration time.Duration) error
	UpdateExpiration(ctx context.Context, key string, expiration time.Duration) error
}

type Database struct {
	db DatabaseInterface
}

func NewDatabase(redisAddr string) DatabaseInterface {
	return &Database{
		db: redisdb.NewRedisClient(redis.Options{
			Addr: redisAddr,
		}),
	}
}

func (d *Database) GetCount(ctx context.Context, key string) (int, error) {
	return d.db.GetCount(ctx, key)
}

func (d *Database) IncrementCount(ctx context.Context, key string, expiration time.Duration) error {
	return d.db.IncrementCount(ctx, key, expiration)
}

func (d *Database) UpdateExpiration(ctx context.Context, key string, expiration time.Duration) error {
	return d.db.IncrementCount(ctx, key, expiration)
}
