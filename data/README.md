# Historical Data Acquisition

This directory contains the historical trading data used to validate the Candlecore Trading Engine strategies (v1.0.0, v1.0.2, and v1.0.3).

## Data Source
All historical data was retrieved from the **Binance Public REST API** (`/api/v3/klines`). 

## Dataset Specifications
- **Symbol**: BTC/USDT
- **Interval**: 1h (1-Hour Candles)
- **Fields**: Open, High, Low, Close, Volume (OHLCV)

## Acquisition Methodology
The data was downloaded using the internal `scripts/fetch_api.go` utility. This script performs the following:
1. Connects to the Binance API.
2. Iterates through the requested timeframes to ensure no gaps in the candle history.
3. Converts the raw JSON response into the engine's internal representation.
4. Saves the results to this directory for use in the backtest engine.

## Test Window Selection
For the "Strategy Laboratory" shootout, we deliberately selected diverse market regimes to ensure the **Triple Guard** logic was robust across different conditions:
- **Bull Regime**: Strong uptrends to measure momentum capture.
- **Bear Regime**: Sharp declines to validate the emergency exit and drawdown protection.
- **Chop Regime**: Sideways consolidation to test the volatility filters and prevent over-trading.

---
*Reference: Generated based on the conversational context of the Candlecore development session.*
