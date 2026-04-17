package main

import (
	"candlecore/internal/exchange"
	"candlecore/internal/strategy"
	"fmt"
	"os"
)

func main() {
	// 1. Load data
	dataDir := "data/historical"
	provider := exchange.NewLocalFileProvider(dataDir)
	candles, err := provider.GetCandles("sol", "5m", 0)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// 2. Setup strategy
	strat := strategy.NewFeeShieldPulse()
	df := &strategy.DataFrame{
		Candles: make([]strategy.Candle, 0),
		Indicators: make(map[string][]float64),
	}

	// 3. Find profitable windows
	fmt.Println("Analyzing SOL/5m for Profitable Pulse Windows...")
	fmt.Println("-------------------------------------------------")
	
	// Convert candles
	stratCandles := make([]strategy.Candle, len(candles))
	for i, c := range candles {
		stratCandles[i] = strategy.Candle{
			Timestamp: c.Timestamp,
			Open: c.Open, High: c.High, Low: c.Low, Close: c.Close,
		}
	}
	df.Candles = stratCandles
	strat.PopulateIndicators(df)

	winCount := 0
	for i := 100; i < len(stratCandles); i++ {
		current := stratCandles[i]
		signal := strat.PopulateEntrySignal(df, current)
		
		if signal.Action == "buy" || signal.Action == "sell" {
			// Simulate a trade for the next 20 candles
			entryPrice := current.Close
			maxProfit := 0.0
			
			for j := i + 1; j < i+40 && j < len(stratCandles); j++ {
				var pnl float64
				if signal.Action == "buy" {
					pnl = (stratCandles[j].High - entryPrice) / entryPrice
				} else {
					pnl = (entryPrice - stratCandles[j].Low) / entryPrice
				}
				
				if pnl > maxProfit {
					maxProfit = pnl
				}
			}

			if maxProfit > 0.02 { // Find moves > 2%
				winCount++
				fmt.Printf("WIN %d [%s] - %s at $%.2f\n", winCount, current.Timestamp.Format("2006-01-02 15:04"), signal.Action, current.Close)
				fmt.Printf("   Max Potential: %.2f%%\n", maxProfit*100)
				fmt.Printf("   ADX: %.2f, RSI: %.2f\n", 
					df.Indicators["adx_14"][i], 
					df.Indicators["rsi_14"][i])
				if winCount >= 5 { break }
			}
		}
	}
}
