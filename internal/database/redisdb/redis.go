package redisdb

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisClient struct {
	Client *redis.Client
}

func NewRedisClient(conf redis.Options) *RedisClient {
	client := redis.NewClient(&conf)
	pong, err := client.Ping(context.Background()).Result()
	if err != nil {
		log.Fatalf("Could not ping Redis: %v", err)
	}
	fmt.Println("Response from Redis:", pong)

	return &RedisClient{
		Client: client,
	}
}

func (r *RedisClient) IncrementCount(ctx context.Context, key string, expiration time.Duration) error {
	err := r.Client.Incr(ctx, key).Err()
	if err != nil {
		return err
	}

	err = r.Client.Expire(ctx, key, expiration).Err()
	if err != nil {
		return err
	}

	return nil
}

func (r *RedisClient) UpdateExpiration(ctx context.Context, key string, expiration time.Duration) error {
	err := r.Client.Expire(ctx, key, expiration).Err()
	if err != nil {
		return err
	}

	return nil
}

func (r *RedisClient) GetCount(ctx context.Context, key string) (int, error) {
	result, err := r.Client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			result = "0"
		} else {
			return 0, err
		}
	}
	val, err := strconv.Atoi(result)
	if err != nil {
		return 0, err
	}
	return val, err
}