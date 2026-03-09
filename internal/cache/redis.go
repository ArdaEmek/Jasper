package cache

import (
	"os"
	"sync"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	client *redis.Client
}

var (
	addr     = os.Getenv("REDIS_ADDR")
	password = os.Getenv("REDIS_PASSWORD")

	cache *Cache = nil
	once  sync.Once
)

func New() *Cache {
	once.Do(func() {
		client := redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       0,
		})

		cache = &Cache{client}
	})

	return cache
}
