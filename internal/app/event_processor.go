package app

import (
	"coinStatApp/internal/domain/model"
	"context"
	"errors"
	"log"

	"coinStatApp/internal/app/dto"
	"coinStatApp/internal/domain/useCases"
)

// ErrContextCancelled is returned when the context is cancelled during processing
var ErrContextCancelled = errors.New("context cancelled during processing")

// EventProcessor processes swap events from a channel, updates stats, and broadcasts updates.
type EventProcessor struct {
	SwapCh       chan *dto.SwapDTO
	StatsService useCases.StatisticsService
	Broadcaster  useCases.Broadcaster
	DedupCache   map[string]struct{} // simple in-memory deduplication, replace with Redis for HA
}

func NewEventProcessor(swapCh chan *dto.SwapDTO, statsService useCases.StatisticsService, broadcaster Broadcaster) *EventProcessor {
	return &EventProcessor{
		SwapCh:       swapCh,
		StatsService: statsService,
		Broadcaster:  broadcaster,
		DedupCache:   make(map[string]struct{}),
	}
}

func (p *EventProcessor) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case swapDto := <-p.SwapCh:
			if err := p.processSwap(ctx, swapDto); err != nil {
				if errors.Is(err, ErrContextCancelled) {
					log.Println("Context cancelled, stopping event processor")
					return ctx.Err()
				}
				// Other errors are just logged but processing continues
				log.Printf("Error processing swap: %v", err)
			}
		}
	}
}

// processSwap handles a single swap event with proper context cancellation checks
func (p *EventProcessor) processSwap(ctx context.Context, swapDto *dto.SwapDTO) error {
	// Check context before starting
	if ctx.Err() != nil {
		return ErrContextCancelled
	}

	if swapDto == nil {
		return nil
	}

	// Deduplication (replace with Redis for distributed setup)
	if _, exists := p.DedupCache[swapDto.ID]; exists {
		return nil
	}
	p.DedupCache[swapDto.ID] = struct{}{}

	// Convert DTO to domain model
	swap := swapDto.ToModel()

	// Check context before processing
	if ctx.Err() != nil {
		return ErrContextCancelled
	}

	// Process swap
	if err := p.StatsService.ProcessSwap(ctx, swap); err != nil {
		return err
	}

	// Check context before getting statistics
	if ctx.Err() != nil {
		return ErrContextCancelled
	}

	// Get updated stats for token
	stats, err := p.StatsService.GetStatistics(ctx, swap.Token)
	if err != nil {
		return err
	}

	// Check context before broadcasting
	if ctx.Err() != nil {
		return ErrContextCancelled
	}

	// Broadcast updated stats
	p.Broadcaster.BroadcastStatistics(stats)

	return nil
}
