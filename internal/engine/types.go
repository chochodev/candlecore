package engine

import (
	"time"
)

// Candle represents OHLCV candle data
type Candle struct {
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}

// OrderSide represents the direction of an order
type OrderSide string

const (
	OrderSideBuy  OrderSide = "buy"
	OrderSideSell OrderSide = "sell"
)

// OrderType represents the type of order
type OrderType string

const (
	OrderTypeMarket OrderType = "market"
	OrderTypeLimit  OrderType = "limit"
)

// OrderStatus represents the current status of an order
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusFilled    OrderStatus = "filled"
	OrderStatusCancelled OrderStatus = "cancelled"
	OrderStatusRejected  OrderStatus = "rejected"
)

// Order represents a trading order
type Order struct {
	ID            string      `json:"id"`
	Timestamp     time.Time   `json:"timestamp"`
	Side          OrderSide   `json:"side"`
	Type          OrderType   `json:"type"`
	Symbol        string      `json:"symbol"`
	Quantity      float64     `json:"quantity"`
	Price         float64     `json:"price"`          // For limit orders
	Status        OrderStatus `json:"status"`
	FilledPrice   float64     `json:"filled_price"`   // Actual execution price
	FilledQty     float64     `json:"filled_qty"`     // Actual filled quantity
	Fee           float64     `json:"fee"`
	Slippage      float64     `json:"slippage"`       // Difference from expected price
}

// Position represents an open position
type Position struct {
	Symbol        string    `json:"symbol"`
	Side          OrderSide `json:"side"`
	EntryPrice    float64   `json:"entry_price"`
	Quantity      float64   `json:"quantity"`
	CurrentPrice  float64   `json:"current_price"`
	UnrealizedPnL float64   `json:"unrealized_pnl"`
	OpenedAt      time.Time `json:"opened_at"`
}

// Trade represents a completed trade (entry + exit)
type Trade struct {
	ID          string    `json:"id"`
	Symbol      string    `json:"symbol"`
	Side        OrderSide `json:"side"`
	EntryPrice  float64   `json:"entry_price"`
	ExitPrice   float64   `json:"exit_price"`
	Quantity    float64   `json:"quantity"`
	PnL         float64   `json:"pnl"`
	Fee         float64   `json:"fee"`
	NetPnL      float64   `json:"net_pnl"`
	OpenedAt    time.Time `json:"opened_at"`
	ClosedAt    time.Time `json:"closed_at"`
}

// Account represents the trading account state
type Account struct {
	Balance      float64     `json:"balance"`
	Equity       float64     `json:"equity"` // Balance + unrealized PnL
	Positions    []*Position `json:"positions"`
	OpenOrders   []*Order    `json:"open_orders"`
	TradeHistory []*Trade    `json:"trade_history"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

// Signal represents a trading signal from a strategy
type Signal struct {
	Action   SignalAction
	Symbol   string
	Quantity float64
	Reason   string // For logging/debugging
}

// SignalAction represents the action to take
type SignalAction string

const (
	SignalActionBuy  SignalAction = "buy"
	SignalActionSell SignalAction = "sell"
	SignalActionHold SignalAction = "hold"
)
