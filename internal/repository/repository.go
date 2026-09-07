package repository

import (
	"context"
	"database/sql"
)

// Domain models
type UnderlyingStock struct {
	ID                 int
	Ticker             string
	CompanyName        string
	SharesOwned        int
	EffectiveCostBasis float64
}

type CoveredCallPosition struct {
	ID               int
	UnderlyingID     int
	StrikePrice      float64
	ExpirationDate   string
	PremiumCollected float64
	ContractsCount   int
	Status           string
}

// Repository Interface
type Repository interface {
	GetUnderlyingStocks(ctx context.Context) ([]*UnderlyingStock, error)
	GetPositionsByStatus(ctx context.Context, status string) ([]*CoveredCallPosition, error)
	CreatePosition(ctx context.Context, pos *CoveredCallPosition) (*CoveredCallPosition, error)
}

// Postgres Repository Struct
type postgresRepo struct {
	db *sql.DB
}

// Constructor function
func NewPostgresRepository(db *sql.DB) Repository {
	return &postgresRepo{db: db}
}

// GetUnderlyingStocks fetches all stocks from the database
func (r *postgresRepo) GetUnderlyingStocks(ctx context.Context) ([]*UnderlyingStock, error) {
	query := `SELECT id, ticker, company_name, shares_owned, effective_cost_basis FROM underlying_stocks`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stocks []*UnderlyingStock
	for rows.Next() {
		s := &UnderlyingStock{}
		if err := rows.Scan(&s.ID, &s.Ticker, &s.CompanyName, &s.SharesOwned, &s.EffectiveCostBasis); err != nil {
			return nil, err
		}
		stocks = append(stocks, s)
	}
	return stocks, nil
}

// GetPositionsByStatus fetches positions filtered by status (e.g. OPEN, EXPIRED)
func (r *postgresRepo) GetPositionsByStatus(ctx context.Context, status string) ([]*CoveredCallPosition, error) {
	query := `SELECT id, underlying_id, strike_price, expiration_date, premium_collected, contracts_count, status 
	          FROM covered_call_positions WHERE status = $1`
	rows, err := r.db.QueryContext(ctx, query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []*CoveredCallPosition
	for rows.Next() {
		p := &CoveredCallPosition{}
		if err := rows.Scan(&p.ID, &p.UnderlyingID, &p.StrikePrice, &p.ExpirationDate, &p.PremiumCollected, &p.ContractsCount, &p.Status); err != nil {
			return nil, err
		}
		positions = append(positions, p)
	}
	return positions, nil
}

// CreatePosition inserts a new covered call position
func (r *postgresRepo) CreatePosition(ctx context.Context, pos *CoveredCallPosition) (*CoveredCallPosition, error) {
	query := `INSERT INTO covered_call_positions (underlying_id, strike_price, expiration_date, premium_collected, contracts_count, status)
	          VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
	err := r.db.QueryRowContext(ctx, query, pos.UnderlyingID, pos.StrikePrice, pos.ExpirationDate, pos.PremiumCollected, pos.ContractsCount, pos.Status).Scan(&pos.ID)
	if err != nil {
		return nil, err
	}
	return pos, nil
}