package app

import (
	"coinStatApp/internal/domain/model"
	"coinStatApp/internal/domain/useCases"
	ws "coinStatApp/internal/handlers/websocket"
	"coinStatApp/internal/infrastructure/queue"
	"context"
	"log"
	"time"
)

// KafkaBroadcaster defines interface for pushing updates to WebSocket/API layers.
type KafkaBroadcaster interface {
	BroadcastStatistics(stats *model.Statistics)
}

// KafkaEventProcessor implements the EventProcessor interface for Kafka
type KafkaEventProcessor struct {
	SwapConsumer  queue.SwapConsumer
	StatsService  useCases.StatisticsService
	Broadcaster   *ws.WebSocketBroadcaster
	DedupCache    map[string]struct{}
	cleanupTicker *time.Ticker
}

// cleanupDedupCache periodically cleans up the deduplication cache
func (p *KafkaEventProcessor) cleanupDedupCache() {
	for range p.cleanupTicker.C {
		p.DedupCache = make(map[string]struct{})
		log.Println("Deduplication cache cleaned up")
	}
}

// Run starts the Kafka event processor
func (p *KafkaEventProcessor) Run(ctx context.Context) error {
	// Subscribe to Kafka topic
	swapCh, err := p.SwapConsumer.Subscribe(ctx)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			p.cleanupTicker.Stop()
			return ctx.Err()
		case swap := <-swapCh:
			if swap == nil {
				continue
			}

			// Check context before processing
			if ctx.Err() != nil {
				return ctx.Err()
			}

			// Process swap with context cancellation checks
			p.processSwap(ctx, swap)

			// Commit the message
			if err := p.SwapConsumer.Commit(ctx, swap); err != nil && ctx.Err() == nil {
				log.Printf("Failed to commit swap: %v", err)
			}
		}
	}
}

// processSwap handles a single swap event with context cancellation checks
func (p *KafkaEventProcessor) processSwap(ctx context.Context, swap *model.Swap) {
	// Check if already processed (deduplication)
	_, exists := p.DedupCache[swap.ID]

	if exists {
		return
	}

	// Add to dedup cache
	p.DedupCache[swap.ID] = struct{}{}

	// Check context before processing
	if ctx.Err() != nil {
		return
	}

	// Process swap
	if err := p.StatsService.ProcessSwap(ctx, swap); err != nil {
		log.Printf("Failed to process swap: %v", err)
		return
	}

	// Check context before getting stats
	if ctx.Err() != nil {
		return
	}

	// Get updated stats for token
	stats, err := p.StatsService.GetStatistics(ctx, swap.Token)
	if err != nil {
		log.Printf("Failed to get statistics: %v", err)
		return
	}

	// Check context before broadcasting
	if ctx.Err() != nil {
		return
	}

	// Broadcast updated stats
	p.Broadcaster.BroadcastStatistics(stats)
}
