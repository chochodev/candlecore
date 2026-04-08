package main

import (
	"candlecore/internal/bot"
	"candlecore/internal/engine"
	"candlecore/internal/exchange"
	"candlecore/internal/strategies"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type VersionReport struct {
	Version     string                  `json:"version"`
	Timestamp   time.Time               `json:"timestamp"`
	Description string                  `json:"description"`
	Benchmarks  []*engine.BacktestReport `json:"benchmarks"`
}

func main() {
	dataDir := "data/historical"
	changelogDir := "changelog"
	version := "1.0.8"
	strategyName := "ma_crossover"

	if err := os.MkdirAll(changelogDir, 0755); err != nil {
		log.Fatal(err)
	}

	files, err := os.ReadDir(dataDir)
	if err != nil {
		log.Fatal(err)
	}

	report := &VersionReport{
		Version:     version,
		Timestamp:   time.Now(),
		Description: "MA Crossover (v1.0.8): Symmetric long/short crossover engine with mirrored bearish entries and buy-to-cover exits for short-side parity.",
		Benchmarks:  make([]*engine.BacktestReport, 0),
	}

	provider := exchange.NewLocalFileProvider(dataDir)

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".csv") {
			continue
		}

		// Simple parsing: btc_1h.csv -> symbol=btc, tf=1h
		parts := strings.Split(strings.TrimSuffix(file.Name(), ".csv"), "_")
		if len(parts) < 2 {
			continue
		}
		
		symbol := parts[0]
		tfStr := parts[1]
		if len(parts) > 2 {
			// handle binance_btc_1h.csv
			symbol = parts[1]
			tfStr = parts[2]
		}
		
		timeframe := exchange.Timeframe(tfStr)

		fmt.Printf("Running benchmark: %s/%s...\n", symbol, tfStr)

		candles, err := provider.GetCandles(symbol, timeframe, 0)
		if err != nil {
			fmt.Printf("  Skipping %s: %v\n", file.Name(), err)
			continue
		}

		strategy, err := strategies.GetStrategy(strategyName)
		if err != nil {
			log.Fatal(err)
		}

		b := bot.NewBot(strategy, provider, bot.Config{
			Symbol:         symbol,
			Timeframe:      timeframe,
			InitialBalance: 10000.0,
			PositionSize:   10,
		})

		start := time.Now()
		if err := b.RunBacktest(candles); err != nil {
			fmt.Printf("  Error in backtest: %v\n", err)
			continue
		}
		duration := time.Since(start)

		reportMetrics := engine.CalculateMetrics(10000.0, toEngineTrades(b.GetTrades()), b.GetBalanceHistory())
		reportMetrics.StrategyName = strategyName
		reportMetrics.Symbol = symbol
		reportMetrics.Timeframe = tfStr
		reportMetrics.Duration = duration
		
		if len(candles) > 0 {
			reportMetrics.StartDate = candles[0].Timestamp
			reportMetrics.EndDate = candles[len(candles)-1].Timestamp
		}

		report.Benchmarks = append(report.Benchmarks, reportMetrics)
	}

	output, _ := json.MarshalIndent(report, "", "  ")
	outputPath := filepath.Join(changelogDir, fmt.Sprintf("report_v%s.json", version))
	if err := os.WriteFile(outputPath, output, 0644); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n✅ Full scale report generated in %s\n", outputPath)
}

func toEngineTrades(botTrades []bot.Position) []engine.Trade {
	res := make([]engine.Trade, len(botTrades))
	for i, t := range botTrades {
		res[i] = engine.Trade{PnL: t.RealizedPnL}
	}
	return res
}
