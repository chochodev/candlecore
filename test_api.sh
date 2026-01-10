# Test Script for Candlecore Trading Bot

# Start the server (run in separate terminal)
# ./candlecore serve --port 8080

# 1. Check health
curl http://localhost:8080/api/v1/health

# 2. Get available symbols
curl http://localhost:8080/api/v1/symbols

# 3. Get supported timeframes  
curl http://localhost:8080/api/v1/timeframes

# 4. Configure bot
curl -X POST http://localhost:8080/api/v1/bot/configure \
  -H "Content-Type: application/json" \
  -d '{
    "symbol": "bitcoin",
    "timeframe": "1h",
    "strategy": "ma_crossover",
    "replay_mode": true
  }'

# 5. Check bot status
curl http://localhost:8080/api/v1/bot/status

# 6. Start bot
curl -X POST http://localhost:8080/api/v1/bot/start

# 7. Watch status (bot will process candles)
curl http://localhost:8080/api/v1/bot/status

# 8. Get trades
curl http://localhost:8080/api/v1/bot/trades

# 9. Stop bot
curl -X POST http://localhost:8080/api/v1/bot/stop

# WebSocket test (use in browser console or wscat)
# ws://localhost:8080/ws
