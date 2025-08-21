package app_test

import (
	"coinStatApp/internal/app"
	"coinStatApp/internal/app/dto"
	"coinStatApp/internal/domain/model"
	"coinStatApp/internal/domain/service"
	"context"
	"sync"
	"testing"
	"time"
)

// MockBroadcaster implements the Broadcaster interface for testing
type MockBroadcaster struct {
	broadcasts []*model.Statistics
	mu         sync.Mutex
}

func NewMockBroadcaster() *MockBroadcaster {
	return &MockBroadcaster{
		broadcasts: make([]*model.Statistics, 0),
	}
}

func (b *MockBroadcaster) BroadcastStatistics(stats *model.Statistics) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.broadcasts = append(b.broadcasts, stats)
}

func (b *MockBroadcaster) GetBroadcasts() []*model.Statistics {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.broadcasts
}

func TestEventProcessor(t *testing.T) {
	// Setup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	swapCh := make(chan *dto.SwapDTO, 10)
	statsService := service.NewInMemoryStatisticsService()
	broadcaster := NewMockBroadcaster()

	// Create processor
	processor := app.NewEventProcessor(swapCh, statsService, broadcaster)

	// Start processor in background
	go processor.Run(ctx)

	// Send test events
	now := time.Now()
	swapCh <- &dto.SwapDTO{ID: "swap1", Token: "ETH", Amount: 1, USD: 3000, Side: "buy", Who: "user1", Timestamp: now}
	swapCh <- &dto.SwapDTO{ID: "swap2", Token: "BTC", Amount: 0.1, USD: 5000, Side: "sell", Who: "user2", Timestamp: now}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Verify statistics were processed
	ethStats, err := statsService.GetStatistics(ctx, "ETH")
	if err != nil {
		t.Fatalf("failed to get ETH stats: %v", err)
	}
	if ethStats == nil {
		t.Fatal("ETH stats not found")
	}
	if ethStats.Volume5Min != 3000 {
		t.Errorf("expected ETH volume to be 3000, got %f", ethStats.Volume5Min)
	}

	btcStats, err := statsService.GetStatistics(ctx, "BTC")
	if err != nil {
		t.Fatalf("failed to get BTC stats: %v", err)
	}
	if btcStats == nil {
		t.Fatal("BTC stats not found")
	}
	if btcStats.Volume5Min != 5000 {
		t.Errorf("expected BTC volume to be 5000, got %f", btcStats.Volume5Min)
	}

	// Test deduplication
	swapCh <- &dto.SwapDTO{ID: "swap1", Token: "ETH", Amount: 1, USD: 3000, Side: "buy", Who: "user1", Timestamp: now}
	time.Sleep(100 * time.Millisecond)

	// Stats should not change due to deduplication
	ethStats, _ = statsService.GetStatistics(ctx, "ETH")
	if ethStats.Volume5Min != 3000 {
		t.Errorf("duplication prevention failed: expected ETH volume to be 3000, got %f", ethStats.Volume5Min)
	}

	// Verify broadcasts happened
	broadcasts := broadcaster.GetBroadcasts()
	if len(broadcasts) < 2 {
		t.Fatalf("expected at least 2 broadcasts, got %d", len(broadcasts))
	}
}
