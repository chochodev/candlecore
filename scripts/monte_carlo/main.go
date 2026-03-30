package main

import (
	"fmt"
	"math/rand"
	"os/exec"
	"time"
)

func main() {
	symbols := []string{"btc", "eth", "sol"}
	strategies := []string{"original", "ensemble", "fusion", "alpha_prime", "sol_sniper"}
	iterations := 20
	days := 7

	fmt.Printf("🎲 Starting Multi-Strategy Monte Carlo: %d iterations of %d days each\n", iterations, days)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Store results: map[strategy]map[symbol][]PnL
	allResults := make(map[string]map[string][]float64)
	for _, s := range strategies {
		allResults[s] = make(map[string][]float64)
	}

	// Pick windows ahead of time so all strategies face the same challenge
	rand.Seed(42) // Constant seed for fair comparison
	type window struct { symbol string; start time.Time; end time.Time }
	windows := make([]window, 0)
	for _, symbol := range symbols {
		startYear := 2018
		if symbol == "sol" { startYear = 2021 }
		for i := 0; i < iterations; i++ {
			start := pickRandomDate(startYear, 2025)
			windows = append(windows, window{symbol, start, start.AddDate(0, 0, days)})
		}
	}

	for _, strategy := range strategies {
		fmt.Printf("\n🚀 Testing Strategy: %s\n", strategy)
		for i, w := range windows {
			startStr := w.start.Format("2006-01-02")
			endStr := w.end.Format("2006-01-02")

			if i % iterations == 0 { fmt.Printf("   [%s] Window Analysis Start\n", w.symbol) }
			
			cmd := exec.Command("go", "run", "cmd/candlecore/main.go", "backtest", 
				"-s", w.symbol, "-t", "1h", "--start", startStr, "--end", endStr, 
				"-b", "10.0", "--strategy", strategy)
			
			output, _ := cmd.CombinedOutput()
			outputStr := string(output)
			pnlLine := grep(outputStr, "Total PnL")
			pnl := 0.0
			if pnlLine != "" {
				parts := split(pnlLine, "(")
				if len(parts) > 1 {
					subParts := split(parts[1], "%")
					if len(subParts) > 0 { fmt.Sscanf(subParts[0], "%f", &pnl) }
				}
			}
			allResults[strategy][w.symbol] = append(allResults[strategy][w.symbol], pnl)
		}
	}

	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🏆 FINAL MONTE CARLO SHOOTOUT (240 SESSIONS)")
	fmt.Println("Strategy       | Symbol | Avg PnL% | Success Rate")
	fmt.Println("━━━━━━━━━━━━━━━|━━━━━━━━|━━━━━━━━━━|━━━━━━━━━━━━━")
	for _, s := range strategies {
		for _, sym := range symbols {
			pnls := allResults[s][sym]
			avg, wins := 0.0, 0
			for _, v := range pnls {
				avg += v
				if v > 0 { wins++ }
			}
			avg /= float64(len(pnls))
			fmt.Printf("%-15s| %-7s| %-+8.2f%% | %d/%d\n", s, sym, avg, wins, iterations)
		}
		fmt.Println("---------------|--------|----------|-------------")
	}
}

func pickRandomDate(startYear, endYear int) time.Time {
	min := time.Date(startYear, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	max := time.Date(endYear, 12, 1, 0, 0, 0, 0, time.UTC).Unix()
	delta := max - min
	sec := rand.Int63n(delta) + min
	return time.Unix(sec, 0).UTC()
}

func grep(input, search string) string {
	lines := splitLines(input)
	for _, line := range lines {
		if contains(line, search) { return line }
	}
	return ""
}

func split(s, sep string) []string {
	var parts []string
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			parts = append(parts, s[start:i])
			start = i + len(sep)
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' || s[i] == '\r' {
			if start < i { lines = append(lines, s[start:i]) }
			start = i + 1
		}
	}
	if start < len(s) { lines = append(lines, s[start:]) }
	return lines
}

func contains(s, search string) bool {
	return (len(s) >= len(search)) && (len(search) == 0 || find(s, search) != -1)
}

func find(s, search string) int {
	for i := 0; i <= len(s)-len(search); i++ {
		if s[i:i+len(search)] == search { return i }
	}
	return -1
}
