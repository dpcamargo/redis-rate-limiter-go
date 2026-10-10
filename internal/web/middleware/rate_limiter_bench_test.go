package middleware

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/dpcamargo/redis-rate-limiter-go/internal/database"
	"github.com/go-redis/redis/v8"
)

// benchDatabase connects straight to a real Redis instance so the numbers
// reported by `go test -bench` reflect actual network + Redis round-trip
// cost (INCR + EXPIRE), not a mock. It is skipped automatically if Redis
// isn't reachable (e.g. CI without `docker compose up redis`).
//
// Run it with:
//
//	docker compose up -d redis
//	REDIS_ADDR=localhost:6379 go test ./internal/web/middleware/... \
//	  -bench=. -benchtime=3s -run=^$ -benchmem
func benchDatabase(b *testing.B) database.DatabaseInterface {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	client := redis.NewClient(&redis.Options{Addr: addr})
	if err := client.Ping(context.Background()).Err(); err != nil {
		b.Skipf("redis not reachable at %s, skipping benchmark: %v", addr, err)
	}
	client.Close()

	return database.NewDatabase(addr)
}

// BenchmarkRateLimiterCore_Allowed measures the end-to-end cost of a single
// allowed request (GetCount + Incr + Expire against Redis) with a limit high
// enough that every call is accepted, isolating the hot path from the
// rejection path below.
func BenchmarkRateLimiterCore_Allowed(b *testing.B) {
	db := benchDatabase(b)
	rl := NewRateLimiter(db)
	key := fmt.Sprintf("bench-allowed-%d", time.Now().UnixNano())
	rl.AddTokenConf(key, b.N+1, time.Minute)

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := RateLimiterCore(ctx, rl, key, false); err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

// BenchmarkRateLimiterCore_Rejected measures the cost of the rejection path
// (GetCount + Expire), which every request hits once the limit is exceeded.
func BenchmarkRateLimiterCore_Rejected(b *testing.B) {
	db := benchDatabase(b)
	rl := NewRateLimiter(db)
	key := fmt.Sprintf("bench-rejected-%d", time.Now().UnixNano())
	rl.AddTokenConf(key, 0, time.Minute) // limit 0 => always rejected

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := RateLimiterCore(ctx, rl, key, false); err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

// BenchmarkRateLimiterCore_Parallel drives the allowed path from multiple
// goroutines to approximate the throughput the middleware can sustain under
// concurrent load, since GOMAXPROCS goroutines each hammer Redis over the
// same connection pool. Use -cpu to vary concurrency.
func BenchmarkRateLimiterCore_Parallel(b *testing.B) {
	db := benchDatabase(b)
	rl := NewRateLimiter(db)
	key := fmt.Sprintf("bench-parallel-%d", time.Now().UnixNano())
	rl.AddTokenConf(key, b.N+1, time.Minute)

	ctx := context.Background()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := RateLimiterCore(ctx, rl, key, false); err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	})
}
