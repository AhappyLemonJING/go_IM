package test

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()

func TestGet(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       0,
	})

	s, err := rdb.Get(ctx, "key").Result()
	if err != nil {
		panic(err)
	}
	println(s)
}

func TestSet(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       0,
	})

	err := rdb.Set(ctx, "key", "value", time.Second*15).Err()
	if err != nil {
		panic(err)
	}
}
