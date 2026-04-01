package main

import (
	"candlecore/internal/bot"
	"candlecore/internal/exchange"
	"candlecore/internal/strategies"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fatih/color"
)

func main() {
	symbol := "SOLUSDT"
	timeframe := exchange.Timeframe("5m") // Indicators calculated on 5m, but evaluated on 1s ticks
	balance := 10.0

	fmt.Println(color.CyanString("🚀 INITIALIZING LIVE PERFORMANCE REVIEW (1s Update Stream)"))
	fmt.Println(color.CyanString("📍 Symbol: %s | Timeframe: %s | Account: $%.2f", symbol, timeframe, balance))
	fmt.Println("----------------------------------------------------------------------")

	// 1. Warm-up: Fetch recent history
	dataDir := "data/historical"
	provider := exchange.NewLocalFileProvider(dataDir)
	
	// Try to get existing history for indicators
	history, err := provider.GetCandles("SOL", timeframe, 200)
	if err != nil {
		log.Fatalf("Critical: Could not load warm-up data: %v", err)
	}
	fmt.Printf("✅ Pulse Engine Warmed Up with %d candles.\n", len(history))

	// 2. Initialize Bot with Pulse Strategy
	strat, _ := strategies.GetStrategy("pulse_scalper")
	b := bot.NewBot(strat, provider, bot.Config{
		Symbol: "SOL", Timeframe: timeframe, InitialBalance: balance, PositionSize: 10,
	})

	// 3. Connect to Live 1s Feed
	ws := exchange.NewBinanceWSClient()
	candleChan := make(chan exchange.Candle)
	if err := ws.Stream1s("solusdt", candleChan); err != nil {
		log.Fatalf("WS Connection Failed: %v", err)
	}

	fmt.Println(color.GreenString("📡 LIVE CONNECTED. Monitoring every 1-second price pulse..."))
	fmt.Println("----------------------------------------------------------------------")

	// 4. Processing Loop
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	green := color.New(color.FgHiGreen).SprintFunc()
	yellow := color.New(color.FgHiYellow).SprintFunc()
	white := color.New(color.FgWhite).SprintFunc()
	
	ticker := time.NewTicker(2 * time.Second) // Periodic status update

	for {
		select {
		case candle := <-candleChan:
			// Process the 1s tick
			decision, _ := b.ProcessCandle(candle)
			
			// Custom log only for the Fee Shield actions
			if b.GetPosition() != nil {
				pnl := b.GetTotalPnL()
				pnlPct := (pnl / balance) * 100
				
				if decision.Signal == "sell" && decision.Quantity > 0 {
					fmt.Printf("%s %s @ $%.2f | Reasoning: %s\n", 
						green("[🛡️  SHIELD]"), 
						green("PARTIAL EXIT"), 
						candle.Close, 
						decision.Reasoning)
				} else if decision.Signal == "sell" {
					fmt.Printf("%s %s @ $%.2f | Balance: $%.2f | PnL: %.2f%%\n", 
						yellow("[🏁 EXIT]"), 
						yellow("POSITION CLOSED"), 
						candle.Close, 
						b.GetBalance(), 
						pnlPct)
				}
			} else if decision.Signal == "buy" {
				fmt.Printf("%s %s @ $%.2f | Reasoning: %s\n", 
					green("[🚀 ENTRY]"), 
					green("POSITION OPEN"), 
					candle.Close, 
					decision.Reasoning)
			}

		case <-ticker.C:
			// Regular heartbeat for the user
			if b.GetPosition() == nil {
				fmt.Printf("%s Active. Waiting for Pulse...\r", white("[👁️  SCAN]"))
			} else {
				pnl := b.GetTotalPnL()
				fmt.Printf("%s Active Position. Unrealized PnL: $%.4f\r", white("[📉 LIVE]"), pnl)
			}

		case <-interrupt:
			fmt.Println("\n\n🛑 Review Terminated by User. Cleaning up...")
			return
		}
	}
}
