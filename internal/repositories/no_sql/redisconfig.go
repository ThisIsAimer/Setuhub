package nosql

import (
	"crypto/tls"
	"os"

	"github.com/redis/go-redis/v9"
)

func RedisCliant() (*redis.Client, error) {
	redisURL := os.Getenv("REDIS_URL")

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	// Upstash requires TLS (rediss://)
	opt.TLSConfig = &tls.Config{}

	rdb := redis.NewClient(opt)

	return rdb, nil
}
