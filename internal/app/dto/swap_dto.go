package dto

import (
	"time"

	"coinStatApp/internal/domain/model"
)

// SwapDTO represents a data transfer object for swap events
type SwapDTO struct {
	ID        string    `json:"id"`
	Token     string    `json:"token"`
	Amount    float64   `json:"amount"`
	USD       float64   `json:"usd"`
	Side      string    `json:"side"`
	Who       string    `json:"who"`
	Timestamp time.Time `json:"timestamp"`
}

// ToModel converts a SwapDTO to a domain model
func (dto *SwapDTO) ToModel() *model.Swap {
	return &model.Swap{
		ID:        dto.ID,
		Token:     dto.Token,
		Amount:    dto.Amount,
		USD:       dto.USD,
		Side:      dto.Side,
		Who:       dto.Who,
		Timestamp: dto.Timestamp,
	}
}

// FromModel creates a SwapDTO from a domain model
func FromModel(swap *model.Swap) *SwapDTO {
	return &SwapDTO{
		ID:        swap.ID,
		Token:     swap.Token,
		Amount:    swap.Amount,
		USD:       swap.USD,
		Side:      swap.Side,
		Who:       swap.Who,
		Timestamp: swap.Timestamp,
	}
}

func FromModels(swaps []*model.Swap) []*SwapDTO {
	dtos := make([]*SwapDTO, len(swaps))
	for i, swap := range swaps {
		dtos[i] = FromModel(swap)
	}
	return dtos
}

// StatisticsDTO represents a data transfer object for statistics
type StatisticsDTO struct {
	Token            string  `json:"token"`
	TotalVolume      float64 `json:"total_volume"`
	TotalUSDVolume   float64 `json:"total_usd_volume"`
	BuyVolume        float64 `json:"buy_volume"`
	SellVolume       float64 `json:"sell_volume"`
	BuyUSDVolume     float64 `json:"buy_usd_volume"`
	SellUSDVolume    float64 `json:"sell_usd_volume"`
	TransactionCount int     `json:"transaction_count"`
	LastUpdated      string  `json:"last_updated"`
}
