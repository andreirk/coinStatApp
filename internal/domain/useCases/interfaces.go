package useCases

import (
	"coinStatApp/internal/domain/model"
	"context"
	"net/http"
)

// StatisticsService defines the interface for statistics calculation and querying.
type StatisticsService interface {
	ProcessSwap(ctx context.Context, swap *model.Swap) error
	GetStatistics(ctx context.Context, token string) (*model.Statistics, error)
	GetAllStatistics(ctx context.Context) ([]*model.Statistics, error)
}

// Broadcaster defines an interface for pushing updates to WebSocket/API layers.
type Broadcaster interface {
	BroadcastStatistics(stats *model.Statistics)
	Handler() func(http.ResponseWriter, *http.Request)
}
