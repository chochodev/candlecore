### What I Understand You're Achieving
[ ] **Local-First Trading Platform** - You've built a "safe-haven" for strategy development where users can validate logic without risking real capital or needing complex cloud infrastructure.
[ ] **Real-Time Simulation** - The "Replay Mode" is a standout feature—it perfectly mimics the "feeling" of live trading by streaming historical or synthetic candles one-by-one, which helps in debugging how a strategy reacts to every tick.
[ ] **Pluggable Strategy Engine** - The backend is architected to allow easy implementation of new strategies (like the existing MA Crossover and RSI) that can be swapped instantly via the frontend dashboard.
[ ] **Visual-First Monitoring** - The frontend serves as a professional-grade command center, providing immediate feedback on bot decisions, reasoning, and P&L through real-time charts and badges.

### How I Think You Should Move Forward
[ ] **Instant Backtesting Engine** - Create a dedicated "Fast-Backtest" mode. While "Replay" is great for UX, a serious trader needs to run 5 years of data in seconds to see the **Sharpe Ratio**, **Max Drawdown**, and **Profit Factor**.
[ ] **Chart Annotations** - Add "Buy/Sell" markers directly onto the `CandlestickChart`. Seeing the entry and exit points visually relative to the candles is much more intuitive than reading a separate signal card.
[ ] **Live Data Integration** - Transition from "Synthetic/CSV" to "Live WebSocket". Connecting to the Binance Public WebSocket for live candles would be the next logical step toward real-world application.
[ ] **Strategy Composition** - Move beyond single indicators. Allow users to combine strategies (e.g., "Only Buy MA Crossover IF RSI is < 30") through the config or a new "Strategy Builder" UI.
[ ] **Multi-Symbol Dashboard** - Expand the bot to track multiple pairs (BTC, ETH, SOL) simultaneously on one dashboard to help users diversify their "paper portfolio."
[ ] **Performance Persistence** - Currently, progress seems to reset on restart. Implementing a lightweight SQLite database (instead of just `.state/` JSON) would allow for long-term tracking of "Paper Trading" accounts over weeks or months.