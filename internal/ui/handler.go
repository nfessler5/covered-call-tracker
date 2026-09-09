package ui

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"covered-call-tracker/internal/repository"
)

type UIHandler struct {
	repo repository.Repository
	tmpl *template.Template
}

type DashboardData struct {
	Positions    []*repository.CoveredCallPosition
	TotalPremium float64
}

func NewUIHandler(repo repository.Repository) (*UIHandler, error) {
	tmpl, err := template.ParseFiles("templates/layout.html", "templates/dashboard.html")
	if err != nil {
		return nil, err
	}
	return &UIHandler{repo: repo, tmpl: tmpl}, nil
}

// RenderFullDashboard renders the layout + content on initial visit
func (h *UIHandler) RenderFullDashboard(w http.ResponseWriter, r *http.Request) {
	positions, err := h.repo.GetPositionsByStatus(r.Context(), "OPEN")
	if err != nil {
		http.Error(w, "Failed to load positions", http.StatusInternalServerError)
		return
	}

	var total float64
	for _, p := range positions {
		total += p.PremiumCollected
	}

	data := DashboardData{
		Positions:    positions,
		TotalPremium: total,
	}

	h.tmpl.ExecuteTemplate(w, "layout.html", data)
}

// RenderPositionsFragment returns only the updated <table> HTML fragment for HTMX polling
func (h *UIHandler) RenderPositionsFragment(w http.ResponseWriter, r *http.Request) {
	positions, err := h.repo.GetPositionsByStatus(r.Context(), "OPEN")
	if err != nil {
		log.Printf("[UI ERROR] GetPositionsByStatus failed: %v", err)
		http.Error(w, "Failed to load positions", http.StatusInternalServerError)
		return
	}

	var total float64
	for _, p := range positions {
		total += p.PremiumCollected
	}

	data := DashboardData{
		Positions:    positions,
		TotalPremium: total,
	}

	if err := h.tmpl.ExecuteTemplate(w, "positions-table", data); err != nil {
		log.Printf("[UI ERROR] Template rendering failed: %v", err)
	}
}

// CreatePositionHandler handles HTMX POST submissions to create positions
func (h *UIHandler) CreatePositionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ticker := strings.ToUpper(strings.TrimSpace(r.FormValue("ticker")))
	costBasis, _ := strconv.ParseFloat(r.FormValue("shares_cost_basis"), 64)
	strike, _ := strconv.ParseFloat(r.FormValue("strike_price"), 64)
	contracts, _ := strconv.Atoi(r.FormValue("contracts"))
	premium, _ := strconv.ParseFloat(r.FormValue("premium"), 64)
	expirationStr := r.FormValue("expiration")

	if ticker == "" {
		http.Error(w, "Ticker symbol is required", http.StatusBadRequest)
		return
	}

	underlyingID, err := h.repo.GetOrCreateUnderlyingAsset(r.Context(), ticker)
	if err != nil {
		log.Printf("[UI ERROR] Failed to resolve ticker %s: %v", ticker, err)
		http.Error(w, "Failed to resolve ticker: "+err.Error(), http.StatusInternalServerError)
		return
	}

	pos := &repository.CoveredCallPosition{
		UnderlyingID:     underlyingID,
		Ticker:           ticker,
		SharesCostBasis:  costBasis,
		StrikePrice:      strike,
		ExpirationDate:   expirationStr,
		ContractsCount:   contracts,
		PremiumCollected: premium * float64(contracts*100),
		Status:           "OPEN",
	}

	_, err = h.repo.CreatePosition(r.Context(), pos)
	if err != nil {
		log.Printf("[UI ERROR] Failed to create position: %v", err)
		http.Error(w, "Failed to create position: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.RenderPositionsFragment(w, r)
}

// RenderPositionDetails renders the slide-over drawer content
func (h *UIHandler) RenderPositionDetails(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid position ID", http.StatusBadRequest)
		return
	}

	pos, err := h.repo.GetPositionByID(r.Context(), id)
	if err != nil {
		log.Printf("[UI ERROR] GetPositionByID failed: %v", err)
		http.Error(w, "Position not found", http.StatusNotFound)
		return
	}

	h.tmpl.ExecuteTemplate(w, "position-details-drawer", pos)
}

// UpdatePositionStatusHandler marks position as EXPIRED, ASSIGNED, or CLOSED
func (h *UIHandler) UpdatePositionStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.FormValue("id")
	status := r.FormValue("status")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid position ID", http.StatusBadRequest)
		return
	}

	if err := h.repo.UpdatePositionStatus(r.Context(), id, status); err != nil {
		log.Printf("[UI ERROR] UpdatePositionStatus failed: %v", err)
		http.Error(w, "Failed to update status", http.StatusInternalServerError)
		return
	}

	// Re-render table fragment after update
	h.RenderPositionsFragment(w, r)
}

// RollPositionHandler processes the position roll transaction
func (h *UIHandler) RollPositionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.FormValue("id")
	newStrikeStr := r.FormValue("new_strike")
	newExpiration := r.FormValue("new_expiration")
	netCreditStr := r.FormValue("net_credit")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid position ID", http.StatusBadRequest)
		return
	}

	newStrike, _ := strconv.ParseFloat(newStrikeStr, 64)
	netCredit, _ := strconv.ParseFloat(netCreditStr, 64)

	if err := h.repo.RollPosition(r.Context(), id, newStrike, newExpiration, netCredit); err != nil {
		log.Printf("[UI ERROR] RollPosition failed: %v", err)
		http.Error(w, "Failed to roll position", http.StatusInternalServerError)
		return
	}

	// Refresh positions table
	h.RenderPositionsFragment(w, r)
}
