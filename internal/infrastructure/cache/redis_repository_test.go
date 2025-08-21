package cache_test

import (
	"coinStatApp/config"
	"coinStatApp/internal/domain/model"
	"coinStatApp/internal/infrastructure/cache"
	"context"
	"testing"
	"time"
)

func TestRedisRepository(t *testing.T) {
	// Load test config
	cfg := config.LoadConfig()

	// Initialize repository
	repo := cache.NewRedisRepository(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)

	// Create test statistics
	ctx := context.Background()
	stats := &model.Statistics{
		Token:      "TEST",
		Volume5Min: 1000.0,
		Volume1H:   5000.0,
		Volume24H:  10000.0,
		Count5Min:  10,
		Count1H:    50,
		Count24H:   100,
		LastUpdate: time.Now(),
	}

	// Test SaveStatistics
	err := repo.SaveStatistics(ctx, stats)
	if err != nil {
		t.Fatalf("Failed to save statistics: %v", err)
	}

	// Test GetStatistics
	retrieved, err := repo.GetStatistics(ctx, "TEST")
	if err != nil {
		t.Fatalf("Failed to get statistics: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Retrieved statistics is nil")
	}

	if retrieved.Token != stats.Token {
		t.Errorf("Expected token %s, got %s", stats.Token, retrieved.Token)
	}

	if retrieved.Volume5Min != stats.Volume5Min {
		t.Errorf("Expected Volume5Min %f, got %f", stats.Volume5Min, retrieved.Volume5Min)
	}

	// Test GetAllStatistics
	all, err := repo.GetAllStatistics(ctx)
	if err != nil {
		t.Fatalf("Failed to get all statistics: %v", err)
	}

	if len(all) < 1 {
		t.Error("Expected at least one statistics entry")
	}
}
