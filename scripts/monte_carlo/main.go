package main

import (
	"fmt"
	"math/rand"
	"os/exec"
	"time"
)

func main() {
	symbols := []string{"btc", "eth", "sol"}
	iterations := 10
	days := 5

	fmt.Printf("🎲 Starting Monte Carlo: 10 random %d-day sessions for each symbol\n", days)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Set seed
	rand.Seed(time.Now().UnixNano())

	// Data ranges (CDD 1h)
	// BTC/ETH: ~2017-08 to 2026-03
	// SOL: ~2020-08 to 2026-03
	
	results := make(map[string][]float64)

	for _, symbol := range symbols {
		fmt.Printf("\n🚀 Testing %s:\n", symbol)
		startYear := 2018
		if symbol == "sol" {
			startYear = 2021
		}

		for i := 0; i < iterations; i++ {
			// Pick a random date
			randomDate := pickRandomDate(startYear, 2025)
			endBotDate := randomDate.AddDate(0, 0, days)
			
			startStr := randomDate.Format("2006-01-02")
			endStr := endBotDate.Format("2006-01-02")

			fmt.Printf("   [%02d] %s to %s: ", i+1, startStr, endStr)
			
			// Run backtest command
			cmd := exec.Command("go", "run", "cmd/candlecore/main.go", "backtest", 
				"-s", symbol, 
				"-t", "1h", 
				"--start", startStr, 
				"--end", endStr, 
				"-b", "10.0")
			
			output, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Printf("❌ Error: %v\n", err)
				continue
			}

			// Parse result (PnL) from output
			pnl := extractPnL(string(output))
			results[symbol] = append(results[symbol], pnl)
			fmt.Printf("%.2f%%\n", pnl)
		}
	}

	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 30-Session Summary (Risking $10 each):")
	for symbol, pnls := range results {
		avg := 0.0
		wins := 0
		for _, v := range pnls {
			avg += v
			if v > 0 { wins++ }
		}
		avg /= float64(len(pnls))
		fmt.Printf("   %s: Avg PnL: %+.2f%% | Success Rate: %d/%d\n", symbol, avg, wins, iterations)
	}
}

func pickRandomDate(startYear, endYear int) time.Time {
	min := time.Date(startYear, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	max := time.Date(endYear, 12, 1, 0, 0, 0, 0, time.UTC).Unix()
	delta := max - min
	sec := rand.Int63n(delta) + min
	return time.Unix(sec, 0).UTC()
}

func extractPnL(output string) float64 {
	// Crude but fast: look for PnL string
	// Format: "Total PnL: $0.04 (0.40%)"
	var pnl float64
	fmt.Sscanf(grep(output, "Total PnL"), "📈 Total PnL: $%f (%f%%)", &pnl, &pnl)
	return pnl
}

func grep(input, search string) string {
	// Helper to find a line containing search string
	lines := splitLines(input)
	for _, line := range lines {
		if contains(line, search) {
			return line
		}
	}
	return ""
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	return lines
}

func contains(s, search string) bool {
	return (len(s) >= len(search)) && (len(search) == 0 || find(s, search) != -1)
}

func find(s, search string) int {
	for i := 0; i <= len(s)-len(search); i++ {
		if s[i:i+len(search)] == search {
			return i
		}
	}
	return -1
}
