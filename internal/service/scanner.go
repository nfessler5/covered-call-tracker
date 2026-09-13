package service

import (
	"context"
	"fmt"

	gen "covered-call-tracker/gen/scanner"
	"covered-call-tracker/internal/repository"
	scannerEngine "covered-call-tracker/internal/scanner"
)

// ScannerService implements gen/scanner.Service
type ScannerService struct {
	engine *scannerEngine.Engine
	repo   repository.ScannerRepository
}

// NewScannerService initializes a new ScannerService instance
func NewScannerService(engine *scannerEngine.Engine, repo repository.ScannerRepository) *ScannerService {
	return &ScannerService{
		engine: engine,
		repo:   repo,
	}
}

// Scan handles the "POST /api/v1/scanner/run" endpoint defined in Goa
func (s *ScannerService) Scan(ctx context.Context, p *gen.ScanPayload) ([]*gen.Candidate, error) {
	minYield := 15.0
	if p.MinAnnualizedYield != nil {
		minYield = *p.MinAnnualizedYield
	}

	var allCandidates []scannerEngine.Candidate

	// Run scanner for each ticker provided in the payload
	for _, ticker := range p.Tickers {
		found, err := s.engine.ScanTicker(ctx, ticker, minYield)
		if err != nil {
			// Log error and continue to remaining tickers
			continue
		}
		allCandidates = append(allCandidates, found...)
	}

	// Persist matching candidates to PostgreSQL
	if len(allCandidates) > 0 {
		if err := s.repo.SaveCandidates(ctx, allCandidates); err != nil {
			return nil, fmt.Errorf("failed to save scanner candidates: %w", err)
		}
	}

	// Map engine domain models to GOA generated result types
	var result []*gen.Candidate
	for _, c := range allCandidates {
		result = append(result, &gen.Candidate{
			Ticker:          c.Ticker,
			StockPrice:      c.StockPrice,
			StrikePrice:     c.StrikePrice,
			ExpirationDate:  c.ExpirationDate.Format("2006-01-02"),
			Premium:         c.Premium,
			AnnualizedYield: c.AnnualizedYield,
		})
	}

	return result, nil
}
