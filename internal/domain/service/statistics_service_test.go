package service_test

import (
	"coinStatApp/internal/domain/model"
	"coinStatApp/internal/domain/service"
	"context"
	"testing"
	"time"
)

func TestInMemoryStatisticsService(t *testing.T) {
	// Setup
	ctx := context.Background()
	statsService := service.NewInMemoryStatisticsService()

	// Test data
	now := time.Now()
	swap1 := &model.Swap{
		ID:        "swap1",
		Who:       "user1",
		Token:     "BTC",
		Amount:    1.0,
		USD:       50000.0,
		Side:      "buy",
		Timestamp: now,
	}
	swap2 := &model.Swap{
		ID:        "swap2",
		Who:       "user2",
		Token:     "BTC",
		Amount:    0.5,
		USD:       25000.0,
		Side:      "sell",
		Timestamp: now.Add(1 * time.Minute),
	}

	// Test: Process swaps
	err := statsService.ProcessSwap(ctx, swap1)
	if err != nil {
		t.Fatalf("failed to process first swap: %v", err)
	}
	err = statsService.ProcessSwap(ctx, swap2)
	if err != nil {
		t.Fatalf("failed to process second swap: %v", err)
	}

	// Test: Get statistics for a specific token
	stats, err := statsService.GetStatistics(ctx, "BTC")
	if err != nil {
		t.Fatalf("failed to get statistics: %v", err)
	}

	// Verify statistics
	expectedVolume := swap1.USD + swap2.USD
	if stats.Volume5Min != expectedVolume {
		t.Errorf("expected 5min volume to be %f, got %f", expectedVolume, stats.Volume5Min)
	}
	if stats.Volume1H != expectedVolume {
		t.Errorf("expected 1h volume to be %f, got %f", expectedVolume, stats.Volume1H)
	}
	if stats.Volume24H != expectedVolume {
		t.Errorf("expected 24h volume to be %f, got %f", expectedVolume, stats.Volume24H)
	}

	if stats.Count5Min != 2 {
		t.Errorf("expected 5min count to be 2, got %d", stats.Count5Min)
	}
	if stats.Count1H != 2 {
		t.Errorf("expected 1h count to be 2, got %d", stats.Count1H)
	}
	if stats.Count24H != 2 {
		t.Errorf("expected 24h count to be 2, got %d", stats.Count24H)
	}

	// Test: Get all statistics
	allStats, err := statsService.GetAllStatistics(ctx)
	if err != nil {
		t.Fatalf("failed to get all statistics: %v", err)
	}

	if len(allStats) != 1 {
		t.Errorf("expected 1 token statistic, got %d", len(allStats))
	}
}
