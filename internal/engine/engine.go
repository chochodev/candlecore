package engine

import (
	"context"
	"fmt"

	"candlecore/internal/logger"
)

// Broker defines the interface for order execution
// This abstraction allows swapping between paper trading and real brokers
type Broker interface {
	// GetAccount returns the current account state
	GetAccount() *Account

	// PlaceOrder submits a new order
	PlaceOrder(order *Order) error

	// CancelOrder cancels an existing order
	CancelOrder(orderID string) error

	// UpdateMarketPrice updates the current market price for P&L calculations
	UpdateMarketPrice(symbol string, price float64)

	// GetPosition returns the current position for a symbol
	GetPosition(symbol string) *Position
}

// Strategy defines the interface for trading strategies
// Implement this interface to create custom trading logic
type Strategy interface {
	// Name returns the strategy name
	Name() string

	// OnCandle is called for each new candle
	// It should analyze the candle and return a trading signal
	OnCandle(candle Candle, account *Account) Signal

	// OnTrade is called after a trade is executed
	// Useful for tracking strategy performance
	OnTrade(trade *Trade)
}

// StateStore defines the interface for persisting engine state
type StateStore interface {
	SaveState(broker Broker) error
	LoadState(broker Broker) error
}

// Engine is the main trading engine that orchestrates everything
type Engine struct {
	broker Broker
	strategy Strategy
	store  StateStore
	logger logger.Logger
}

// New creates a new trading engine
func New(broker Broker, strategy Strategy, store StateStore, log logger.Logger) *Engine {
	return &Engine{
		broker:   broker,
		strategy: strategy,
		store:    store,
		logger:   log,
	}
}

// Run executes the backtest/paper trading loop
func (e *Engine) Run(ctx context.Context, candles []Candle) error {
	e.logger.Info("Engine starting",
		"strategy", e.strategy.Name(),
		"candles", len(candles),
	)

	for i, candle := range candles {
		// Check if context was cancelled (graceful shutdown)
		select {
		case <-ctx.Done():
			e.logger.Info("Engine stopped by context", "processed_candles", i)
			return ctx.Err()
		default:
		}

		// Update market price for position valuation
		e.broker.UpdateMarketPrice("BTC/USD", candle.Close)

		// Get current account state
		account := e.broker.GetAccount()

		// Log current state
		e.logger.Debug("Processing candle",
			"index", i,
			"timestamp", candle.Timestamp,
			"close", candle.Close,
			"balance", account.Balance,
			"equity", account.Equity,
		)

		// Get strategy signal
		signal := e.strategy.OnCandle(candle, account)

		// Execute signal
		if err := e.executeSignal(signal, candle); err != nil {
			e.logger.Error("Failed to execute signal",
				"error", err,
				"signal", signal.Action,
				"candle_index", i,
			)
			// Continue processing rather than failing completely
			continue
		}

		// Periodically save state (every 10 candles for example)
		if i%10 == 0 && i > 0 {
			if err := e.store.SaveState(e.broker); err != nil {
				e.logger.Warn("Failed to save state", "error", err)
			}
		}
	}

	e.logger.Info("Engine completed successfully", "total_candles", len(candles))
	return nil
}

// executeSignal converts a strategy signal into broker orders
func (e *Engine) executeSignal(signal Signal, candle Candle) error {
	switch signal.Action {
	case SignalActionBuy:
		return e.executeBuy(signal, candle)
	case SignalActionSell:
		return e.executeSell(signal, candle)
	case SignalActionHold:
		// Do nothing
		return nil
	default:
		return fmt.Errorf("unknown signal action: %s", signal.Action)
	}
}

// executeBuy executes a buy signal
func (e *Engine) executeBuy(signal Signal, candle Candle) error {
	e.logger.Info("Executing BUY signal",
		"symbol", signal.Symbol,
		"quantity", signal.Quantity,
		"price", candle.Close,
		"reason", signal.Reason,
	)

	order := &Order{
		Timestamp: candle.Timestamp,
		Side:      OrderSideBuy,
		Type:      OrderTypeMarket,
		Symbol:    signal.Symbol,
		Quantity:  signal.Quantity,
		Price:     candle.Close, // Market order uses current price
		Status:    OrderStatusPending,
	}

	return e.broker.PlaceOrder(order)
}

// executeSell executes a sell signal
func (e *Engine) executeSell(signal Signal, candle Candle) error {
	// Check if we have a position to sell
	position := e.broker.GetPosition(signal.Symbol)
	if position == nil || position.Quantity == 0 {
		e.logger.Debug("Ignoring SELL signal - no position to close")
		return nil
	}

	e.logger.Info("Executing SELL signal",
		"symbol", signal.Symbol,
		"quantity", signal.Quantity,
		"price", candle.Close,
		"reason", signal.Reason,
	)

	order := &Order{
		Timestamp: candle.Timestamp,
		Side:      OrderSideSell,
		Type:      OrderTypeMarket,
		Symbol:    signal.Symbol,
		Quantity:  signal.Quantity,
		Price:     candle.Close,
		Status:    OrderStatusPending,
	}

	return e.broker.PlaceOrder(order)
}
