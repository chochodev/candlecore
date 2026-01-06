package broker

import (
	"fmt"
	"sync"
	"time"

	"candlecore/internal/engine"
	"candlecore/internal/logger"

	"github.com/google/uuid"
)

// PaperBroker simulates a trading broker for backtesting and paper trading
// It handles order execution, fees, slippage, and position management
type PaperBroker struct {
	mu sync.RWMutex

	initialBalance float64
	account        *engine.Account
	positions      map[string]*engine.Position
	marketPrices   map[string]float64

	// Fee configuration
	takerFee    float64 // percentage, e.g., 0.001 for 0.1%
	makerFee    float64
	slippageBps float64 // basis points

	logger logger.Logger
}

// NewPaperBroker creates a new paper trading broker
func NewPaperBroker(initialBalance, takerFee, makerFee, slippageBps float64, log logger.Logger) *PaperBroker {
	return &PaperBroker{
		initialBalance: initialBalance,
		account: &engine.Account{
			Balance:      initialBalance,
			Equity:       initialBalance,
			Positions:    []*engine.Position{},
			OpenOrders:   []*engine.Order{},
			TradeHistory: []*engine.Trade{},
			UpdatedAt:    time.Now(),
		},
		positions:    make(map[string]*engine.Position),
		marketPrices: make(map[string]float64),
		takerFee:     takerFee,
		makerFee:     makerFee,
		slippageBps:  slippageBps,
		logger:       log,
	}
}

// GetAccount returns the current account state
func (b *PaperBroker) GetAccount() *engine.Account {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Create a copy to avoid race conditions
	account := &engine.Account{
		Balance:      b.account.Balance,
		Equity:       b.calculateEquity(),
		Positions:    make([]*engine.Position, len(b.account.Positions)),
		OpenOrders:   make([]*engine.Order, len(b.account.OpenOrders)),
		TradeHistory: b.account.TradeHistory, // Share history (read-only)
		UpdatedAt:    time.Now(),
	}

	copy(account.Positions, b.account.Positions)
	copy(account.OpenOrders, b.account.OpenOrders)

	return account
}

// PlaceOrder simulates order execution
func (b *PaperBroker) PlaceOrder(order *engine.Order) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Generate order ID
	order.ID = uuid.New().String()

	// Validate order
	if err := b.validateOrder(order); err != nil {
		order.Status = engine.OrderStatusRejected
		b.logger.Warn("Order rejected", "error", err, "order_id", order.ID)
		return err
	}

	// For market orders, execute immediately
	if order.Type == engine.OrderTypeMarket {
		return b.executeMarketOrder(order)
	}

	// For limit orders, add to open orders
	// (Not implemented in this version, but stub is here)
	order.Status = engine.OrderStatusPending
	b.account.OpenOrders = append(b.account.OpenOrders, order)

	return nil
}

// CancelOrder cancels an open order
func (b *PaperBroker) CancelOrder(orderID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i, order := range b.account.OpenOrders {
		if order.ID == orderID {
			order.Status = engine.OrderStatusCancelled
			// Remove from open orders
			b.account.OpenOrders = append(b.account.OpenOrders[:i], b.account.OpenOrders[i+1:]...)
			b.logger.Info("Order cancelled", "order_id", orderID)
			return nil
		}
	}

	return fmt.Errorf("order not found: %s", orderID)
}

// UpdateMarketPrice updates the current market price for a symbol
func (b *PaperBroker) UpdateMarketPrice(symbol string, price float64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.marketPrices[symbol] = price

	// Update unrealized PnL for positions
	if pos, exists := b.positions[symbol]; exists {
		pos.CurrentPrice = price
		pos.UnrealizedPnL = b.calculatePositionPnL(pos)
	}
}

// GetPosition returns the current position for a symbol
func (b *PaperBroker) GetPosition(symbol string) *engine.Position {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.positions[symbol]
}

// executeMarketOrder simulates immediate execution of a market order
func (b *PaperBroker) executeMarketOrder(order *engine.Order) error {
	// Calculate execution price with slippage
	marketPrice := b.marketPrices[order.Symbol]
	if marketPrice == 0 {
		marketPrice = order.Price // Use order price if no market data
	}

	slippage := marketPrice * (b.slippageBps / 10000.0)
	if order.Side == engine.OrderSideBuy {
		order.FilledPrice = marketPrice + slippage
	} else {
		order.FilledPrice = marketPrice - slippage
	}

	order.FilledQty = order.Quantity
	order.Slippage = slippage

	// Calculate fee (using taker fee for market orders)
	orderValue := order.FilledPrice * order.FilledQty
	order.Fee = orderValue * b.takerFee

	// Check if we have enough balance
	if order.Side == engine.OrderSideBuy {
		totalCost := orderValue + order.Fee
		if totalCost > b.account.Balance {
			return fmt.Errorf("insufficient balance: need %.2f, have %.2f", totalCost, b.account.Balance)
		}
		b.account.Balance -= totalCost
	}

	// Update or create position
	if order.Side == engine.OrderSideBuy {
		b.openPosition(order)
	} else {
		b.closePosition(order)
	}

	order.Status = engine.OrderStatusFilled
	b.account.UpdatedAt = time.Now()

	b.logger.Info("Order executed",
		"order_id", order.ID,
		"side", order.Side,
		"symbol", order.Symbol,
		"quantity", order.FilledQty,
		"price", order.FilledPrice,
		"fee", order.Fee,
		"balance", b.account.Balance,
	)

	return nil
}

// openPosition opens a new position or adds to existing
func (b *PaperBroker) openPosition(order *engine.Order) {
	pos, exists := b.positions[order.Symbol]
	if !exists {
		// Create new position
		pos = &engine.Position{
			Symbol:       order.Symbol,
			Side:         order.Side,
			EntryPrice:   order.FilledPrice,
			Quantity:     order.FilledQty,
			CurrentPrice: order.FilledPrice,
			OpenedAt:     order.Timestamp,
		}
		b.positions[order.Symbol] = pos
		b.account.Positions = append(b.account.Positions, pos)
	} else {
		// Average up position
		totalCost := (pos.EntryPrice * pos.Quantity) + (order.FilledPrice * order.FilledQty)
		pos.Quantity += order.FilledQty
		pos.EntryPrice = totalCost / pos.Quantity
	}

	b.logger.Debug("Position opened/updated",
		"symbol", order.Symbol,
		"quantity", pos.Quantity,
		"entry_price", pos.EntryPrice,
	)
}

// closePosition closes or reduces a position
func (b *PaperBroker) closePosition(order *engine.Order) {
	pos, exists := b.positions[order.Symbol]
	if !exists {
		b.logger.Warn("Attempting to close non-existent position", "symbol", order.Symbol)
		return
	}

	// Calculate P&L
	pnl := (order.FilledPrice - pos.EntryPrice) * order.FilledQty
	netPnl := pnl - order.Fee

	// Create trade record
	trade := &engine.Trade{
		ID:         uuid.New().String(),
		Symbol:     order.Symbol,
		Side:       pos.Side,
		EntryPrice: pos.EntryPrice,
		ExitPrice:  order.FilledPrice,
		Quantity:   order.FilledQty,
		PnL:        pnl,
		Fee:        order.Fee,
		NetPnL:     netPnl,
		OpenedAt:   pos.OpenedAt,
		ClosedAt:   order.Timestamp,
	}

	b.account.TradeHistory = append(b.account.TradeHistory, trade)

	// Update balance
	proceeds := order.FilledPrice * order.FilledQty
	b.account.Balance += proceeds - order.Fee

	// Update position
	pos.Quantity -= order.FilledQty

	// Remove position if fully closed
	if pos.Quantity <= 0.0001 {
		delete(b.positions, order.Symbol)
		// Remove from account positions slice
		for i, p := range b.account.Positions {
			if p.Symbol == order.Symbol {
				b.account.Positions = append(b.account.Positions[:i], b.account.Positions[i+1:]...)
				break
			}
		}
	}

	b.logger.Info("Position closed",
		"symbol", order.Symbol,
		"pnl", pnl,
		"net_pnl", netPnl,
		"remaining_qty", pos.Quantity,
	)
}

// validateOrder checks if an order is valid
func (b *PaperBroker) validateOrder(order *engine.Order) error {
	if order.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}

	if order.Symbol == "" {
		return fmt.Errorf("symbol is required")
	}

	return nil
}

// calculateEquity calculates total account equity (balance + unrealized PnL)
func (b *PaperBroker) calculateEquity() float64 {
	equity := b.account.Balance

	for _, pos := range b.positions {
		equity += pos.UnrealizedPnL
	}

	return equity
}

// calculatePositionPnL calculates unrealized P&L for a position
func (b *PaperBroker) calculatePositionPnL(pos *engine.Position) float64 {
	if pos.CurrentPrice == 0 {
		return 0
	}

	return (pos.CurrentPrice - pos.EntryPrice) * pos.Quantity
}
