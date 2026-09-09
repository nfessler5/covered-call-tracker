package scanner

import (
	"time"

	"github.com/nfessler5/covered-call-tracker/internal/provider"
)

type Engine struct {
	client provider.StockClient
}

func NewEngine(client provider.StockClient) *Engine {
	return &Engine{client: client}
}

func (e *Engine) EvaluateOption(quote provider.MarketQuote) (float64, bool) {
	if quote.StockPrice <= 0 || quote.Bid <= 0 {
		return 0, false
	}

	// Yield Calculation: (Premium / Stock Price) * (365 / Days To Expiration) * 100
	daysToExpiration := time.Until(quote.ExpirationDate).Hours() / 24
	if daysToExpiration <= 0 {
		return 0, false
	}

	rawYield := quote.Bid / quote.StockPrice
	annualizedYield := rawYield * (365.0 / daysToExpiration) * 100.0

	return annualizedYield, annualizedYield >= 15.0 // Filter for >= 15% annualized yield
}
