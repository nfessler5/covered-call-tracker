package marketdata

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

type MockClient struct{}

func NewMockClient() Client {
	return &MockClient{}
}

func (m *MockClient) GetStockQuote(ctx context.Context, ticker string) (*StockQuote, error) {
	// Generate a mock price around 30.00
	basePrice := 30.00 + (rand.Float64()*2.0 - 1.0)
	return &StockQuote{
		Ticker:       ticker,
		CurrentPrice: basePrice,
		Bid:          basePrice - 0.05,
		Ask:          basePrice + 0.05,
		UpdatedAt:    time.Now(),
	}, nil
}

func (m *MockClient) GetOptionQuote(ctx context.Context, ticker string, strike float64, expiration string) (*OptionQuote, error) {
	if ticker == "" || strike <= 0 {
		return nil, fmt.Errorf("invalid quote parameters")
	}

	bid := 1.20
	ask := 1.30
	return &OptionQuote{
		UnderlyingTicker: ticker,
		Strike:           strike,
		Expiration:       expiration,
		OptionType:       "CALL",
		Bid:              bid,
		Ask:              ask,
		MidPrice:         (bid + ask) / 2.0,
		Delta:            0.35,
		UpdatedAt:        time.Now(),
	}, nil
}