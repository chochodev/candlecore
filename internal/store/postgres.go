package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"candlecore/internal/engine"

	_ "github.com/lib/pq"
)

// PostgresStore implements StateStore using PostgreSQL
// This provides robust, queryable state persistence
type PostgresStore struct {
	db        *sql.DB
	accountID int64
}

// NewPostgresStore creates a new PostgreSQL-based state store
func NewPostgresStore(connectionString string, accountID int64) (*PostgresStore, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &PostgresStore{
		db:        db,
		accountID: accountID,
	}, nil
}

// Initialize sets up the database schema if it doesn't exist
func (s *PostgresStore) Initialize(ctx context.Context) error {
	schema := `
		-- Accounts table
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
			side VARCHAR(10) NOT NULL,
			entry_price DECIMAL(20, 8) NOT NULL,
			quantity DECIMAL(20, 8) NOT NULL,
			current_price DECIMAL(20, 8) NOT NULL,
			unrealized_pnl DECIMAL(20, 8) NOT NULL,
			opened_at TIMESTAMP WITH TIME ZONE NOT NULL,
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE,
			UNIQUE(account_id, symbol)
		);

		-- Orders table
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

		-- Trades table
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

		-- Indexes
		CREATE INDEX IF NOT EXISTS idx_positions_account ON positions(account_id);
		CREATE INDEX IF NOT EXISTS idx_positions_symbol ON positions(symbol);
		CREATE INDEX IF NOT EXISTS idx_orders_account ON orders(account_id);
		CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
		CREATE INDEX IF NOT EXISTS idx_trades_account ON trades(account_id);
		CREATE INDEX IF NOT EXISTS idx_trades_symbol ON trades(symbol);
		CREATE INDEX IF NOT EXISTS idx_trades_closed_at ON trades(closed_at);

		-- Views
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
	`

	// Execute schema creation
	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Create trigger function for equity updates
	triggerFunc := `
		CREATE OR REPLACE FUNCTION update_account_equity()
		RETURNS TRIGGER AS $$
		BEGIN
			UPDATE accounts
			SET equity = balance + COALESCE((
				SELECT SUM(unrealized_pnl)
				FROM positions
				WHERE account_id = COALESCE(NEW.account_id, OLD.account_id)
			), 0),
			updated_at = NOW()
			WHERE id = COALESCE(NEW.account_id, OLD.account_id);
			
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;
	`

	if _, err := s.db.ExecContext(ctx, triggerFunc); err != nil {
		return fmt.Errorf("failed to create trigger function: %w", err)
	}

	// Create trigger
	trigger := `
		DROP TRIGGER IF EXISTS trigger_update_equity ON positions;
		CREATE TRIGGER trigger_update_equity
			AFTER INSERT OR UPDATE OR DELETE ON positions
			FOR EACH ROW
			EXECUTE FUNCTION update_account_equity();
	`

	if _, err := s.db.ExecContext(ctx, trigger); err != nil {
		return fmt.Errorf("failed to create trigger: %w", err)
	}

	// Check if account exists, create if not
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM accounts WHERE id = $1)", s.accountID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check account existence: %w", err)
	}

	if !exists {
		initialBalance := 10000.0
		_, err = s.db.ExecContext(ctx, `
			INSERT INTO accounts (id, balance, equity, updated_at, created_at)
			VALUES ($1, $2, $2, NOW(), NOW())
		`, s.accountID, initialBalance)
		if err != nil {
			return fmt.Errorf("failed to create initial account: %w", err)
		}
	}

	return nil
}

// Close closes the database connection
func (s *PostgresStore) Close() error {
	return s.db.Close()
}

// SaveState persists the broker state to PostgreSQL
func (s *PostgresStore) SaveState(broker engine.Broker) error {
	account := broker.GetAccount()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update account
	_, err = tx.ExecContext(ctx, `
		UPDATE accounts 
		SET balance = $1, equity = $2, updated_at = $3
		WHERE id = $4
	`, account.Balance, account.Equity, account.UpdatedAt, s.accountID)
	if err != nil {
		return fmt.Errorf("failed to update account: %w", err)
	}

	// Clear existing positions for this account
	_, err = tx.ExecContext(ctx, `DELETE FROM positions WHERE account_id = $1`, s.accountID)
	if err != nil {
		return fmt.Errorf("failed to clear positions: %w", err)
	}

	// Insert current positions
	for _, pos := range account.Positions {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO positions (account_id, symbol, side, entry_price, quantity, current_price, unrealized_pnl, opened_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, s.accountID, pos.Symbol, pos.Side, pos.EntryPrice, pos.Quantity, pos.CurrentPrice, pos.UnrealizedPnL, pos.OpenedAt)
		if err != nil {
			return fmt.Errorf("failed to insert position: %w", err)
		}
	}

	// Clear existing open orders
	_, err = tx.ExecContext(ctx, `DELETE FROM orders WHERE account_id = $1 AND status = 'pending'`, s.accountID)
	if err != nil {
		return fmt.Errorf("failed to clear open orders: %w", err)
	}

	// Insert open orders
	for _, order := range account.OpenOrders {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO orders (id, account_id, timestamp, side, type, symbol, quantity, price, status, filled_price, filled_qty, fee, slippage)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		`, order.ID, s.accountID, order.Timestamp, order.Side, order.Type, order.Symbol, order.Quantity, order.Price, order.Status, order.FilledPrice, order.FilledQty, order.Fee, order.Slippage)
		if err != nil {
			return fmt.Errorf("failed to insert order: %w", err)
		}
	}

	// Save trade history (only new trades not already in DB)
	for _, trade := range account.TradeHistory {
		// Use INSERT with ON CONFLICT DO NOTHING to avoid duplicates
		_, err = tx.ExecContext(ctx, `
			INSERT INTO trades (id, account_id, symbol, side, entry_price, exit_price, quantity, pnl, fee, net_pnl, opened_at, closed_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			ON CONFLICT (id) DO NOTHING
		`, trade.ID, s.accountID, trade.Symbol, trade.Side, trade.EntryPrice, trade.ExitPrice, trade.Quantity, trade.PnL, trade.Fee, trade.NetPnL, trade.OpenedAt, trade.ClosedAt)
		if err != nil {
			return fmt.Errorf("failed to insert trade: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// LoadState restores the broker state from PostgreSQL
func (s *PostgresStore) LoadState(broker engine.Broker) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// This is a read-only operation for now
	// In a full implementation, you'd need to extend the Broker interface
	// with a SetState method to properly restore state from the database

	// For now, we'll just verify the account exists and return the data
	var balance, equity float64
	var updatedAt time.Time

	err := s.db.QueryRowContext(ctx, `
		SELECT balance, equity, updated_at
		FROM accounts
		WHERE id = $1
	`, s.accountID).Scan(&balance, &equity, &updatedAt)

	if err == sql.ErrNoRows {
		return fmt.Errorf("account not found in database")
	}
	if err != nil {
		return fmt.Errorf("failed to load account: %w", err)
	}

	// TODO: To fully restore state, we would need:
	// 1. Extend Broker interface with SetState method
	// 2. Load positions from database
	// 3. Load open orders from database
	// 4. Load trade history from database
	// 5. Call broker.SetState() with loaded data

	// For now, just return success if account exists
	return nil
}

// GetAccount retrieves account information from the database
func (s *PostgresStore) GetAccount(ctx context.Context) (*engine.Account, error) {
	account := &engine.Account{
		Positions:    []*engine.Position{},
		OpenOrders:   []*engine.Order{},
		TradeHistory: []*engine.Trade{},
	}

	// Load account info
	err := s.db.QueryRowContext(ctx, `
		SELECT balance, equity, updated_at
		FROM accounts
		WHERE id = $1
	`, s.accountID).Scan(&account.Balance, &account.Equity, &account.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to load account: %w", err)
	}

	// Load positions
	rows, err := s.db.QueryContext(ctx, `
		SELECT symbol, side, entry_price, quantity, current_price, unrealized_pnl, opened_at
		FROM positions
		WHERE account_id = $1
	`, s.accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to load positions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		pos := &engine.Position{}
		err := rows.Scan(&pos.Symbol, &pos.Side, &pos.EntryPrice, &pos.Quantity, &pos.CurrentPrice, &pos.UnrealizedPnL, &pos.OpenedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan position: %w", err)
		}
		account.Positions = append(account.Positions, pos)
	}

	// Load trade history
	tradeRows, err := s.db.QueryContext(ctx, `
		SELECT id, symbol, side, entry_price, exit_price, quantity, pnl, fee, net_pnl, opened_at, closed_at
		FROM trades
		WHERE account_id = $1
		ORDER BY closed_at ASC
	`, s.accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to load trades: %w", err)
	}
	defer tradeRows.Close()

	for tradeRows.Next() {
		trade := &engine.Trade{}
		err := tradeRows.Scan(&trade.ID, &trade.Symbol, &trade.Side, &trade.EntryPrice, &trade.ExitPrice, &trade.Quantity, &trade.PnL, &trade.Fee, &trade.NetPnL, &trade.OpenedAt, &trade.ClosedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trade: %w", err)
		}
		account.TradeHistory = append(account.TradeHistory, trade)
	}

	return account, nil
}

// GetTradeHistory retrieves trade history with optional filters
func (s *PostgresStore) GetTradeHistory(ctx context.Context, symbol string, limit int) ([]*engine.Trade, error) {
	query := `
		SELECT id, symbol, side, entry_price, exit_price, quantity, pnl, fee, net_pnl, opened_at, closed_at
		FROM trades
		WHERE account_id = $1
	`
	args := []interface{}{s.accountID}

	if symbol != "" {
		query += ` AND symbol = $2`
		args = append(args, symbol)
		query += ` ORDER BY closed_at DESC LIMIT $3`
		args = append(args, limit)
	} else {
		query += ` ORDER BY closed_at DESC LIMIT $2`
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query trades: %w", err)
	}
	defer rows.Close()

	var trades []*engine.Trade
	for rows.Next() {
		trade := &engine.Trade{}
		err := rows.Scan(&trade.ID, &trade.Symbol, &trade.Side, &trade.EntryPrice, &trade.ExitPrice, &trade.Quantity, &trade.PnL, &trade.Fee, &trade.NetPnL, &trade.OpenedAt, &trade.ClosedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trade: %w", err)
		}
		trades = append(trades, trade)
	}

	return trades, nil
}

// GetPerformanceMetrics calculates performance metrics from trade history
func (s *PostgresStore) GetPerformanceMetrics(ctx context.Context) (map[string]interface{}, error) {
	var totalTrades int
	var winningTrades int
	var totalPnL, totalFees float64
	var avgWin, avgLoss float64

	err := s.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*) as total_trades,
			COUNT(*) FILTER (WHERE net_pnl > 0) as winning_trades,
			COALESCE(SUM(net_pnl), 0) as total_pnl,
			COALESCE(SUM(fee), 0) as total_fees,
			COALESCE(AVG(net_pnl) FILTER (WHERE net_pnl > 0), 0) as avg_win,
			COALESCE(AVG(net_pnl) FILTER (WHERE net_pnl < 0), 0) as avg_loss
		FROM trades
		WHERE account_id = $1
	`, s.accountID).Scan(&totalTrades, &winningTrades, &totalPnL, &totalFees, &avgWin, &avgLoss)

	if err != nil {
		return nil, fmt.Errorf("failed to calculate metrics: %w", err)
	}

	winRate := 0.0
	if totalTrades > 0 {
		winRate = float64(winningTrades) / float64(totalTrades) * 100
	}

	return map[string]interface{}{
		"total_trades":    totalTrades,
		"winning_trades":  winningTrades,
		"losing_trades":   totalTrades - winningTrades,
		"win_rate":        winRate,
		"total_pnl":       totalPnL,
		"total_fees":      totalFees,
		"avg_win":         avgWin,
		"avg_loss":        avgLoss,
		"profit_factor":   calculateProfitFactor(avgWin, avgLoss, winningTrades, totalTrades-winningTrades),
	}, nil
}

func calculateProfitFactor(avgWin, avgLoss float64, wins, losses int) float64 {
	if losses == 0 || avgLoss == 0 {
		return 0
	}
	totalWins := avgWin * float64(wins)
	totalLosses := avgLoss * float64(losses) * -1 // avgLoss is negative
	if totalLosses == 0 {
		return 0
	}
	return totalWins / totalLosses
}
