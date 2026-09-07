CREATE TABLE IF NOT EXISTS underlying_stocks (
    id SERIAL PRIMARY KEY,
    ticker VARCHAR(10) NOT NULL UNIQUE,
    company_name VARCHAR(255) NOT NULL,
    shares_owned INT NOT NULL DEFAULT 100 CHECK (shares_owned >= 0),
    effective_cost_basis NUMERIC(10, 2) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_underlying_stocks_ticker ON underlying_stocks(ticker);