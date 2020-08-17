package db

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	ctx = context.Background()
	rdb *redis.Client
)

func Init() error {
	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	err := rdb.Ping(ctx).Err()
	return err
}

func SaveData(id string, timestamp time.Time) {
	str := timestamp.Format("15h04")
	rdb.Set(ctx, id, str, 24*time.Hour).Result()
	rdb.Persist(ctx, id)
}

func GetAllData(data map[string]time.Time) {
	keys, _ := rdb.Keys(ctx, "*").Result()
	for _, key := range keys {
		if val, err := rdb.Get(ctx, key).Result(); err != nil {
			panic(err)
		} else {
			if t, err := time.Parse("15h04", val); err != nil {
				//panic(err)
			} else {
				data[key] = t
			}
		}
	}
}
