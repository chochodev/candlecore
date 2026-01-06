# Candlecore

**Candlecore** is a local-first trading engine designed for candle-based strategies. The primary goal is to validate trading logic through backtesting and paper trading before any real capital or cloud infrastructure is involved.

## 🎯 Project Philosophy

- **Local-first**: Runs entirely on your machine
- **No real money**: Paper trading and backtesting only
- **Deterministic**: Reproducible results for testing
- **Testable**: Clean interfaces and separation of concerns
- **Extensible**: Easy to add new strategies without refactoring

## 🏗️ Architecture

```
candlecore/
├── cmd/
│   └── candlecore/          # Main application entry point
│       └── main.go
├── internal/
│   ├── broker/              # Order execution and account management
│   │   └── paper.go         # Paper trading broker implementation
│   ├── config/              # Configuration loading and validation
│   │   └── config.go
│   ├── engine/              # Core trading engine
│   │   ├── engine.go        # Main engine loop and orchestration
│   │   └── types.go         # Data models (Candle, Order, Position, etc.)
│   ├── logger/              # Structured logging
│   │   └── logger.go
│   ├── store/               # State persistence
│   │   └── filestore.go     # File-based state storage
│   └── strategy/            # Trading strategies
│       └── simple_ma.go     # Example: Moving Average crossover
├── config.yaml              # Configuration file
├── go.mod                   # Go module definition
└── README.md                # This file
```

## 🚀 Quick Start

### Prerequisites

- Go 1.25.5 or later
- No external paid dependencies required

### Installation

```bash
# Clone or navigate to the project directory
cd candlecore

# Download dependencies
go mod download

# Build the application
go build -o candlecore ./cmd/candlecore
```

### Running a Backtest

```bash
# Run with default configuration
./candlecore

# Run with custom config file
./candlecore -config custom_config.yaml
```

### Configuration

Edit `config.yaml` to customize:

- **initial_balance**: Starting capital for backtesting
- **fees**: Taker/maker fees to simulate
- **slippage_bps**: Slippage in basis points
- **data_source**: Path to your candle data
- **log_level**: Verbosity (debug, info, warn, error)
- **strategy**: Strategy-specific parameters

### Environment Variables

You can override configuration with environment variables:

```bash
export CANDLECORE_INITIAL_BALANCE=50000
export CANDLECORE_LOG_LEVEL=debug
export CANDLECORE_DATA_SOURCE=data/btc_1h.csv
```

## 📊 Core Concepts

### Data Models

- **Candle**: OHLCV market data
- **Order**: Trading order with execution details
- **Position**: Open position with P&L tracking
- **Trade**: Completed trade (entry + exit)
- **Account**: Account state with balance and equity

### Components

#### Engine
The main orchestrator that:
- Processes candle data sequentially
- Calls strategy for signals
- Executes orders through broker
- Manages state persistence

#### Broker
Handles order execution and position management:
- Simulates market orders
- Applies fees and slippage
- Tracks account balance and equity
- Maintains position state

#### Strategy
User-defined trading logic:
- Implements the `Strategy` interface
- Analyzes candles and account state
- Returns trading signals (buy/sell/hold)
- Can maintain internal state

#### StateStore
Persists engine state for restart safety:
- Saves account state to disk
- Allows resuming from interruptions
- JSON-based for human readability

## 🔧 Creating Custom Strategies

Implement the `Strategy` interface:

```go
type Strategy interface {
    Name() string
    OnCandle(candle Candle, account *Account) Signal
    OnTrade(trade *Trade)
}
```

Example:

```go
package strategy

import "candlecore/internal/engine"

type MyStrategy struct {
    // Your strategy state
}

func (s *MyStrategy) Name() string {
    return "MyStrategy"
}

func (s *MyStrategy) OnCandle(candle engine.Candle, account *engine.Account) engine.Signal {
    // Your trading logic here
    
    return engine.Signal{
        Action:   engine.SignalActionBuy,
        Symbol:   "BTC/USD",
        Quantity: 0.1,
        Reason:   "my custom logic",
    }
}

func (s *MyStrategy) OnTrade(trade *engine.Trade) {
    // Optional: track performance
}
```

Then update `main.go` to use your strategy:

```go
strat := strategy.NewMyStrategy(/* params */)
```

## 📈 Example Output

```
[2026-01-06 00:53:26.000] INFO: Starting Candlecore trading engine
[2026-01-06 00:53:26.000] INFO: Loading candle data source=data/candles.csv
[2026-01-06 00:53:26.000] INFO: Loaded candle data count=100
[2026-01-06 00:53:26.000] INFO: Starting backtesting run candles=100 initial_balance=10000 strategy=SimpleMAStrategy
[2026-01-06 00:53:26.001] INFO: Executing BUY signal symbol=BTC/USD quantity=10 price=101.0 reason=fast MA crossed above slow MA (golden cross)
[2026-01-06 00:53:26.001] INFO: Order executed order_id=abc-123 side=buy symbol=BTC/USD quantity=10 price=101.05 fee=1.01 balance=8898.94
[2026-01-06 00:53:26.015] INFO: Executing SELL signal symbol=BTC/USD quantity=10 price=105.0 reason=fast MA crossed below slow MA (death cross)
[2026-01-06 00:53:26.015] INFO: Position closed symbol=BTC/USD pnl=39.50 net_pnl=38.44 remaining_qty=0
[2026-01-06 00:53:26.020] INFO: Backtest completed final_balance=10038.44 total_pnl=38.44 total_trades=1
```

## 🛠️ Extending Candlecore

### Adding Live Data Sources

1. Create a data loader in `internal/loader/`
2. Implement CSV/JSON/API parsing
3. Update `main.go` to use your loader

### Adding New Order Types

1. Extend `OrderType` enum in `engine/types.go`
2. Implement execution logic in `broker/paper.go`
3. Update strategy signals as needed

### Adding Real Broker Integration

1. Implement the `Broker` interface for your exchange
2. Handle authentication and API calls
3. Swap `PaperBroker` with your implementation in `main.go`

⚠️ **Note**: Always test thoroughly with paper trading before any live trading!

## 🔒 State Management

Candlecore automatically saves state every 10 candles and on shutdown. State is stored in the `.state/` directory as JSON.

To resume from a previous state:
- Ensure `.state/account.json` exists
- Run the engine normally - it will load the state automatically

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/broker/
```

## 📝 Logging

Candlecore uses structured logging with key-value pairs:

```
[timestamp] LEVEL: message key1=value1 key2=value2
```

Log levels:
- **debug**: Detailed information for diagnosing issues
- **info**: General operational messages (default)
- **warn**: Warning messages for potentially harmful situations
- **error**: Error messages for failures

## 🚧 Roadmap (Out of Scope for Now)

- [ ] Real exchange adapters
- [ ] Cloud/VPS deployment
- [ ] Web UI/dashboard
- [ ] AI-powered strategies
- [ ] Live portfolio tracking
- [ ] Multi-timeframe analysis

## 📄 License

This project is for educational and testing purposes only. Use at your own risk.

## 🤝 Contributing

This is a personal project scaffold. Feel free to fork and customize for your needs!

---

**Remember**: Always validate your strategies thoroughly with backtesting and paper trading before considering any real capital deployment.
