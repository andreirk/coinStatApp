package utils

import (
	"time"

	"github.com/google/uuid"

	"coinStatApp/internal/domain/model"
)

// SwapGenerator provides methods to generate test swap data
type SwapGenerator struct{}

// NewSwapGenerator creates a new swap generator
func NewSwapGenerator() *SwapGenerator {
	return &SwapGenerator{}
}

// GenerateSwaps creates a specified number of test swap events
func (g *SwapGenerator) GenerateSwaps(count int) []*model.Swap {
	tokens := []string{"ETH", "BTC", "XRP", "ADA", "SOL", "DOT", "LTC", "LINK", "BCH", "XLM"}
	sides := []string{"buy", "sell"}

	swaps := make([]*model.Swap, count)
	for i := 0; i < count; i++ {
		swaps[i] = &model.Swap{
			ID:        uuid.New().String(),
			Token:     tokens[i%len(tokens)],
			Amount:    float64(1 + i%10),
			USD:       float64(1000 + i*100),
			Side:      sides[i%2],
			Who:       "user" + string(rune('0'+i%10)),
			Timestamp: time.Now(),
		}
	}

	return swaps
}

// GenerateRandomSwap creates a single random swap event
func (g *SwapGenerator) GenerateRandomSwap(count int) []*model.Swap {
	tokens := []string{"ETH", "BTC", "XRP", "ADA", "SOL", "DOT", "LTC", "LINK", "BCH", "XLM"}
	sides := []string{"buy", "sell"}
	swaps := make([]*model.Swap, count)

	for i := 0; i < count; i++ {
		randInt := time.Now().Nanosecond()
		swaps[i] = &model.Swap{
			ID:        uuid.New().String(),
			Token:     tokens[i%len(tokens)],
			Amount:    float64(1 + i%10),
			USD:       float64(1000 + randInt%5000),
			Side:      sides[i%2],
			Who:       "user" + string(rune('0'+i%10)),
			Timestamp: time.Now(),
		}
	}
	return swaps
}
