package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"time"

	"candlecore/internal/fetcher"
)

func main() {
	symbols := []string{"BTCUSDT", "ETHUSDT", "SOLUSDT"}
	interval := "15m"
	limit := 1000 // Binance max per request

	bf := fetcher.NewBinanceFetcher()
	ctx := context.Background()

	for _, symbol := range symbols {
		fmt.Printf("📥 Fetching %d candles for %s (%s)... ", limit, symbol, interval)
		candles, err := bf.FetchCandles(ctx, symbol, interval, limit)
		if err != nil {
			log.Fatalf("\n❌ Error: %v", err)
		}
		fmt.Printf("✅ Got %d candles\n", len(candles))

		// Save to CSV
		filename := fmt.Sprintf("data/historical/%s_%s.csv", symbol, interval)
		if symbol == "BTCUSDT" { filename = "data/historical/btc_15m.csv" }
		if symbol == "ETHUSDT" { filename = "data/historical/eth_15m.csv" }
		if symbol == "SOLUSDT" { filename = "data/historical/sol_15m.csv" }

		file, err := os.Create(filename)
		if err != nil {
			log.Fatalf("❌ CSV Create Error: %v", err)
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		defer writer.Flush()

		writer.Write([]string{"timestamp", "open", "high", "low", "close", "volume"})

		for _, c := range candles {
			writer.Write([]string{
				c.Timestamp.Format(time.RFC3339),
				fmt.Sprintf("%.8f", c.Open),
				fmt.Sprintf("%.8f", c.High),
				fmt.Sprintf("%.8f", c.Low),
				fmt.Sprintf("%.8f", c.Close),
				fmt.Sprintf("%.8f", c.Volume),
			})
		}
		fmt.Printf("💾 Saved to %s\n", filename)
	}
	fmt.Println("\n✨ Data Fetch Complete!")
}
