package utils

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	app_error "github.com/xenn00/chat-system/internal/errors"
)

func GetCacheData[T any](ctx context.Context, rdb *redis.Client, cacheKey string) (*T, *app_error.AppError) {
	val, err := rdb.Get(ctx, cacheKey).Result()
	if err == redis.Nil {
		return nil, nil // cache-miss
	} else if err != nil {
		return nil, app_error.NewAppError(http.StatusInternalServerError, "unexpected error occur when trying to get from redis", "redis")
	}

	var data T
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return nil, app_error.NewAppError(http.StatusInternalServerError, "unexpected error occur when unmarshal json", "json")
	}

	return &data, nil
}

func SetCacheData[T any](ctx context.Context, rdb *redis.Client, cacheKey string, data *T, expire time.Duration) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return app_error.NewAppError(http.StatusInternalServerError, "unexpected error occur when marshal json", "json")
	}

	return rdb.Set(ctx, cacheKey, bytes, expire).Err()
}

func DeleteCacheData(ctx context.Context, rdb *redis.Client, cacheKey string) error {
	return rdb.Del(ctx, cacheKey).Err()
}
