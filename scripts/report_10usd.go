package main

import (
	"candlecore/internal/bot"
	"candlecore/internal/exchange"
	"candlecore/internal/strategies"
	"fmt"
)

func main() {
	symbol := "SOL"
	timeframe := exchange.Timeframe("5m")
	initialBalance := 10.0
	dataDir := "data/historical"

	provider := exchange.NewLocalFileProvider(dataDir)
	allCandles, _ := provider.GetCandles(symbol, timeframe, 0)
	
	strat, _ := strategies.GetStrategy("pulse_scalper")

	runReport := func(days int) (float64, int, float64) {
		b := bot.NewBot(strat, provider, bot.Config{
			Symbol: symbol, Timeframe: timeframe, InitialBalance: initialBalance, PositionSize: 10,
		})
		
		numCandles := days * 12 * 24 // 12 candles per hour, 24 hours
		if numCandles > len(allCandles) { numCandles = len(allCandles) }
		testCandles := allCandles[len(allCandles)-numCandles:]
		b.RunBacktest(testCandles)
		
		return b.GetBalance(), len(b.GetTrades()), (b.GetBalance() - initialBalance) / initialBalance * 100
	}

	fmt.Println("🏆 TURBO PULSE v1.3.0 PERFORMANCE REPORT ($10 Balance)")
	fmt.Println("---------------------------------------------------------")
	
	bal1d, trades1d, pnl1d := runReport(1)
	fmt.Printf("📆 1 Day:    Balance: $%.2f | Trades: %d | PnL: %.2f%%\n", bal1d, trades1d, pnl1d)
	
	bal1w, trades1w, pnl1w := runReport(7)
	fmt.Printf("📅 1 Week:   Balance: $%.2f | Trades: %d | PnL: %.2f%%\n", bal1w, trades1w, pnl1w)
	
	bal1m, trades1m, pnl1m := runReport(30)
	fmt.Printf("🗓️ 1 Month:  Balance: $%.2f | Trades: %d | PnL: %.2f%%\n", bal1m, trades1m, pnl1m)
	
	fmt.Println("---------------------------------------------------------")
	fmt.Println("⚡ ALL FEES (0.1% ENTRY/EXIT) AND SLIPPAGE (0.05%) DEDUCTED")
}
