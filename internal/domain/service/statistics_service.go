package service

import (
	"coinStatApp/internal/domain/model"
	"context"
	"fmt"
	"sync"
	"time"
)

// InMemoryStatisticsService is a simple in-memory implementation of StatisticsService for demo/testing.
// Note: This implementation correctly considers time windows when processing swaps,
// but it does not perform periodic cleanup. For production use, prefer TimeWindowedStatisticsService.
type InMemoryStatisticsService struct {
	stats map[string]*model.Statistics
	mutex sync.RWMutex            // For thread safety
	swaps map[string][]model.Swap // For tracking historical swaps
}

func NewInMemoryStatisticsService() *InMemoryStatisticsService {
	return &InMemoryStatisticsService{
		stats: make(map[string]*model.Statistics),
		swaps: make(map[string][]model.Swap),
	}
}

func (s *InMemoryStatisticsService) ProcessSwap(ctx context.Context, swap *model.Swap) error {
	if swap == nil {
		return fmt.Errorf("cannot process nil swap")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Store the swap for future recalculation
	if _, exists := s.swaps[swap.Token]; !exists {
		s.swaps[swap.Token] = make([]model.Swap, 0)
	}
	s.swaps[swap.Token] = append(s.swaps[swap.Token], *swap)

	// Get or create statistics object
	st, ok := s.stats[swap.Token]
	if !ok {
		st = &model.Statistics{
			Token:      swap.Token,
			LastUpdate: swap.Timestamp,
		}
		s.stats[swap.Token] = st
	}

	// Check time windows and update statistics accordingly
	now := time.Now()

	// Update 5min window if swap is within the last 5 minutes
	if swap.Timestamp.After(now.Add(-5 * time.Minute)) {
		st.Volume5Min += swap.USD
		st.Count5Min++
	}

	// Update 1h window if swap is within the last hour
	if swap.Timestamp.After(now.Add(-1 * time.Hour)) {
		st.Volume1H += swap.USD
		st.Count1H++
	}

	// Update 24h window if swap is within the last 24 hours
	if swap.Timestamp.After(now.Add(-24 * time.Hour)) {
		st.Volume24H += swap.USD
		st.Count24H++
	}

	st.LastUpdate = now
	return nil
}

func (s *InMemoryStatisticsService) GetStatistics(ctx context.Context, token string) (*model.Statistics, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Recalculate statistics to ensure they're fresh
	s.recalculateStatsForToken(token)

	st, ok := s.stats[token]
	if !ok {
		return nil, nil
	}

	// Return a copy to prevent external modification
	statsCopy := *st
	return &statsCopy, nil
}

func (s *InMemoryStatisticsService) GetAllStatistics(ctx context.Context) ([]*model.Statistics, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Recalculate all statistics to ensure they're fresh
	for token := range s.stats {
		s.recalculateStatsForToken(token)
	}

	result := make([]*model.Statistics, 0, len(s.stats))
	for _, st := range s.stats {
		// Create copies to prevent external modification
		statsCopy := *st
		result = append(result, &statsCopy)
	}
	return result, nil
}

// recalculateStatsForToken recalculates statistics for a token based on stored swaps
// Note: This function assumes the mutex is already locked
func (s *InMemoryStatisticsService) recalculateStatsForToken(token string) {
	swapList, exists := s.swaps[token]
	if !exists {
		return
	}

	// Create a new statistics object or get the existing one
	stats, exists := s.stats[token]
	if !exists {
		stats = &model.Statistics{Token: token}
		s.stats[token] = stats
	}

	// Reset counters
	stats.Volume5Min = 0
	stats.Volume1H = 0
	stats.Volume24H = 0
	stats.Count5Min = 0
	stats.Count1H = 0
	stats.Count24H = 0

	// Calculate time windows
	now := time.Now()
	window5min := now.Add(-5 * time.Minute)
	window1h := now.Add(-1 * time.Hour)
	window24h := now.Add(-24 * time.Hour)

	// Calculate statistics based on time windows
	for _, swap := range swapList {
		// Only consider swaps within the 24h window
		if swap.Timestamp.After(window24h) {
			// Count in appropriate windows
			if swap.Timestamp.After(window5min) {
				stats.Volume5Min += swap.USD
				stats.Count5Min++
			}

			if swap.Timestamp.After(window1h) {
				stats.Volume1H += swap.USD
				stats.Count1H++
			}

			stats.Volume24H += swap.USD
			stats.Count24H++
		}
	}

	// Update the last update time
	stats.LastUpdate = now
}
