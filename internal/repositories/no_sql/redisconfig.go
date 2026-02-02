package nosql

import (
	"context"
	"os"

	"github.com/redis/go-redis/v9"
)

type noopLogger struct{}

func (noopLogger) Printf(ctx context.Context, format string, v ...interface{}) {}

func RedisCliant() (*redis.Client, error) {
	redisURL := os.Getenv("REDIS_URL")
	redis.SetLogger(noopLogger{})

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	// Upstash requires TLS (rediss://)
	// opt.TLSConfig = &tls.Config{}

	rdb := redis.NewClient(opt)

	return rdb, nil
}
