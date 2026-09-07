package marketdata

import (
	"context"
	"time"
)

// StockQuote contains current pricing metrics for an underlying stock.
type StockQuote struct {
	Ticker       string    `json:"ticker"`
	CurrentPrice float64   `json:"current_price"`
	Bid          float64   `json:"bid"`
	Ask          float64   `json:"ask"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// OptionQuote contains pricing, greeks, and contract details for a covered call option.
type OptionQuote struct {
	UnderlyingTicker string    `json:"underlying_ticker"`
	Strike           float64   `json:"strike"`
	Expiration       string    `json:"expiration"` // YYYY-MM-DD
	OptionType       string    `json:"option_type"` // "CALL" or "PUT"
	Bid              float64   `json:"bid"`
	Ask              float64   `json:"ask"`
	MidPrice         float64   `json:"mid_price"`
	ImpliedVol       float64   `json:"implied_volatility,omitempty"`
	Delta            float64   `json:"delta,omitempty"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Client defines the contract for fetching real-time market data from any external source.
type Client interface {
	GetStockQuote(ctx context.Context, ticker string) (*StockQuote, error)
	GetOptionQuote(ctx context.Context, ticker string, strike float64, expiration string) (*OptionQuote, error)
}