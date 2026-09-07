package marketdata

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type IBKRClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewIBKRClient(baseURL string) Client {
	return &IBKRClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (ib *IBKRClient) GetStockQuote(ctx context.Context, ticker string) (*StockQuote, error) {
	// TODO: Perform HTTP GET to IBKR Web API / Client Portal endpoint
	return nil, fmt.Errorf("IBKR integration not implemented yet")
}

func (ib *IBKRClient) GetOptionQuote(ctx context.Context, ticker string, strike float64, expiration string) (*OptionQuote, error) {
	// TODO: Query IBKR option chain endpoint
	return nil, fmt.Errorf("IBKR integration not implemented yet")
}