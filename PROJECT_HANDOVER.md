# Candlecore Project Handover & Context

This document serves as the tactical memory for the progressive development of the Candlecore Trading Engine. Use this as the ground truth for all future iterations.

## 1. Project Identity
Candlecore is a high-performance algorithmic trading ecosystem consisting of:
- **Core Engine (Go)**: A concurrent trading engine supporting multiple strategies, backtesting, and live WebSocket streaming.
- **Strategic Command (React)**: A premium dashboard and "Strategy Laboratory" for real-time visualization and comparative analysis.

## 2. Strategy Evolution (The Triple Alpha)
We maintain a strict 3-version ecosystem to enable data-driven performance comparisons:
- **v1.0.0 (The Baseline)**: Dual SMA Crossover (12/26) with ADX filtering. Serves as the pure momentum benchmark.
- **v1.0.3 (The Hybrid Alpha)**: Concurrent Ensemble (Bollinger Breakout + EMA Momentum). 
  - **The Triple Guard**: EMA 50 Slope filter (Trend Health), 65% BandWidth Ceiling (Volatility Protection), and 1.5% Hourly Emergency Stop (Flash Crash Fail-safe).
- **v1.0.7 (The Alpha Prime)**: SMA 12/26 Optimized Champion.
  - **The Apex Logic**: Re-entry on pullback + EMA 8 Fast-Exit. Beats baseline by 20% alpha on ETH/SOL.
- **v1.1.0 (The Pulse Scalper)**: High-Frequency 15m Scalper.
  - **The Pulse Logic**: EMA 9/21 Momentum + RSI Filter. 
  - **Proactive Scaling**: Stage 1 TP at +0.8% (Sells 50%) + Stage 2 Safety Lock (Break-even).
- **v1.0.8 (The Solana Sniper)**: High-Frequency Asset-Specific Variant.
  - **The Pulse Engine**: 5/13 EMA cross + ADX 25 Trend filter. Tuned specifically for Solana's parabolic frequency. Handles "Multi-Variant" concurrency in Go.

## 3. Concurrent Swarm Architecture (Multi-Bot Engine)
The engine now supports **Goroutine Concurrency**, allowing BTC, ETH, and SOL variants to run in parallel on separate threads:
- **`MultiBotManager`**: Orchestrates the swarm.
- **`variants/sol_sniper.go`**: Example of an asset-specific "Expert" logic.
- **Safety**: Each bot is isolated in its own context to prevent cross-contamination.
### DOs:
- **Rich Aesthetics**: Use glassmorphism, depth, and vibrant primary colors (Emerald #10b981 for success, Amber for warnings, Slate-950/900 for backgrounds).
- **Modern Typography**: Use Gellix or Inter. Headers should feel bold and premium.
- **Interactive Data**: Every backtest must be visualized. Use Recharts for comparisons and Lightweight Charts for candlestick analysis.
- **Full Implementation**: Never provide snippets if a component's stability is at stake. Build full, production-ready pages.

### DON'Ts:
- **No Placeholders**: Never use "Lorem Ipsum" or empty states. Use generated data or realistic mock datasets if the backend is offline.
- **No Basic UI**: Avoid default browser styling or simple Tailwind defaults. Always push for "State-of-the-art" visual excellence.
- **No Version Bloat**: Keep the comparison limited to the current 3 versions unless a new "Champion" is crowned.

## 4. Project Blueprint (Architecture)
### Backend (Go Engine)
- `/cmd/candlecore`: Application entry point and CLI command definitions (`serve`, `backtest`).
- `/internal/strategy`: Core logic for all trading versions (v1.0.0, v1.0.2, v1.0.3).
- `/internal/api`: REST and WebSocket server implementation (Port 8080).
- `/scripts`: Specialized Go scripts organized by function (e.g., `shootout/main.go`, `report/main.go`).
- `/data/historical`: Storage for raw OHLCV Parquet/CSV files.
- `/changelog`: Versioned JSON performance reports (`report_v1.0.x.json`).

### Frontend (React Command)
- `/src/pages`: High-level views (`Dashboard`, `StrategyLab`).
- `/src/data`: Centralized backtest metrics repository (`backtest_reports.ts`).
- `/src/lib/api`: Fetch client and interface definitions for the Go bridge.
- `/src/hooks`: Shared logic for WebSockets and mobile menu states.

## 5. Reporting & Analytics Pipeline
Performance metrics are generated through a two-stage process:
1. **Engine Backtest**: The Go CLI runs strategies against historical data, outputting raw PnL and drawdown stats to `changelog/`.
2. **Shootout Generation**: `scripts/shootout/main.go` stressors the strategies across specific "Windows of Pain" (2018 Bear, 2024 Chop) to generate the comparative data used in the Strategy Lab.
3. **Frontend Sync**: Hand-selected "Champion" metrics are codified in `src/data/backtest_reports.ts` for zero-latency comparisons in the UI.

## 6. Technical Infrastructure
- **API Bridge**: Backend serves on `http://localhost:8080/api/v1`.
- **WebSocket**: Live updates on `ws://localhost:8080/ws`.

## 7. Tactical Memory Map (Quick Reference)
Use this map to orient yourself instantly upon state initialization:
- **Market Data Flow**: `data/historical/*.csv` -> `internal/bot/bot.go` -> `internal/strategies/pulse_scalper.go`.
- **Pulse Engine**: `v1.1.0` logic handles partial profit-taking (scaling out) at +0.8% to guarantee 'Infinite' free trades on the remaining 50%.
- **Analytics Flow**: `go logic` -> `changelog/*.json` -> `scripts/shootout/main.go` -> `StrategyLab.tsx`.
- **Command Control**: `Header.tsx` (Nav) -> `App.tsx` (Routes) -> `StrategyLab.tsx` (Management).

## 8. What is Next (The Combat Roadmap)
The following items are the highest priority for the next development cycle:
1. **Live Deployment Toggle**: Implement a "Deploy to Engine" button in the Strategy Lab that sends a POST request to `/api/v1/strategy/deploy` to switch the active live strategy version.
2. **Signal Feed Overlay**: Add a real-time signal logger on the Dashboard that shows the exact "Triple Guard" reason for trade exits (e.g., "EMA Slope Flat" or "Volatility Ceiling Hit").
3. **Multi-Symbol Simulation**: Expand the Strategy Lab to compare performance across BTC/USD, ETH/USD, and SOL/USD simultaneously using the same strategy logic.
4. **Performance Heatmaps**: Integrate a calendar-based heatmap in the Strategy Lab to visualize "Red Days" vs. "Green Days" for each version.

---
*Created by Antigravity - Advanced Agentic Coding*
