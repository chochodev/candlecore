# Quantitative Research Playbook: Finding Your Edge

This guide outlines the professional research pipeline for discovering high-expectancy trading strategies on any asset using the Candlecore Research Engine.

## Step 1: Data Backfilling (1-Year Minimum)
Strategy validation fails without enough "Lookback." You need at least 30,000 candles (approx. 1 year of 15m data).
1.  Update the `scratch/backfill.ps1` script with your target symbol (e.g., BTCUSDT).
2.  Run the script to download 12 months of Binance monthly ZIPs.
3.  Run `scratch/process_backfill.go` to merge and format the data into `data/historical/<symbol>_15m.csv`.

## Step 2: Multiscale Extraction
The engine aligns two timeframes to prevent "Micro-Noise" trading:
- **H1 (HTF):** Sets the global trend context (EMA 50/200).
- **M15 (Execution):** Captures local price action signatures.
*Tool:* `internal/research/engine.go -> GenerateDataset(symbol)`

## Step 3: Four-Dimensional Clustering
We categorize every candle into a unique "Market Context Signature" using 4 pillars:
1.  **Trend:** H1 bullish/bearish alignment.
2.  **Volume:** Relative volume v. 20-period average (Institutional vs. Retail).
3.  **BOS (Structure):** Local high/low breakouts.
4.  **Pullback:** Proximity to the EMA mean (Deep/Moderate/Extreme).

## Step 4: Expectancy Mining
We group these signatures and calculate the **Expectancy (Profit per Risk Dollar)**.
*Tool:* `cmd/candlecore discover`
> [!IMPORTANT]
> Reject any cluster with fewer than 30-50 samples in the training set.

## Step 5: Walk-Forward Validation (Stress Testing)
We split the data into **70% Train** and **30% Test**. 
1.  Select the top cluster from the Train set.
2.  Simulate it on the Test set (unseen data).
3.  **Drift Check:** If the Expectancy drops by more than 30% (e.g., from 0.40 to 0.10), the strategy is likely overfit and will fail live. 
*Tool:* `internal/research/validator.go`

## Step 6: Codification
Once a cluster passes validation (Positive Expectancy + Low Drift), hardcode its parameters into a new Strategy file.
Example (Sovereign Sniper):
- Trend: Bearish
- Volume: High
- Pullback: Deep
- Structure break: None

## Summary of Success Metrics
- **Expectancy:** > 0.10 is tradable. > 0.20 is elite.
- **Sample Size:** > 300 total samples across 1 year for high confidence.
- **Drift:** < 30% to ensure regime-stability.
