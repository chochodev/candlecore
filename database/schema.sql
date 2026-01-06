-- Candlecore PostgreSQL Database Schema
-- This schema stores trading engine state for restart-safe operation

-- Create database (run this manually first)
-- CREATE DATABASE candlecore;

-- Account state table
CREATE TABLE IF NOT EXISTS accounts (
    id SERIAL PRIMARY KEY,
    balance DECIMAL(20, 8) NOT NULL,
    equity DECIMAL(20, 8) NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Positions table
CREATE TABLE IF NOT EXISTS positions (
    id SERIAL PRIMARY KEY,
    account_id INTEGER NOT NULL,
    symbol VARCHAR(50) NOT NULL,
    side VARCHAR(10) NOT NULL, -- 'buy' or 'sell'
    entry_price DECIMAL(20, 8) NOT NULL,
    quantity DECIMAL(20, 8) NOT NULL,
    current_price DECIMAL(20, 8) NOT NULL,
    unrealized_pnl DECIMAL(20, 8) NOT NULL,
    opened_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE,
    UNIQUE(account_id, symbol)
);

-- Orders table (for open orders)
CREATE TABLE IF NOT EXISTS orders (
    id VARCHAR(100) PRIMARY KEY,
    account_id INTEGER NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    side VARCHAR(10) NOT NULL,
    type VARCHAR(20) NOT NULL,
    symbol VARCHAR(50) NOT NULL,
    quantity DECIMAL(20, 8) NOT NULL,
    price DECIMAL(20, 8) NOT NULL,
    status VARCHAR(20) NOT NULL,
    filled_price DECIMAL(20, 8),
    filled_qty DECIMAL(20, 8),
    fee DECIMAL(20, 8),
    slippage DECIMAL(20, 8),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE
);

-- Trade history table
CREATE TABLE IF NOT EXISTS trades (
    id VARCHAR(100) PRIMARY KEY,
    account_id INTEGER NOT NULL,
    symbol VARCHAR(50) NOT NULL,
    side VARCHAR(10) NOT NULL,
    entry_price DECIMAL(20, 8) NOT NULL,
    exit_price DECIMAL(20, 8) NOT NULL,
    quantity DECIMAL(20, 8) NOT NULL,
    pnl DECIMAL(20, 8) NOT NULL,
    fee DECIMAL(20, 8) NOT NULL,
    net_pnl DECIMAL(20, 8) NOT NULL,
    opened_at TIMESTAMP WITH TIME ZONE NOT NULL,
    closed_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE
);

-- Indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_positions_account ON positions(account_id);
CREATE INDEX IF NOT EXISTS idx_positions_symbol ON positions(symbol);
CREATE INDEX IF NOT EXISTS idx_orders_account ON orders(account_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_trades_account ON trades(account_id);
CREATE INDEX IF NOT EXISTS idx_trades_symbol ON trades(symbol);
CREATE INDEX IF NOT EXISTS idx_trades_closed_at ON trades(closed_at);

-- Insert initial account (run once)
-- This will create account with ID 1
INSERT INTO accounts (balance, equity, updated_at, created_at)
VALUES (10000.0, 10000.0, NOW(), NOW())
ON CONFLICT DO NOTHING;

-- View for account summary
CREATE OR REPLACE VIEW account_summary AS
SELECT 
    a.id,
    a.balance,
    a.equity,
    COUNT(DISTINCT p.id) as open_positions,
    COUNT(DISTINCT o.id) as open_orders,
    COUNT(DISTINCT t.id) as total_trades,
    COALESCE(SUM(t.net_pnl), 0) as total_pnl,
    a.updated_at
FROM accounts a
LEFT JOIN positions p ON a.id = p.account_id
LEFT JOIN orders o ON a.id = o.account_id AND o.status = 'pending'
LEFT JOIN trades t ON a.id = t.account_id
GROUP BY a.id, a.balance, a.equity, a.updated_at;

-- View for position summary
CREATE OR REPLACE VIEW position_summary AS
SELECT 
    p.id,
    p.symbol,
    p.side,
    p.entry_price,
    p.current_price,
    p.quantity,
    p.unrealized_pnl,
    ROUND(((p.current_price - p.entry_price) / p.entry_price * 100), 2) as pnl_percentage,
    p.opened_at,
    EXTRACT(EPOCH FROM (NOW() - p.opened_at))/3600 as hours_open
FROM positions p;

-- Function to update account equity based on positions
CREATE OR REPLACE FUNCTION update_account_equity()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE accounts
    SET equity = balance + COALESCE((
        SELECT SUM(unrealized_pnl)
        FROM positions
        WHERE account_id = NEW.account_id
    ), 0),
    updated_at = NOW()
    WHERE id = NEW.account_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-update equity when positions change
DROP TRIGGER IF EXISTS trigger_update_equity ON positions;
CREATE TRIGGER trigger_update_equity
    AFTER INSERT OR UPDATE OR DELETE ON positions
    FOR EACH ROW
    EXECUTE FUNCTION update_account_equity();

COMMENT ON TABLE accounts IS 'Stores account balance and equity';
COMMENT ON TABLE positions IS 'Stores open positions with unrealized P&L';
COMMENT ON TABLE orders IS 'Stores pending and historical orders';
COMMENT ON TABLE trades IS 'Stores completed trade history';
