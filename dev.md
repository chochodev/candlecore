# Candlecore Development Guide

## Available Commands

### Start Server

```bash
./candlecore serve --port 8080
```

### Help

```bash
./candlecore --help
./candlecore serve --help
```

## API Endpoints

### Bot Control

- POST /api/v1/bot/start
- POST /api/v1/bot/stop
- POST /api/v1/bot/configure
- GET /api/v1/bot/status
- GET /api/v1/bot/trades

### Data

- GET /api/v1/symbols
- GET /api/v1/timeframes
- GET /api/v1/health

### WebSocket

- GET /ws

## Data Files

Place CSV files in `data/historical/` with format: `{symbol}_{timeframe}.csv`

Required format:

```
timestamp,open,high,low,close,volume
2024-01-01T00:00:00Z,42000.00,42500.00,41800.00,42300.00,1234567.89
```

Supported timeframes: 1m, 5m, 15m, 1h, 4h, 1d

## Environment Variables

Create `.env` file:

```
COINGECKO_API_KEY=your_key_here
DATA_DIR=data/historical
PORT=8080
```

## Build

```bash
go build -o candlecore.exe ./cmd/candlecore
```
