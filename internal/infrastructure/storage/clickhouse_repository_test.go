package storage_test

import (
	"coinStatApp/config"
	"coinStatApp/internal/domain/model"
	"coinStatApp/internal/infrastructure/storage"
	"context"
	"testing"
	"time"
)

func TestClickHouseRepository(t *testing.T) {
	t.Skip("Skipping ClickHouse test - requires live ClickHouse instance")

	// Load test config
	cfg := config.LoadConfig()

	// Initialize repository
	repo, err := storage.NewClickHouseRepository(cfg.ClickHouseDSN)
	if err != nil {
		t.Fatalf("Failed to connect to ClickHouse: %v", err)
	}

	// Create test swap
	ctx := context.Background()
	swap := &model.Swap{
		ID:        "test-swap-1",
		Who:       "test-user",
		Token:     "TEST",
		Amount:    1.0,
		USD:       1000.0,
		Side:      "buy",
		Timestamp: time.Now(),
	}

	// Test SaveSwap
	err = repo.SaveSwap(ctx, swap)
	if err != nil {
		t.Fatalf("Failed to save swap: %v", err)
	}

	// Test GetSwapsSince
	since := time.Now().Add(-1 * time.Hour)
	swaps, err := repo.GetSwapsSince(ctx, since.Unix())
	if err != nil {
		t.Fatalf("Failed to get swaps: %v", err)
	}

	found := false
	for _, s := range swaps {
		if s.ID == swap.ID {
			found = true
			break
		}
	}

	if !found {
		t.Error("Saved swap not found in retrieved swaps")
	}
}
