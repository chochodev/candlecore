# Candle Data Directory

This directory is for storing your candle (OHLCV) data files.

## CSV Format

Place your CSV files here with the following format:

```csv
timestamp,open,high,low,close,volume
2024-01-01T00:00:00Z,42000.00,42500.00,41800.00,42300.00,1250.50
2024-01-01T01:00:00Z,42300.00,42800.00,42100.00,42600.00,1380.25
2024-01-01T02:00:00Z,42600.00,43000.00,42400.00,42900.00,1520.75
```

### Field Descriptions

- **timestamp**: ISO 8601 format (RFC3339), e.g., `2024-01-01T00:00:00Z`
- **open**: Opening price for the candle
- **high**: Highest price during the candle period
- **low**: Lowest price during the candle period
- **close**: Closing price for the candle
- **volume**: Trading volume for the candle period

### Data Requirements

1. Header row must be present
2. All timestamps must be in RFC3339 format
3. All prices and volume must be valid numbers
4. Candle validation:
   - `high` must be >= `low`
   - `open` and `close` must be within `[low, high]` range

## Usage

To use CSV data in your backtest, update `config.yaml`:

```yaml
data_source: 'data/your_candles.csv'
```

Or set environment variable:

```bash
$env:CANDLECORE_DATA_SOURCE="data/your_candles.csv"
```

Then update `main.go` to use the CSV loader:

```go
import "candlecore/internal/loader"

// In main():
csvLoader := loader.NewCSVLoader(cfg.DataSource)
candles, err := csvLoader.Load()
if err != nil {
    log.Error("Failed to load candles", "error", err)
    os.Exit(1)
}
```

## Data Sources

You can obtain historical candle data from:

1. **Exchange APIs**:

   - Binance: https://api.binance.com/api/v3/klines
   - Coinbase: https://api.exchange.coinbase.com/products/{id}/candles
   - Kraken: https://api.kraken.com/0/public/OHLC

2. **Data Providers**:

   - CryptoCompare
   - CoinGecko
   - Yahoo Finance (for traditional markets)

3. **Your Own Recording**:
   - Record live data yourself for paper trading

## Example: Fetch from Binance

```bash
# Fetch 1-hour BTC/USDT candles (last 100)
curl "https://api.binance.com/api/v3/klines?symbol=BTCUSDT&interval=1h&limit=100" > btc_raw.json

# Convert to CSV format (you'll need to write a converter or use jq/python)
```

## Notes

- This directory is gitignored to avoid committing large data files
- Always validate your data before running backtests
- Consider data quality and outliers in your analysis
