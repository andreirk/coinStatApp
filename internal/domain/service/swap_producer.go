package service

import (
	"coinStatApp/internal/domain/model"
	"coinStatApp/internal/infrastructure/queue"
	"context"
	"log"
)

// SwapProducerUseCase handles publishing swaps to Kafka
type SwapProducerUseCase struct {
	Producer queue.SwapProducer
}

// NewSwapProducerUseCase creates a new use case for publishing swaps
func NewSwapProducerUseCase(producer queue.SwapProducer) *SwapProducerUseCase {
	return &SwapProducerUseCase{
		Producer: producer,
	}
}

// Execute publishes a swap event to Kafka
func (uc *SwapProducerUseCase) Execute(ctx context.Context, swap *model.Swap) error {
	err := uc.Producer.PublishSwap(ctx, swap)
	if err != nil {
		log.Printf("Failed to publish swap to Kafka: %v", err)
		return err
	}
	return nil
}
