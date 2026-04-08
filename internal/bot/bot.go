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
	TrailingSL *float64           `json:"trailing_sl,omitempty"` // Propagated from strategy
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
	StopLoss   float64   `json:"stop_loss"`
	TakeProfit float64   `json:"take_profit"`
	TrailingSL *float64   `json:"trailing_sl,omitempty"` // For the dynamic SL line
}

// Strategy defines the interface for trading strategies
type Strategy interface {
	// Name returns the strategy name
	Name() string
	
	// Analyze analyzes candles and produces a decision
	Analyze(candles []exchange.Candle, currentPos *Position) (*Decision, error)
	
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
	balance        float64
	initialBalance float64
	trades         []Position
	balanceHistory []float64
	history        []exchange.Candle
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
		balanceHistory: []float64{config.InitialBalance},
		history:        make([]exchange.Candle, 0),
	}
}

// ProcessCandle processes a new candle and executes strategy
func (b *Bot) ProcessCandle(candle exchange.Candle) (*Decision, error) {
	// Maintain internal sliding window of history
	b.history = append(b.history, candle)
	if len(b.history) > 300 {
		b.history = b.history[1:]
	}

	// Warm-up check (need enough data for indicators)
	if len(b.history) < 30 {
		return &Decision{
			Timestamp: candle.Timestamp,
			Signal:    SignalHold,
			Reasoning: "Engine warming up...",
		}, nil
	}

	// Run strategy analysis on the actual historical context
	decision, err := b.strategy.Analyze(b.history, b.position)
	if err != nil {
		return nil, err
	}

	// Execute decision
	b.executeDecision(decision, candle)

	// APPLY RISK GUARDS (Live/Paper Enforcement)
	b.ApplyRiskGuards(candle)

	return decision, nil
}

// ApplyRiskGuards enforces SL/TP and handles dynamic "Shield" logic
func (b *Bot) ApplyRiskGuards(candle exchange.Candle) {
	if b.position == nil {
		return
	}

	pos := b.position
	price := candle.Close

	// ─── SHIELD LOGIC (DYNAMIC SL) ───────────────────────────────────────────

	if pos.Side == "long" {
		// ─── SHIELD LOGIC (LONG) ─────────────────────────────────────────────
		feeBreakeven := pos.EntryPrice * 1.0022
		if candle.High > feeBreakeven && pos.TrailingSL == nil {
			pos.TrailingSL = &feeBreakeven
		}

		warpThreshold := pos.EntryPrice + (pos.TakeProfit-pos.EntryPrice)*0.5
		if candle.High >= warpThreshold {
			lockProfit := pos.EntryPrice + (pos.TakeProfit-pos.EntryPrice)*0.25
			if pos.TrailingSL == nil || lockProfit > *pos.TrailingSL {
				pos.TrailingSL = &lockProfit
			}
		}

		// ─── EXIT ENFORCEMENT (LONG) ──────────────────────────────────────────
		slPrice := pos.StopLoss
		if pos.TrailingSL != nil {
			slPrice = *pos.TrailingSL
		}

		if candle.Low <= slPrice {
			b.closePosition(slPrice)
		} else if candle.High >= pos.TakeProfit {
			b.closePosition(pos.TakeProfit)
		}

	} else if pos.Side == "short" {
		// ─── SHIELD LOGIC (SHORT) ────────────────────────────────────────────
		feeBreakeven := pos.EntryPrice * 0.9978
		if candle.Low < feeBreakeven && pos.TrailingSL == nil {
			pos.TrailingSL = &feeBreakeven
		}

		warpThreshold := pos.EntryPrice - (pos.EntryPrice-pos.TakeProfit)*0.5
		if candle.Low <= warpThreshold {
			lockProfit := pos.EntryPrice - (pos.EntryPrice-pos.TakeProfit)*0.25
			if pos.TrailingSL == nil || lockProfit < *pos.TrailingSL {
				pos.TrailingSL = &lockProfit
			}
		}

		// ─── EXIT ENFORCEMENT (SHORT) ─────────────────────────────────────────
		slPrice := pos.StopLoss
		if pos.TrailingSL != nil {
			slPrice = *pos.TrailingSL
		}

		if candle.High >= slPrice {
			b.closePosition(slPrice)
		} else if candle.Low <= pos.TakeProfit {
			b.closePosition(pos.TakeProfit)
		}
	}

	// Update unrealized PnL for streaming
	if b.position != nil {
		b.updatePosition(price)
	}
}

// RunBacktest runs a fast backtest on a slice of candles
func (b *Bot) RunBacktest(candles []exchange.Candle) error {
	for i, candle := range candles {
		// Minimum 200 candles for robust strategy warm-up
		if i < 200 {
			continue
		}

		// Run strategy on historical window (last 200 candles)
		start := i - 199
		if start < 0 {
			start = 0
		}
		window := candles[start : i+1]

		// Run strategy analysis
		decision, err := b.strategy.Analyze(window, b.position)
		if err != nil {
			return err
		}

		// Execute decision (this handles positions and balance internally)
		b.executeDecision(decision, candle)
		
		// Record balance for drawdown/sharpe
		currBalance := b.balance
		if b.position != nil {
			// AUTONOMOUS RISK GUARD: Check intra-candle TP/SL hits
			b.ApplyRiskGuards(candle)

			// If position still exists, update normal PnL
			if b.position != nil {
				currBalance += b.position.UnrealizedPnL
			}
		}
		b.balanceHistory = append(b.balanceHistory, currBalance)
	}
	return nil
}

// GetBalanceHistory returns the history of balance/equity
func (b *Bot) GetBalanceHistory() []float64 {
	return b.balanceHistory
}


// executeDecision executes a trading decision
func (b *Bot) executeDecision(decision *Decision, candle exchange.Candle) {
	// 1. Sync TrailingSL from decision to current position
	if b.position != nil && decision.TrailingSL != nil {
		b.position.TrailingSL = decision.TrailingSL
	}

	switch decision.Signal {
	case SignalBuy:
		if b.position == nil {
			b.enterPosition("long", candle.Close, decision)
		} else if b.position.Side == "short" {
			b.closePosition(candle.Close)
			// Only re-enter Long if it's an explicit entry reasoning
			if decision.Reasoning == "Pulse Entry (Long)" {
				b.enterPosition("long", candle.Close, decision)
			}
		}
	case SignalSell:
		if b.position == nil {
			b.enterPosition("short", candle.Close, decision)
		} else if b.position.Side == "long" {
			if decision.Quantity > 0 && decision.Quantity < b.position.Quantity {
				b.partialClose(candle.Close, decision.Quantity)
			} else {
				b.closePosition(candle.Close)
				// Only re-enter Short if it's an explicit entry reasoning
				if decision.Reasoning == "Pulse Entry (Short)" {
					b.enterPosition("short", candle.Close, decision)
				}
			}
		}
	case SignalHold:
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

	// Calculate slippage (0.05% typically for high-liquidity pairs)
	slippedPrice := price * 1.0005 

	// Calculate position size (use 10% of balance for simplicity)
	quantity := (b.balance * 0.1) / slippedPrice
	fee := (quantity * slippedPrice) * 0.001 // 0.1% Taker Fee
	b.balance -= fee

	b.position = &Position{
		ID:         b.generateID(),
		Symbol:     b.symbol,
		Side:       side,
		EntryPrice: slippedPrice,
		Quantity:   quantity,
		CurrentPrice: slippedPrice,
		UnrealizedPnL: 0,
		OpenedAt:   decision.Timestamp,
		StopLoss:   decision.Price * 0.992,   // Dynamic fallback: 0.8% stop
		TakeProfit: decision.Price * 1.018,   // Dynamic fallback: 1.8% TP
	}

	// STRATEGY OVERRIDE: Prioritize adapter TP/SL if calibrated
	// We'll add custom logic here to read from strategy metadata in next pulse
	
	// Correct for Short positions
	if side == "short" {
		b.position.StopLoss = decision.Price * 1.008
		b.position.TakeProfit = decision.Price * 0.982
	}
}

// closePosition closes the current position
func (b *Bot) closePosition(price float64) {
	if b.position == nil {
		return
	}

	slippedPrice := price * 0.9995

	// Calculate PnL
	var pnl float64
	if b.position.Side == "long" {
		pnl = (slippedPrice - b.position.EntryPrice) * b.position.Quantity
	} else {
		pnl = (b.position.EntryPrice - slippedPrice) * b.position.Quantity
	}

	b.position.CurrentPrice = slippedPrice
	b.position.RealizedPnL = pnl
	now := time.Now()
	b.position.ClosedAt = &now

	// Update balance
	exitFee := (b.position.Quantity * slippedPrice) * 0.001
	b.balance += pnl - exitFee

	// Store trade
	trade := *b.position
	trade.RealizedPnL -= exitFee
	b.trades = append(b.trades, trade)

	// Clear position
	b.position = nil
}

// partialClose closes a portion of the current position
func (b *Bot) partialClose(price, quantity float64) {
	if b.position == nil || quantity <= 0 || quantity >= b.position.Quantity {
		return
	}

	// Calculate PnL for the sold portion
	var pnl float64
	if b.position.Side == "long" {
		pnl = (price - b.position.EntryPrice) * quantity
	} else {
		pnl = (b.position.EntryPrice - price) * quantity
	}

	// Record the partial trade
	partialTrade := *b.position
	partialTrade.Quantity = quantity
	partialTrade.RealizedPnL = pnl
	partialTrade.CurrentPrice = price
	now := time.Now()
	partialTrade.ClosedAt = &now
	b.trades = append(b.trades, partialTrade)

	// Update remaining position and balance
	b.position.Quantity -= quantity
	b.balance += (price * quantity) // Simplified: adding proceeds to balance
	
	// Note: We don't update entry price for partial sells, 
	// but we could adjust it if it were a partial buy (DCA).
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
	return time.Now().Format("20060102150405.000000000")
}
