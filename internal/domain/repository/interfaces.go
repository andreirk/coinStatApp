// Package repository defines all the repository interfaces used by domain services
// Following the dependency inversion principle, domain logic depends on these interfaces,
// and infrastructure implementations provide concrete implementations
package repository

import (
	"coinStatApp/internal/domain/model"
	"context"
)

// StatisticsCache defines the interface for caching statistics
// This is used for high-performance, in-memory or near-memory storage
// Implementations should prioritize speed over durability
type StatisticsCache interface {
	// SaveStatistics saves statistics in the cache
	// This should be optimized for quick writes and should not block
	SaveStatistics(ctx context.Context, stats *model.Statistics) error

	// GetStatistics retrieves statistics for a specific token from the cache
	// This should be optimized for quick reads
	GetStatistics(ctx context.Context, token string) (*model.Statistics, error)

	// GetAllStatistics retrieves all statistics from the cache
	// This should return all available statistics quickly for API responses
	GetAllStatistics(ctx context.Context) ([]*model.Statistics, error)
}

// StatisticsPersistence defines the interface for persistent statistics storage
// This is used for durable, long-term storage of statistics
// Implementations should prioritize durability and consistency over speed
type StatisticsPersistence interface {
	// SaveStatistics persists statistics to durable storage
	// This operation may be slower but should ensure data isn't lost
	SaveStatistics(ctx context.Context, stats *model.Statistics) error

	// GetStatistics retrieves statistics for a specific token from persistence
	// This should return the most consistent and accurate data available
	GetStatistics(ctx context.Context, token string) (*model.Statistics, error)

	// GetAllStatistics retrieves all statistics from persistence
	// This might be used for analytics, reporting, or initializing the cache
	GetAllStatistics(ctx context.Context) ([]*model.Statistics, error)
}

// EventPersistence defines the interface for persistent event storage
// This is used for storing individual events (swaps) for historical analysis
// and audit purposes
type EventPersistence interface {
	// SaveSwap persists a swap event to durable storage
	// This should store the full details of the swap for later analysis
	SaveSwap(ctx context.Context, swap *model.Swap) error

	// GetSwapsSince retrieves swap events since the given timestamp
	// This is useful for rebuilding state or analyzing historical data
	GetSwapsSince(ctx context.Context, since int64) ([]*model.Swap, error)
}
