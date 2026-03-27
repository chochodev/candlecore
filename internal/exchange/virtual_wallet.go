package exchange

import (
	"fmt"
	"sync"
	"time"
)

// VirtualWallet simulates a trading wallet for dry-run mode
type VirtualWallet struct {
	balance       float64
	initialBalance float64
	positions     map[string]*VirtualPosition
	trades        []VirtualTrade
	mu            sync.RWMutex
}

// VirtualPosition represents a simulated open position
type VirtualPosition struct {
	Symbol     string
	Side       string    // "long" or "short"
	EntryPrice float64
	Quantity   float64
	OpenedAt   time.Time
}

// VirtualTrade represents a completed simulated trade
type VirtualTrade struct {
	Symbol     string
	Side       string
	EntryPrice float64
	ExitPrice  float64
	Quantity   float64
	PnL        float64
	OpenedAt   time.Time
	ClosedAt   time.Time
}

// NewVirtualWallet creates a new virtual wallet
func NewVirtualWallet(initialBalance float64) *VirtualWallet {
	return &VirtualWallet{
		balance:       initialBalance,
		initialBalance: initialBalance,
		positions:     make(map[string]*VirtualPosition),
		trades:        make([]VirtualTrade, 0),
	}
}

// GetBalance returns the current balance
func (vw *VirtualWallet) GetBalance() float64 {
	vw.mu.RLock()
	defer vw.mu.RUnlock()
	return vw.balance
}

// PlaceOrder simulates placing an order
func (vw *VirtualWallet) PlaceOrder(symbol, side string, price, quantity float64) error {
	vw.mu.Lock()
	defer vw.mu.Unlock()

	cost := price * quantity

	if side == "buy" || side == "long" {
		if cost > vw.balance {
			return fmt.Errorf("insufficient balance: have %.2f, need %.2f", vw.balance, cost)
		}

		vw.balance -= cost
		vw.positions[symbol] = &VirtualPosition{
			Symbol:     symbol,
			Side:       "long",
			EntryPrice: price,
			Quantity:   quantity,
			OpenedAt:   time.Now(),
		}
	}

	return nil
}

// ClosePosition simulates closing a position
func (vw *VirtualWallet) ClosePosition(symbol string, exitPrice float64) error {
	vw.mu.Lock()
	defer vw.mu.Unlock()

	pos, exists := vw.positions[symbol]
	if !exists {
		return fmt.Errorf("no position found for %s", symbol)
	}

	var pnl float64
	if pos.Side == "long" {
		proceeds := exitPrice * pos.Quantity
		vw.balance += proceeds
		pnl = proceeds - (pos.EntryPrice * pos.Quantity)
	}

	// Record trade
	vw.trades = append(vw.trades, VirtualTrade{
		Symbol:     symbol,
		Side:       pos.Side,
		EntryPrice: pos.EntryPrice,
		ExitPrice:  exitPrice,
		Quantity:   pos.Quantity,
		PnL:        pnl,
		OpenedAt:   pos.OpenedAt,
		ClosedAt:   time.Now(),
	})

	delete(vw.positions, symbol)
	return nil
}

// GetPosition returns the current position for a symbol
func (vw *VirtualWallet) GetPosition(symbol string) *VirtualPosition {
	vw.mu.RLock()
	defer vw.mu.RUnlock()
	return vw.positions[symbol]
}

// GetTrades returns all completed trades
func (vw *VirtualWallet) GetTrades() []VirtualTrade {
	vw.mu.RLock()
	defer vw.mu.RUnlock()
	return vw.trades
}

// GetTotalPnL calculates total profit/loss
func (vw *VirtualWallet) GetTotalPnL() float64 {
	vw.mu.RLock()
	defer vw.mu.RUnlock()

	total := vw.balance - vw.initialBalance

	// Add unrealized PnL from open positions
	for _, pos := range vw.positions {
		// We'd need current price here, simplified for now
		_ = pos
	}

	return total
}

// Reset resets the wallet to initial state
func (vw *VirtualWallet) Reset() {
	vw.mu.Lock()
	defer vw.mu.Unlock()

	vw.balance = vw.initialBalance
	vw.positions = make(map[string]*VirtualPosition)
	vw.trades = make([]VirtualTrade, 0)
}
