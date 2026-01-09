# Candlecore Development Guide

## Build Commands

```bash
# Build the application
go build -o candlecore ./cmd/candlecore

# Download dependencies
go mod tidy

# Run tests
go test ./...

# Format code
go fmt ./...
```

## CLI Commands

### API Server

```bash
# Start REST API server (default port 8080)
./candlecore serve

# Custom port
./candlecore serve --port 3000
```

### Data Management

```bash
# Scrape all supported coins (CoinGecko, max 365 days)
./candlecore data scrape

# Scrape specific coin
./candlecore data scrape bitcoin
./candlecore data scrape ethereum

# Update existing data (append new candles)
./candlecore data update

# Update specific coin
./candlecore data update bitcoin

# List available data
./candlecore data list
```

## REST API Endpoints

### Base URL

`http://localhost:8080/api/v1`

### Endpoints

#### GET /health

Health check endpoint.

**Response:**

```json
{
  "status": "healthy",
  "version": "1.0.0",
  "time": "2026-01-09T13:00:00Z"
}
```

#### GET /data

List all available data files.

**Response:**

```json
{
  "files": [
    {
      "coin_id": "bitcoin",
      "interval": "daily",
      "total_candles": 92,
      "first_date": "2025-01-08T00:00:00Z",
      "last_date": "2026-01-07T00:00:00Z",
      "file_size_kb": 8.95,
      "file_path": "data/historical/bitcoin_daily.csv"
    }
  ],
  "total": 1
}
```

#### GET /data/:coin/:interval

Get candle data for a specific coin.

**Parameters:**

- `coin`: Coin ID (e.g., bitcoin, ethereum)
- `interval`: Timeframe (e.g., daily, 1h, 15m)
- `limit`: (optional) Max number of candles to return

**Response:**

```json
{
  "coin": "bitcoin",
  "interval": "daily",
  "total": 92,
  "candles": [
    {
      "timestamp": "2025-01-08T00:00:00Z",
      "open": 95000.5,
      "high": 96000.0,
      "low": 94500.0,
      "close": 95500.0,
      "volume": 1234567.89
    }
  ]
}
```

#### POST /backtest

Run a backtest with specified parameters.

**Request:**

```json
{
  "coin_id": "bitcoin",
  "interval": "daily",
  "strategy": "simple_ma",
  "initial_balance": 10000,
  "fast_period": 10,
  "slow_period": 30,
  "position_size": 1000
}
```

**Response:**

```json
{
  "message": "Backtest queued",
  "id": "backtest-123",
  "status": "pending"
}
```

#### GET /backtest/results/:id

Get backtest results by ID.

**Response:**

```json
{
  "id": "backtest-123",
  "status": "completed",
  "results": {
    "initial_balance": 10000.0,
    "final_balance": 12500.0,
    "total_pnl": 2500.0,
    "total_trades": 15,
    "win_rate": 0.67
  }
}
```

## Data Storage

### Historical Data Location

- Default: `data/historical/`
- Format: CSV files per coin and timeframe
- Files: `{coin}_{interval}.csv`

### File Structure

```
data/
└── historical/
    ├── bitcoin_5m.csv
    ├── bitcoin_15m.csv
    ├── bitcoin_1h.csv
    ├── bitcoin_4h.csv
    ├── bitcoin_1d.csv
    ├── ethereum_5m.csv
    ├── ethereum_15m.csv
    └── ethereum_1d.csv
```

### CSV Format

```csv
timestamp,open,high,low,close,volume
2026-01-07T00:00:00Z,95000.50,96000.00,94500.00,95500.00,1234567.89
```

## Configuration

### Environment Variables

```env
# Database
CANDLECORE_DB_ENABLED=false
CANDLECORE_DB_PASSWORD=your_password

# Live Data
CANDLECORE_LIVE_DATA_ENABLED=false
CANDLECORE_LIVE_DATA_SYMBOL=BTCUSDT
CANDLECORE_LIVE_DATA_INITIAL_FETCH=90

# Strategy
CANDLECORE_STRATEGY_FAST_PERIOD=5
CANDLECORE_STRATEGY_SLOW_PERIOD=20

# Logging
CANDLECORE_LOG_LEVEL=error
```

## Development Workflow

### Frontend Integration

```bash
# Start API server
./candlecore serve

# In your frontend project (example with fetch)
fetch('http://localhost:8080/api/v1/data')
  .then(res => res.json())
  .then(data => console.log(data));
```

### CORS

CORS is enabled by default for all origins. The API accepts:

- GET, POST, PUT, DELETE, OPTIONS methods
- Content-Type and Authorization headers

## Testing with API

```bash
# 1. Start the server
./candlecore serve

# 2. Test health endpoint
curl http://localhost:8080/api/v1/health

# 3. List available data
curl http://localhost:8080/api/v1/data

# 4. Get Bitcoin candles
curl http://localhost:8080/api/v1/data/bitcoin/daily

# 5. Run backtest
curl -X POST http://localhost:8080/api/v1/backtest \
  -H "Content-Type: application/json" \
  -d '{
    "coin_id": "bitcoin",
    "interval": "daily",
    "strategy": "simple_ma",
    "initial_balance": 10000
  }'
```

## Project Structure

```
candlecore/
├── cmd/candlecore/          # Main entry point
├── internal/
│   ├── api/                 # REST API server
│   ├── broker/              # Paper trading simulation
│   ├── cmd/                 # CLI commands
│   ├── config/              # Configuration management
│   ├── engine/              # Trading engine core
│   ├── fetcher/             # API clients (CoinGecko)
│   ├── loader/              # CSV data loading
│   ├── logger/              # Logging utilities
│   ├── scraper/             # Historical data scraper
│   ├── store/               # State persistence
│   ├── strategy/            # Trading strategies
│   └── ui/                  # Terminal UI components
├── data/
│   └── historical/          # Downloaded market data
├── database/                # PostgreSQL schema
├── .state/                  # Runtime state files
├── config.yaml              # Configuration
├── .env                     # Environment variables
└── dev.md                   # This file
```

## Dependencies

```
github.com/spf13/cobra          - CLI framework
github.com/gin-gonic/gin        - HTTP web framework
github.com/fatih/color          - Colored output
github.com/joho/godotenv        - .env loading
github.com/lib/pq               - PostgreSQL driver
```

## Next Development Priorities

1. Implement full backtest execution in API
2. Add WebSocket support for real-time updates
3. Implement trade history endpoints
4. Add performance analytics endpoints (Sharpe ratio, drawdown)
5. Create multi-strategy backtesting support
6. Add authentication/API keys
7. Implement backtest result caching
8. Export results to JSON/CSV via API
