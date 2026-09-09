package provider

import (
	"context"
	"time"
)

type MarketQuote struct {
	Ticker         string
	StockPrice     float64
	StrikePrice    float64
	ExpirationDate time.Time
	OptionType     string // "CALL" or "PUT"
	Bid            float64
	Ask            float64
	ImpliedVol     float64
	Volume         int64
	OpenInterest   int64
}

// StockClient isolates third-party API concerns (auth, rate limiting, parsing)
type StockClient interface {
	GetMarketQuote(ctx context.Context, ticker string) (*MarketQuote, error)
	GetOptionChain(ctx context.Context, ticker string) ([]MarketQuote, error)
}
