package scanner

import (
	"context"
	"fmt"
	"time"

	"covered-call-tracker/internal/provider"
)

// Candidate represents an option opportunity identified by the engine
type Candidate struct {
	Ticker          string
	StockPrice      float64
	StrikePrice     float64
	ExpirationDate  time.Time
	Premium         float64
	AnnualizedYield float64
}

type Engine struct {
	client provider.StockClient
}

func NewEngine(client provider.StockClient) *Engine {
	return &Engine{client: client}
}

// EvaluateOption calculates annualized return and checks if it meets the minimum yield threshold
func (e *Engine) EvaluateOption(quote provider.MarketQuote, minYield float64) (float64, bool) {
	if quote.StockPrice <= 0 || quote.Bid <= 0 {
		return 0, false
	}

	daysToExpiration := time.Until(quote.ExpirationDate).Hours() / 24
	if daysToExpiration <= 0 {
		return 0, false
	}

	rawYield := quote.Bid / quote.StockPrice
	annualizedYield := rawYield * (365.0 / daysToExpiration) * 100.0

	return annualizedYield, annualizedYield >= minYield
}

// ScanTicker fetches option quotes for a given symbol and returns matching Candidates
func (e *Engine) ScanTicker(ctx context.Context, ticker string, minYield float64) ([]Candidate, error) {
	// Fetch quotes from the stock provider
	quotes, err := e.client.GetOptionChain(ctx, ticker)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch option chain for %s: %w", ticker, err)
	}

	var candidates []Candidate
	for _, q := range quotes {
		yield, qualifies := e.EvaluateOption(q, minYield)
		if qualifies {
			candidates = append(candidates, Candidate{
				Ticker:          ticker,
				StockPrice:      q.StockPrice,
				StrikePrice:     q.StrikePrice,
				ExpirationDate:  q.ExpirationDate,
				Premium:         q.Bid, // Using bid price as premium
				AnnualizedYield: yield,
			})
		}
	}

	return candidates, nil
}
