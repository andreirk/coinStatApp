package cache

import (
	"coinStatApp/internal/domain/model"
	"coinStatApp/internal/domain/repository"
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// RedisRepository implements the StatisticsCache interface using Redis as the backend
// It provides fast access to statistics data with configurable TTL
type RedisRepository struct {
	client *redis.Client
}

func NewRedisRepository(addr, password string, db int) *RedisRepository {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	return &RedisRepository{client: client}
}

// Ensure RedisRepository implements the StatisticsCache interface
var _ repository.StatisticsCache = (*RedisRepository)(nil)

// Generic Redis methods
func (r *RedisRepository) Set(ctx context.Context, key string, value interface{}) error {
	return r.client.Set(ctx, key, value, 0).Err()
}

func (r *RedisRepository) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

// StatisticsStorage interface implementation
func (r *RedisRepository) SaveStatistics(ctx context.Context, stats *model.Statistics) error {
	data, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("failed to marshal statistics: %w", err)
	}

	key := fmt.Sprintf("stats:%s", stats.Token)
	err = r.client.Set(ctx, key, data, 0).Err()
	return err
}

func (r *RedisRepository) GetStatistics(ctx context.Context, token string) (*model.Statistics, error) {
	key := fmt.Sprintf("stats:%s", token)
	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Token not found
		}
		return nil, err
	}

	var stats model.Statistics
	if err := json.Unmarshal([]byte(data), &stats); err != nil {
		return nil, fmt.Errorf("failed to unmarshal statistics: %w", err)
	}

	return &stats, nil
}

func (r *RedisRepository) GetAllStatistics(ctx context.Context) ([]*model.Statistics, error) {
	// Get all keys matching the stats pattern
	keys, err := r.client.Keys(ctx, "stats:*").Result()
	if err != nil {
		return nil, err
	}

	if len(keys) == 0 {
		return []*model.Statistics{}, nil
	}

	// Get all values in a pipeline for efficiency
	pipe := r.client.Pipeline()
	cmds := make([]*redis.StringCmd, len(keys))

	for i, key := range keys {
		cmds[i] = pipe.Get(ctx, key)
	}

	_, err = pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, err
	}

	// Process results
	result := make([]*model.Statistics, 0, len(keys))
	for _, cmd := range cmds {
		if cmd.Err() != nil && cmd.Err() != redis.Nil {
			continue // Skip failed keys
		}

		data, err := cmd.Result()
		if err != nil {
			continue // Skip error
		}

		var stats model.Statistics
		if err := json.Unmarshal([]byte(data), &stats); err != nil {
			continue // Skip malformed data
		}

		result = append(result, &stats)
	}

	return result, nil
}
