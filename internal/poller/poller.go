package poller

import (
	"context"
	"log"
	"time"

	"covered-call-tracker/internal/repository"
	"covered-call-tracker/pkg/marketdata"
)

type Poller struct {
	repo       repository.Repository
	mdClient   marketdata.Client
	interval   time.Duration
	stopSignal chan struct{}
}

func NewPoller(repo repository.Repository, mdClient marketdata.Client, interval time.Duration) *Poller {
	return &Poller{
		repo:       repo,
		mdClient:   mdClient,
		interval:   interval,
		stopSignal: make(chan struct{}),
	}
}

// Start launches the poller background loop in a non-blocking goroutine
func (p *Poller) Start(ctx context.Context) {
	ticker := time.NewTicker(p.interval)

	go func() {
		log.Printf("[Poller] Background poller started (interval: %s)", p.interval)
		for {
			select {
			case <-ticker.C:
				if err := p.pollPositions(ctx); err != nil {
					log.Printf("[Poller Error] Failed polling cycle: %v", err)
				}
			case <-p.stopSignal:
				ticker.Stop()
				log.Println("[Poller] Poller background worker stopped.")
				return
			case <-ctx.Done():
				ticker.Stop()
				log.Println("[Poller] Context cancelled, shutting down poller.")
				return
			}
		}
	}()
}

// Stop gracefully terminates the background loop
func (p *Poller) Stop() {
	close(p.stopSignal)
}

// pollPositions executes a single polling cycle
func (p *Poller) pollPositions(ctx context.Context) error {
	// 1. Fetch open covered call positions from database
	openPositions, err := p.repo.GetPositionsByStatus(ctx, "OPEN")
	if err != nil {
		return err
	}

	if len(openPositions) == 0 {
		log.Println("[Poller] No open positions to update.")
		return nil
	}

	log.Printf("[Poller] Checking market data for %d open position(s)...", len(openPositions))

	for _, pos := range openPositions {
		// 2. Fetch market quote via our marketdata client interface
		// (In a production setup, we look up the ticker from the underlying_id mapping)
		quote, err := p.mdClient.GetOptionQuote(ctx, "SOXL", pos.StrikePrice, pos.ExpirationDate)
		if err != nil {
			log.Printf("[Poller Error] Position ID %d quote fetch failed: %v", pos.ID, err)
			continue
		}

		// 3. Compute option profit metrics
		// Max profit = Initial premium collected.
		// Current contract value to buy back = Ask price * 100 * contracts.
		currentBuybackCost := quote.Ask * 100.0 * float64(pos.ContractsCount)
		unrealizedProfit := pos.PremiumCollected - currentBuybackCost
		profitPercentage := (unrealizedProfit / pos.PremiumCollected) * 100.0

		log.Printf("[Position %d - SOXL $%.2f Call] Premium: $%.2f | Buyback: $%.2f | PnL: $%.2f (%.1f%%)",
			pos.ID, pos.StrikePrice, pos.PremiumCollected, currentBuybackCost, unrealizedProfit, profitPercentage)

		// 4. Alert on key covered call thresholds (e.g., 50% max profit reached)
		if profitPercentage >= 50.0 {
			log.Printf("🔥 [ALERT] Position ID %d reached 50%% max profit target! Consider closing/rolling.", pos.ID)
		}
	}

	return nil
}