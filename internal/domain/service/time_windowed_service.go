// Package service provides implementations of domain services that implement core business logic
// This package depends only on domain models and repository interfaces (not implementations)
package service

import (
	"coinStatApp/internal/domain/model"
	"coinStatApp/internal/domain/repository"
	"coinStatApp/internal/domain/useCases"
	"context"
	"sync"
	"time"
)

// TimeWindowedStatisticsService is an implementation of StatisticsService that handles time windows
// for statistics calculations. It implements proper caching, persistence, and time-based cleanup.
// It follows the dependency inversion principle by depending only on repository interfaces.
type TimeWindowedStatisticsService struct {
	mu              sync.RWMutex
	stats           map[string]*model.Statistics     // In-memory statistics cache
	swaps           map[string][]model.Swap          // Store swaps per token for recalculation
	cache           repository.StatisticsCache       // Interface for fast cache operations
	storage         repository.StatisticsPersistence // Interface for persistent storage (optional)
	cleanupInterval time.Duration
}

// NewTimeWindowedStatisticsService creates a new TimeWindowedStatisticsService with the provided
// cache and storage implementations. This constructor follows dependency injection pattern,
// allowing the service to use any implementation that satisfies the required interfaces.
//
// Parameters:
//   - cache: Implementation of StatisticsCache interface for fast data access
//   - storage: Implementation of StatisticsPersistence interface for durable storage (can be nil)
func NewTimeWindowedStatisticsService(cache repository.StatisticsCache, storage repository.StatisticsPersistence) *TimeWindowedStatisticsService {
	s := &TimeWindowedStatisticsService{
		stats:           make(map[string]*model.Statistics),
		swaps:           make(map[string][]model.Swap),
		cache:           cache,   // Inject cache implementation
		storage:         storage, // Inject storage implementation (optional)
		cleanupInterval: 1 * time.Minute,
	}

	// Start background cleanup
	go s.periodicCleanup()

	return s
}

func (s *TimeWindowedStatisticsService) periodicCleanup() {
	ticker := time.NewTicker(s.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		s.expireOldSwaps()
	}
}

func (s *TimeWindowedStatisticsService) expireOldSwaps() {
	now := time.Now()
	windowCutoffs := map[string]time.Time{
		"5min": now.Add(-5 * time.Minute),
		"1h":   now.Add(-1 * time.Hour),
		"24h":  now.Add(-24 * time.Hour),
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// For each token, recalculate stats based on time windows
	for token, swapList := range s.swaps {
		stats := &model.Statistics{Token: token}
		validSwaps := make([]model.Swap, 0, len(swapList))

		for _, swap := range swapList {
			// Keep swap if it's within any window we care about
			if swap.Timestamp.After(windowCutoffs["24h"]) {
				// Count in appropriate windows
				if swap.Timestamp.After(windowCutoffs["5min"]) {
					stats.Volume5Min += swap.USD
					stats.Count5Min++
				}

				if swap.Timestamp.After(windowCutoffs["1h"]) {
					stats.Volume1H += swap.USD
					stats.Count1H++
				}

				stats.Volume24H += swap.USD
				stats.Count24H++

				validSwaps = append(validSwaps, swap)
			}
		}

		// Update last update time
		if len(validSwaps) > 0 {
			stats.LastUpdate = now
		}

		// Store updated statistics and valid swaps
		s.stats[token] = stats
		s.swaps[token] = validSwaps

		// Persist to cache and storage if available
		ctx := context.Background()
		if s.cache != nil {
			_ = s.cache.SaveStatistics(ctx, stats)
		}
		if s.storage != nil {
			_ = s.storage.SaveStatistics(ctx, stats)
		}
	}
}

func (s *TimeWindowedStatisticsService) ProcessSwap(ctx context.Context, swap *model.Swap) error {
	if swap == nil {
		return nil // Ignore nil swaps
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicates by ID
	if _, exists := s.swaps[swap.Token]; !exists {
		s.swaps[swap.Token] = make([]model.Swap, 0)
	} else {
		// Check if this swap already exists by ID
		for _, existingSwap := range s.swaps[swap.Token] {
			if existingSwap.ID == swap.ID {
				// This is a duplicate - ignore it
				return nil
			}
		}
	}

	// Add to swaps list for this token
	s.swaps[swap.Token] = append(s.swaps[swap.Token], *swap)

	// Update statistics
	now := time.Now()
	stats := s.stats[swap.Token]
	if stats == nil {
		stats = &model.Statistics{
			Token:      swap.Token,
			LastUpdate: now,
		}
		s.stats[swap.Token] = stats
	}

	// Always count in the 5min window for real-time processing
	stats.Volume5Min += swap.USD
	stats.Count5Min++

	// Always count in the 1h window for real-time processing
	stats.Volume1H += swap.USD
	stats.Count1H++

	// Always count in the 24h window for real-time processing
	stats.Volume24H += swap.USD
	stats.Count24H++

	stats.LastUpdate = now

	// Save to cache and storage if available
	var err error
	if s.cache != nil {
		err = s.cache.SaveStatistics(ctx, stats)
	}
	if s.storage != nil {
		if storageErr := s.storage.SaveStatistics(ctx, stats); storageErr != nil && err == nil {
			err = storageErr
		}
	}

	return err
}

func (s *TimeWindowedStatisticsService) GetStatistics(ctx context.Context, token string) (*model.Statistics, error) {
	s.mu.RLock()
	inMemoryStats, ok := s.stats[token]
	s.mu.RUnlock()

	// First priority: memory cache
	if ok {
		// Return a copy to prevent external modification
		statsCopy := *inMemoryStats
		return &statsCopy, nil
	}

	// Second priority: cache storage
	if s.cache != nil {
		cacheStats, err := s.cache.GetStatistics(ctx, token)
		if err == nil && cacheStats != nil {
			return cacheStats, nil
		}
	}

	// Third priority: persistent storage
	if s.storage != nil {
		storageStats, err := s.storage.GetStatistics(ctx, token)
		if err == nil && storageStats != nil {
			// Update in-memory cache and Redis cache for future calls
			s.updateCaches(ctx, storageStats)
			return storageStats, nil
		}
	}

	// Not found anywhere
	return nil, nil
}

func (s *TimeWindowedStatisticsService) GetAllStatistics(ctx context.Context) ([]*model.Statistics, error) {
	// First try in-memory cache
	s.mu.RLock()
	inMemoryStats := make(map[string]*model.Statistics)
	for token, stats := range s.stats {
		// Create copies to prevent external modification
		statsCopy := *stats
		inMemoryStats[token] = &statsCopy
	}
	s.mu.RUnlock()

	// Second priority: cache storage
	var cacheStats []*model.Statistics
	if s.cache != nil {
		var err error
		cacheStats, err = s.cache.GetAllStatistics(ctx)
		if err != nil {
			cacheStats = nil
		}
	}

	// Third priority: persistent storage
	var storageStats []*model.Statistics
	if s.storage != nil {
		var err error
		storageStats, err = s.storage.GetAllStatistics(ctx)
		if err != nil {
			storageStats = nil
		}
	}

	// Merge results, with in-memory taking precedence, then cache, then storage
	result := make([]*model.Statistics, 0)
	tokenMap := make(map[string]bool)

	// Add in-memory stats first
	for _, stats := range inMemoryStats {
		result = append(result, stats)
		tokenMap[stats.Token] = true
	}

	// Add cache stats if not already added
	for _, stats := range cacheStats {
		if !tokenMap[stats.Token] {
			result = append(result, stats)
			tokenMap[stats.Token] = true
		}
	}

	// Add storage stats if not already added
	for _, stats := range storageStats {
		if !tokenMap[stats.Token] {
			result = append(result, stats)
			// Update caches for future calls
			s.updateCaches(ctx, stats)
		}
	}

	return result, nil
}

// updateCaches updates both the in-memory cache and the external cache implementation
// with statistics data. This helper method abstracts the caching logic and handles the case
// where the cache implementation might be nil.
func (s *TimeWindowedStatisticsService) updateCaches(ctx context.Context, stats *model.Statistics) {
	// Update in-memory cache
	s.mu.Lock()
	s.stats[stats.Token] = stats
	s.mu.Unlock()

	// Update external cache if available (dependency might be nil)
	if s.cache != nil {
		_ = s.cache.SaveStatistics(ctx, stats)
	}
}

// Ensure interface compliance
var _ useCases.StatisticsService = (*TimeWindowedStatisticsService)(nil)
