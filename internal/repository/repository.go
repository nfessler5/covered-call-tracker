package repository

import (
	"context"
	"database/sql"
)

// Domain Models

type User struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

type CoveredCallPosition struct {
	ID               int64   `json:"id"`
	UnderlyingID     int     `json:"underlying_id"`
	Ticker           string  `json:"ticker"`
	SharesCostBasis  float64 `json:"shares_cost_basis"` // Purchase price per share
	StrikePrice      float64 `json:"strike_price"`
	ExpirationDate   string  `json:"expiration_date"`
	ContractsCount   int     `json:"contracts_count"`
	PremiumCollected float64 `json:"premium_collected"`
	Status           string  `json:"status"`
}

// EffectiveCostBasis calculates break-even price per share after option premium
func (p *CoveredCallPosition) EffectiveCostBasis() float64 {
	if p.ContractsCount == 0 {
		return p.SharesCostBasis
	}
	premiumPerShare := p.PremiumCollected / float64(p.ContractsCount*100)
	return p.SharesCostBasis - premiumPerShare
}

// MaxROI calculates potential return on investment if shares are assigned at strike
func (p *CoveredCallPosition) MaxROI() float64 {
	if p.SharesCostBasis <= 0 {
		return 0.0
	}
	premiumPerShare := p.PremiumCollected / float64(p.ContractsCount*100)
	capitalGainPerShare := p.StrikePrice - p.SharesCostBasis
	totalProfitPerShare := capitalGainPerShare + premiumPerShare
	return (totalProfitPerShare / p.SharesCostBasis) * 100
}

// Repository Interfaces

type UserRepository interface {
	GetUserByID(ctx context.Context, id int64) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
}

type PositionRepository interface {
	GetPositionsByStatus(ctx context.Context, status string) ([]*CoveredCallPosition, error)
	GetPositionByID(ctx context.Context, id int64) (*CoveredCallPosition, error)
	CreatePosition(ctx context.Context, pos *CoveredCallPosition) (*CoveredCallPosition, error)
	GetOrCreateUnderlyingAsset(ctx context.Context, ticker string) (int, error)
	UpdatePositionStatus(ctx context.Context, id int64, status string) error
	RollPosition(ctx context.Context, oldID int64, newStrike float64, newExpiration string, netCredit float64) error
}

// Repository combines PositionRepository and UserRepository for access across services
type Repository interface {
	PositionRepository
	UserRepository
}

// Postgres Implementation

type postgresRepo struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) Repository {
	return &postgresRepo{db: db}
}

// UserRepository Implementation

func (r *postgresRepo) GetUserByID(ctx context.Context, id int64) (*User, error) {
	query := `SELECT id, email, username FROM users WHERE id = $1`
	var u User
	err := r.db.QueryRowContext(ctx, query, id).Scan(&u.ID, &u.Email, &u.Username)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *postgresRepo) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `SELECT id, email, username FROM users WHERE email = $1`
	var u User
	err := r.db.QueryRowContext(ctx, query, email).Scan(&u.ID, &u.Email, &u.Username)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// PositionRepository Implementation

func (r *postgresRepo) GetOrCreateUnderlyingAsset(ctx context.Context, ticker string) (int, error) {
	var id int
	err := r.db.QueryRowContext(ctx, "SELECT id FROM underlying_assets WHERE ticker = $1", ticker).Scan(&id)
	if err == nil {
		return id, nil
	}

	err = r.db.QueryRowContext(ctx, "INSERT INTO underlying_assets (ticker) VALUES ($1) RETURNING id", ticker).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *postgresRepo) GetPositionsByStatus(ctx context.Context, status string) ([]*CoveredCallPosition, error) {
	query := `
        SELECT 
            p.id, p.underlying_id, COALESCE(u.ticker, 'UNKNOWN') AS ticker,
            COALESCE(p.shares_cost_basis, 0.00) AS shares_cost_basis,
            p.strike_price, p.expiration_date, p.contracts_count, 
            p.premium_collected, p.status
        FROM covered_call_positions p
        LEFT JOIN underlying_assets u ON p.underlying_id = u.id
        WHERE p.status = $1
        ORDER BY p.id DESC
    `
	rows, err := r.db.QueryContext(ctx, query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []*CoveredCallPosition
	for rows.Next() {
		var p CoveredCallPosition
		if err := rows.Scan(
			&p.ID, &p.UnderlyingID, &p.Ticker, &p.SharesCostBasis, &p.StrikePrice,
			&p.ExpirationDate, &p.ContractsCount, &p.PremiumCollected, &p.Status,
		); err != nil {
			return nil, err
		}
		positions = append(positions, &p)
	}
	return positions, nil
}

func (r *postgresRepo) GetPositionByID(ctx context.Context, id int64) (*CoveredCallPosition, error) {
	query := `
        SELECT 
            p.id, p.underlying_id, COALESCE(u.ticker, 'UNKNOWN') AS ticker,
            COALESCE(p.shares_cost_basis, 0.00) AS shares_cost_basis,
            p.strike_price, p.expiration_date, p.contracts_count, 
            p.premium_collected, p.status
        FROM covered_call_positions p
        LEFT JOIN underlying_assets u ON p.underlying_id = u.id
        WHERE p.id = $1
    `
	var p CoveredCallPosition
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.UnderlyingID, &p.Ticker, &p.SharesCostBasis, &p.StrikePrice,
		&p.ExpirationDate, &p.ContractsCount, &p.PremiumCollected, &p.Status,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *postgresRepo) CreatePosition(ctx context.Context, pos *CoveredCallPosition) (*CoveredCallPosition, error) {
	query := `
        INSERT INTO covered_call_positions 
            (underlying_id, shares_cost_basis, strike_price, expiration_date, contracts_count, premium_collected, status)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id
    `
	err := r.db.QueryRowContext(ctx, query,
		pos.UnderlyingID, pos.SharesCostBasis, pos.StrikePrice, pos.ExpirationDate,
		pos.ContractsCount, pos.PremiumCollected, pos.Status,
	).Scan(&pos.ID)

	if err != nil {
		return nil, err
	}
	return pos, nil
}

func (r *postgresRepo) UpdatePositionStatus(ctx context.Context, id int64, status string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE covered_call_positions SET status = $1 WHERE id = $2", status, id)
	return err
}

func (r *postgresRepo) RollPosition(ctx context.Context, oldID int64, newStrike float64, newExpiration string, netCredit float64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Fetch current position details
	var current CoveredCallPosition
	query := `SELECT underlying_id, shares_cost_basis, contracts_count FROM covered_call_positions WHERE id = $1`
	if err := tx.QueryRowContext(ctx, query, oldID).Scan(&current.UnderlyingID, &current.SharesCostBasis, &current.ContractsCount); err != nil {
		return err
	}

	// 2. Mark current position as ROLLED
	if _, err := tx.ExecContext(ctx, "UPDATE covered_call_positions SET status = 'ROLLED' WHERE id = $1", oldID); err != nil {
		return err
	}

	// 3. Insert new rolled position
	insertQuery := `
        INSERT INTO covered_call_positions 
        (underlying_id, shares_cost_basis, strike_price, expiration_date, contracts_count, premium_collected, status)
        VALUES ($1, $2, $3, $4, $5, $6, 'OPEN')
    `
	if _, err := tx.ExecContext(ctx, insertQuery, current.UnderlyingID, current.SharesCostBasis, newStrike, newExpiration, current.ContractsCount, netCredit); err != nil {
		return err
	}

	return tx.Commit()
}
