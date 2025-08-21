package useCases

import (
	"coinStatApp/internal/domain/model"
	"context"
)

// StatisticsService defines the interface for statistics calculation and querying.
type StatisticsService interface {
	ProcessSwap(ctx context.Context, swap *model.Swap) error
	GetStatistics(ctx context.Context, token string) (*model.Statistics, error)
	GetAllStatistics(ctx context.Context) ([]*model.Statistics, error)
}
