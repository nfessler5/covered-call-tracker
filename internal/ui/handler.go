package ui

import (
	"html/template"
	"net/http"
	"strconv"

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

	// Render ONLY the defined positions-table block
	h.tmpl.ExecuteTemplate(w, "positions-table", data)
}

// CreatePositionHandler handles HTMX POST submissions to create positions
func (h *UIHandler) CreatePositionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	strike, _ := strconv.ParseFloat(r.FormValue("strike_price"), 64)
	contracts, _ := strconv.Atoi(r.FormValue("contracts"))
	premium, _ := strconv.ParseFloat(r.FormValue("premium"), 64)
	underlyingID, _ := strconv.Atoi(r.FormValue("underlying_id"))
	expirationStr := r.FormValue("expiration") // Pass raw YYYY-MM-DD string

	pos := &repository.CoveredCallPosition{
		UnderlyingID:     underlyingID,
		StrikePrice:      strike,
		ExpirationDate:   expirationStr,
		ContractsCount:   contracts,
		PremiumCollected: premium * float64(contracts*100),
		Status:           "OPEN",
	}

	// Capture both returned position and error from CreatePosition
	_, err := h.repo.CreatePosition(r.Context(), pos)
	if err != nil {
		http.Error(w, "Failed to create position: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Re-render updated positions table fragment
	h.RenderPositionsFragment(w, r)
}