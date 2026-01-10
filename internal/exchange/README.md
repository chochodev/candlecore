# Exchange Package

## Overview

The exchange package provides a clean interface for ingesting candlestick data from multiple sources. It supports various timeframes and allows easy switching between data providers.

## Features

- ✅ Support for multiple timeframes (1m, 5m, 15m, 1h, 4h, 1d)
- ✅ Local CSV file reading with caching
- ✅ Streaming interface for backtesting/replay
- ✅ Clean `DataProvider` interface for extensibility
- ✅ Thread-safe caching mechanism

## Usage

### Basic Usage

```go
import "candlecore/internal/exchange"

// Create a local file provider
provider := exchange.NewLocalFileProvider("data/historical")

// Get candles for a symbol and timeframe
candles, err := provider.GetCandles("bitcoin", exchange.Timeframe1h, 100)
if err != nil {
    log.Fatal(err)
}

// Stream candles for backtesting
ch, err := provider.StreamCandles("bitcoin", exchange.Timeframe15m)
if err != nil {
    log.Fatal(err)
}

for candle := range ch {
    fmt.Printf("Candle: %+v\n", candle)
}
```

### Supported Timeframes

- `Timeframe1m` - 1 minute
- `Timeframe5m` - 5 minutes
- `Timeframe15m` - 15 minutes
- `Timeframe1h` - 1 hour
- `Timeframe4h` - 4 hours
- `Timeframe1d` - 1 day

### File Naming Convention

CSV files should be named: `{symbol}_{timeframe}.csv`

Examples:

- `bitcoin_1h.csv`
- `ethereum_15m.csv`
- `BTCUSDT_5m.csv`

### CSV Format

```csv
timestamp,open,high,low,close,volume
2024-01-01T00:00:00Z,42000.50,42500.00,41800.00,42300.00,1234567.89
```

## Interface

The `DataProvider` interface allows plugging in different data sources:

```go
type DataProvider interface {
    GetCandles(symbol string, timeframe Timeframe, limit int) ([]Candle, error)
    StreamCandles(symbol string, timeframe Timeframe) (<-chan Candle, error)
    GetSupportedTimeframes() []Timeframe
    GetSupportedSymbols() []string
}
```

## Implementation Details

- **Caching**: Loaded candles are cached in memory to avoid repeated file I/O
- **Thread Safety**: Uses RWMutex for safe concurrent access
- **Error Handling**: Returns descriptive errors for missing files or malformed data
- **Flexibility**: Easy to extend with new providers (e.g., live exchange APIs)

## Next Steps

The exchange package is complete for Task #1. Ready to proceed with:

- Task #2: Bot logic execution with modular strategies
- Task #3: WebSocket streaming
- Task #4: Historical replay
