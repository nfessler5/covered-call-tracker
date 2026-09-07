CREATE TYPE position_status AS ENUM ('OPEN', 'EXPIRED', 'ASSIGNED', 'ROLLED', 'CLOSED');

CREATE TABLE IF NOT EXISTS covered_call_positions (
    id SERIAL PRIMARY KEY,
    underlying_id INT NOT NULL REFERENCES underlying_stocks(id) ON DELETE CASCADE,
    strike_price NUMERIC(10, 2) NOT NULL,
    expiration_date DATE NOT NULL,
    premium_collected NUMERIC(10, 2) NOT NULL,
    contracts_count INT NOT NULL DEFAULT 1 CHECK (contracts_count > 0),
    status position_status NOT NULL DEFAULT 'OPEN',
    opened_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    closed_at TIMESTAMP WITH TIME ZONE,
    realized_pnl NUMERIC(10, 2) DEFAULT 0.00,
    notes TEXT
);

CREATE INDEX idx_positions_underlying_id ON covered_call_positions(underlying_id);
CREATE INDEX idx_positions_status ON covered_call_positions(status);