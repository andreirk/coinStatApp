package model

import "time"

type SwapEvent struct {
	ID        string // unique event id for deduplication
	Timestamp int64  // unix timestamp
	Who       string
	Token     string
	Amount    float64
	USD       float64
	Side      string // buy/sell
}

// Swap represents a swap event in the domain.
type Swap struct {
	ID        string
	Who       string
	Token     string
	Amount    float64
	USD       float64
	Side      string
	Timestamp time.Time
}

// Statistics holds aggregated statistics for a token.
type Statistics struct {
	Token      string
	Volume5Min float64
	Volume1H   float64
	Volume24H  float64
	Count5Min  int
	Count1H    int
	Count24H   int
	LastUpdate time.Time
}
