package bot

import (
	"candlecore/internal/exchange"
	"time"
)

// Signal represents a trading signal
type Signal string

const (
	SignalBuy  Signal = "buy"
	SignalSell Signal = "sell"
	SignalHold Signal = "hold"
)

// Decision represents a bot decision with reasoning
type Decision struct {
	Timestamp  time.Time         `json:"timestamp"`
	Signal     Signal            `json:"signal"`
	Symbol     string            `json:"symbol"`
	Price      float64           `json:"price"`
	Quantity   float64           `json:"quantity"`
	Confidence float64           `json:"confidence"` // 0-100
	Reasoning  string            `json:"reasoning"`
	Indicators map[string]float64 `json:"indicators"` // indicator values at decision time
}

// Position represents an open position
type Position struct {
	ID         string    `json:"id"`
	Symbol     string    `json:"symbol"`
	Side       string    `json:"side"` // "long" or "short"
	EntryPrice float64   `json:"entry_price"`
	Quantity   float64   `json:"quantity"`
	CurrentPrice float64 `json:"current_price"`
	UnrealizedPnL float64 `json:"unrealized_pnl"`
	RealizedPnL   float64 `json:"realized_pnl"`
	OpenedAt   time.Time `json:"opened_at"`
	ClosedAt   *time.Time `json:"closed_at,omitempty"`
}

// Strategy defines the interface for trading strategies
type Strategy interface {
	// Name returns the strategy name
	Name() string
	
	// Analyze analyzes candles and produces a decision
	Analyze(candles []exchange.Candle) (*Decision, error)
	
	// Configure updates strategy parameters
	Configure(params map[string]interface{}) error
}

// Bot represents the trading bot
type Bot struct {
	strategy      Strategy
	symbol        string
	timeframe     exchange.Timeframe
	provider      exchange.DataProvider
	position      *Position
	balance       float64
	initialBalance float64
	trades        []Position
}

// Config contains bot configuration
type Config struct {
	Symbol         string
	Timeframe      exchange.Timeframe
	InitialBalance float64
	PositionSize   float64 // Percentage of balance per trade (0-100)
}

// NewBot creates a new trading bot
func NewBot(strategy Strategy, provider exchange.DataProvider, config Config) *Bot {
	return &Bot{
		strategy:       strategy,
		symbol:         config.Symbol,
		timeframe:      config.Timeframe,
		provider:       provider,
		balance:        config.InitialBalance,
		initialBalance: config.InitialBalance,
		trades:         make([]Position, 0),
	}
}

// ProcessCandle processes a new candle and executes strategy
func (b *Bot) ProcessCandle(candle exchange.Candle) (*Decision, error) {
	// Get recent candles for analysis
	candles, err := b.provider.GetCandles(b.symbol, b.timeframe, 200)
	if err != nil {
		return nil, err
	}

	// Run strategy analysis
	decision, err := b.strategy.Analyze(candles)
	if err != nil {
		return nil, err
	}

	// Execute decision
	b.executeDecision(decision, candle)

	return decision, nil
}

// executeDecision executes a trading decision
func (b *Bot) executeDecision(decision *Decision, candle exchange.Candle) {
	switch decision.Signal {
	case SignalBuy:
		if b.position == nil || b.position.Side == "short" {
			b.enterPosition("long", candle.Close, decision)
		}
	case SignalSell:
		if b.position != nil && b.position.Side == "long" {
			b.closePosition(candle.Close)
		}
	case SignalHold:
		// Update unrealized PnL if position exists
		if b.position != nil {
			b.updatePosition(candle.Close)
		}
	}
}

// enterPosition opens a new position
func (b *Bot) enterPosition(side string, price float64, decision *Decision) {
	// Close existing position if opposite direction
	if b.position != nil && b.position.Side != side {
		b.closePosition(price)
	}

	// Calculate position size (use 10% of balance for simplicity)
	quantity := (b.balance * 0.1) / price

	b.position = &Position{
		ID:         b.generateID(),
		Symbol:     b.symbol,
		Side:       side,
		EntryPrice: price,
		Quantity:   quantity,
		CurrentPrice: price,
		UnrealizedPnL: 0,
		OpenedAt:   decision.Timestamp,
	}
}

// closePosition closes the current position
func (b *Bot) closePosition(price float64) {
	if b.position == nil {
		return
	}

	// Calculate PnL
	var pnl float64
	if b.position.Side == "long" {
		pnl = (price - b.position.EntryPrice) * b.position.Quantity
	} else {
		pnl = (b.position.EntryPrice - price) * b.position.Quantity
	}

	b.position.CurrentPrice = price
	b.position.RealizedPnL = pnl
	now := time.Now()
	b.position.ClosedAt = &now

	// Update balance
	b.balance += pnl

	// Store trade
	b.trades = append(b.trades, *b.position)

	// Clear position
	b.position = nil
}

// updatePosition updates unrealized PnL
func (b *Bot) updatePosition(price float64) {
	if b.position == nil {
		return
	}

	b.position.CurrentPrice = price
	
	if b.position.Side == "long" {
		b.position.UnrealizedPnL = (price - b.position.EntryPrice) * b.position.Quantity
	} else {
		b.position.UnrealizedPnL = (b.position.EntryPrice - price) * b.position.Quantity
	}
}

// GetPosition returns the current position
func (b *Bot) GetPosition() *Position {
	return b.position
}

// GetBalance returns current balance
func (b *Bot) GetBalance() float64 {
	return b.balance
}

// GetTotalPnL returns total profit/loss
func (b *Bot) GetTotalPnL() float64 {
	total := b.balance - b.initialBalance
	if b.position != nil {
		total += b.position.UnrealizedPnL
	}
	return total
}

// GetTrades returns all completed trades
func (b *Bot) GetTrades() []Position {
	return b.trades
}

// generateID generates a simple ID
func (b *Bot) generateID() string {
	return time.Now().Format("20060102150405")
}
