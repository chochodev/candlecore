package main

import (
	"candlecore/internal/bot"
	"candlecore/internal/exchange"
	"candlecore/internal/strategies"
	"candlecore/internal/strategy"
	_ "candlecore/internal/strategy"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

type Window struct {
	Name  string
	Start time.Time
	End   time.Time
}

func main() {
	windows := []Window{
		{"Bull Run (Blow-off Top)", time.Date(2017, 12, 1, 0, 0, 0, 0, time.UTC), time.Date(2017, 12, 15, 23, 59, 0, 0, time.UTC)},
		{"Bear Crash (Post-Peak)", time.Date(2018, 1, 15, 0, 0, 0, 0, time.UTC), time.Date(2018, 1, 31, 23, 59, 0, 0, time.UTC)},
		{"Choppy Range (2024)", time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 5, 14, 23, 59, 0, 0, time.UTC)},
	}

	strategies_to_test := []struct {
		tag  string
		name string
	}{
		{"original", "v1.0.0"},
		{"ma_crossover", "v1.0.2"},
		{"ensemble", "v1.0.3"},
	}

	dataDir := "data/historical"
	candles, _ := loadCandles(filepath.Join(dataDir, "btc_1h.csv"))

	fmt.Printf("%-25s | %-10s | %-10s | %-10s\n", "Window", "Strategy", "PnL %", "Trades")
	fmt.Println("--------------------------------------------------------------------------------")

	for _, w := range windows {
		windowCandles := filterCandles(candles, w.Start, w.End)
		for _, s := range strategies_to_test {
			strat, _ := strategy.Get(s.tag)
			adapter := strategies.NewStrategyAdapter(strat)
			b := bot.NewBot(adapter, nil, bot.Config{Symbol: "btc", Timeframe: "1h", InitialBalance: 10000.0})
			b.RunBacktest(windowCandles)
			
			trades := b.GetTrades()
			pnl := (b.GetBalance() - 10000.0) / 100.0
			fmt.Printf("%-25s | %-10s | %-10.2f | %-10d\n", w.Name, s.name, pnl, len(trades))
		}
		fmt.Println("--------------------------------------------------------------------------------")
	}
}

func filterCandles(all []exchange.Candle, start, end time.Time) []exchange.Candle {
	var filtered []exchange.Candle
	firstIdx := -1
	for i, c := range all {
		if (c.Timestamp.Equal(start) || c.Timestamp.After(start)) && (c.Timestamp.Before(end) || c.Timestamp.Equal(end)) {
			if firstIdx == -1 {
				firstIdx = i
				warmIdx := i - 200
				if warmIdx < 0 { warmIdx = 0 }
				filtered = append(filtered, all[warmIdx:i]...)
			}
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func loadCandles(path string) ([]exchange.Candle, error) {
	file, _ := os.Open(path)
	defer file.Close()
	reader := csv.NewReader(file)
	records, _ := reader.ReadAll()

	var candles []exchange.Candle
	for i, record := range records {
		if i == 0 { continue }
		t, _ := time.Parse(time.RFC3339, record[0])
		open, _ := strconv.ParseFloat(record[1], 64)
		high, _ := strconv.ParseFloat(record[2], 64)
		low, _ := strconv.ParseFloat(record[3], 64)
		close, _ := strconv.ParseFloat(record[4], 64)
		vol, _ := strconv.ParseFloat(record[5], 64)

		candles = append(candles, exchange.Candle{
			Timestamp: t.UTC(),
			Open:      open, High: high, Low: low, Close: close, Volume: vol,
		})
	}
	sort.Slice(candles, func(i, j int) bool { return candles[i].Timestamp.Before(candles[j].Timestamp) })
	return candles, nil
}
