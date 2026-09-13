package main

import (
	"context"
	"database/sql"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"covered-call-tracker/internal/poller"
	"covered-call-tracker/internal/repository"
	"covered-call-tracker/internal/ui"
	"covered-call-tracker/pkg/marketdata"
)

type ScannerCandidate struct {
	Rank          int       `json:"rank"`
	Ticker        string    `json:"ticker"`
	StockPrice    float64   `json:"stock_price"`
	StrikePrice   float64   `json:"strike_price"`
	Premium       float64   `json:"premium"`
	Expiration    string    `json:"expiration"`
	DTE           int       `json:"dte"`
	MaxROI        float64   `json:"max_roi"`
	AnnualizedROI float64   `json:"annualized_roi"`
	AddedAt       time.Time `json:"added_at"`
}

var (
	scannerMu         sync.RWMutex
	scannerCandidates = []string{"NVDA", "AMD", "TSLA", "AAPL", "PLTR"}

	// Add persistent filter state variables
	activeDTE    = 7
	activeMinROI = 1.0
)

func main() {
	// 1. Initialize Postgres connection
	dbDSN := os.Getenv("DB_DSN")
	if dbDSN == "" {
		dbDSN = "postgres://tracker_user:tracker_password@localhost:5432/covered_call_tracker?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbDSN)
	if err != nil {
		log.Fatalf("Failed connecting to Postgres: %v", err)
	}
	defer db.Close()

	// 2. Initialize Repositories and Market Data Client
	repo := repository.NewPostgresRepository(db)
	mdClient := marketdata.NewMockClient()

	// 3. Instantiate and Start Background Poller (Polls every 30 seconds)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bgPoller := poller.NewPoller(repo, mdClient, 30*time.Second)
	bgPoller.Start(ctx)

	// UI handler
	uiHandler, err := ui.NewUIHandler(repo)
	if err != nil {
		log.Fatalf("Failed loading UI templates: %v", err)
	}

	// Route Handlers
	http.HandleFunc("/", uiHandler.RenderFullDashboard)
	http.HandleFunc("/ui/position/detail", uiHandler.RenderPositionDetails)
	http.HandleFunc("/ui/position/roll", uiHandler.RollPositionHandler)
	http.HandleFunc("/ui/position/status", uiHandler.UpdatePositionStatusHandler)
	http.HandleFunc("/ui/positions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			uiHandler.CreatePositionHandler(w, r)
			return
		}
		uiHandler.RenderPositionsFragment(w, r)
	})

	// Dynamic Option Scanner Route Handler (GET, POST, DELETE)
	http.HandleFunc("/ui/scanner", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			log.Printf("Error parsing form: %v", err)
		}

		scannerMu.Lock()
		// Update active state only if explicit form values were provided
		if val := r.FormValue("dte"); val != "" {
			if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
				activeDTE = parsed
			}
		}

		if val := r.FormValue("min_roi"); val != "" {
			if parsed, err := strconv.ParseFloat(val, 64); err == nil && parsed >= 0 {
				activeMinROI = parsed
			}
		}

		// Use stored active state
		currentDTE := activeDTE
		currentMinROI := activeMinROI
		scannerMu.Unlock()

		// 2. Handle POST (Add Ticker)
		if r.Method == http.MethodPost {
			ticker := strings.ToUpper(strings.TrimSpace(r.FormValue("ticker")))
			if ticker != "" {
				scannerMu.Lock()
				exists := false
				for _, t := range scannerCandidates {
					if t == ticker {
						exists = true
						break
					}
				}
				if !exists {
					scannerCandidates = append(scannerCandidates, ticker)
				}
				scannerMu.Unlock()
			}
		}

		// 3. Handle DELETE (Remove Ticker)
		if r.Method == http.MethodDelete {
			ticker := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("ticker")))
			if ticker != "" {
				scannerMu.Lock()
				updated := make([]string, 0, len(scannerCandidates))
				for _, t := range scannerCandidates {
					if t != ticker {
						updated = append(updated, t)
					}
				}
				scannerCandidates = updated
				scannerMu.Unlock()
			}
		}

		// 4. Calculate and sort candidates
		candidates := getSortedScannerCandidates(currentDTE, currentMinROI)

		viewData := struct {
			Candidates []ScannerCandidate
			DTE        int
			MinROI     float64
		}{
			Candidates: candidates,
			DTE:        currentDTE,
			MinROI:     currentMinROI,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := uiHandler.ExecuteTemplate(w, "scanner-table", viewData); err != nil {
			log.Printf("Template execution error for 'scanner-table': %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	go func() {
		log.Println("Starting HTMX UI Server on http://localhost:8080...")
		if err := http.ListenAndServe(":8080", nil); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// 4. Handle Graceful Shutdown (Listen for Ctrl+C or SIGTERM)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Println("Covered Call Tracker service running. Press Ctrl+C to exit.")
	<-sigChan

	log.Println("Shutting down service...")
	bgPoller.Stop()
}

// Generates simulated option chain data and computes ROI metrics using the requested DTE
func computeCandidateData(ticker string, dte int) ScannerCandidate {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))

	// Deterministic base seed from ticker characters for mock pricing
	var seed int
	for _, ch := range ticker {
		seed += int(ch)
	}

	stockPrice := 50.0 + float64(seed%250) + (float64(time.Now().Second()%10) * 0.15)
	strikePrice := math.Ceil(stockPrice * 1.03) // 3% OTM strike
	premium := math.Round((stockPrice*0.035)*100) / 100

	// ROI Calculations using requested DTE
	maxROI := (premium / stockPrice) * 100
	annualizedROI := maxROI * (365.0 / float64(dte))

	expDate := time.Now().AddDate(0, 0, dte).Format("2006-01-02")

	return ScannerCandidate{
		Ticker:        ticker,
		StockPrice:    stockPrice,
		StrikePrice:   strikePrice,
		Premium:       premium,
		Expiration:    expDate,
		DTE:           dte,
		MaxROI:        maxROI,
		AnnualizedROI: annualizedROI,
		AddedAt:       time.Now(),
	}
}

// Retrieves, filters by minROI, orders by highest ROI, and assigns Rank
func getSortedScannerCandidates(dte int, minROI float64) []ScannerCandidate {
	scannerMu.RLock()
	tickers := make([]string, len(scannerCandidates))
	copy(tickers, scannerCandidates)
	scannerMu.RUnlock()

	candidates := make([]ScannerCandidate, 0, len(tickers))
	for _, ticker := range tickers {
		cand := computeCandidateData(ticker, dte)
		// Filter candidates by minimum ROI threshold
		if cand.MaxROI >= minROI {
			candidates = append(candidates, cand)
		}
	}

	// Sort descending by Max ROI (Highest -> Lowest)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].MaxROI > candidates[j].MaxROI
	})

	// Assign 1-based rank after sorting
	for i := range candidates {
		candidates[i].Rank = i + 1
	}

	return candidates
}
