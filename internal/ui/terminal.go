package ui

import (
	"fmt"
	"strings"
	"time"

	"candlecore/internal/engine"

	"github.com/fatih/color"
)

var (
	green   = color.New(color.FgGreen).SprintFunc()
	red     = color.New(color.FgRed).SprintFunc()
	yellow  = color.New(color.FgYellow).SprintFunc()
	cyan    = color.New(color.FgCyan).SprintFunc()
	magenta = color.New(color.FgMagenta).SprintFunc()
	bold    = color.New(color.Bold).SprintFunc()
)

// PrintBanner prints application banner
func PrintBanner() {
	banner := `
╔═══════════════════════════════════════════════════════════╗
║                                                           ║
║   ██████╗ █████╗ ███╗   ██╗██████╗ ██╗     ███████╗      ║
║  ██╔════╝██╔══██╗████╗  ██║██╔══██╗██║     ██╔════╝      ║
║  ██║     ███████║██╔██╗ ██║██║  ██║██║     █████╗        ║
║  ██║     ██╔══██║██║╚██╗██║██║  ██║██║     ██╔══╝        ║
║  ╚██████╗██║  ██║██║ ╚████║██████╔╝███████╗███████╗      ║
║   ╚═════╝╚═╝  ╚═╝╚═╝  ╚═══╝╚═════╝ ╚══════╝╚══════╝      ║
║                                                           ║
║            Algorithmic Crypto Trading Engine             ║
║                                                           ║
╚═══════════════════════════════════════════════════════════╝
`
	fmt.Println(cyan(banner))
}

// PrintSection prints a section header
func PrintSection(title string) {
	line := strings.Repeat("═", 60)
	fmt.Printf("\n%s\n", cyan(line))
	fmt.Printf("%s %s\n", cyan("▶"), bold(title))
	fmt.Printf("%s\n\n", cyan(line))
}

// PrintSuccess prints success message
func PrintSuccess(msg string) {
	fmt.Printf("%s %s\n", green("✓"), msg)
}

// PrintError prints error message
func PrintError(msg string) {
	fmt.Printf("%s %s\n", red("✗"), msg)
}

// PrintWarning prints warning message
func PrintWarning(msg string) {
	fmt.Printf("%s %s\n", yellow("⚠"), msg)
}

// PrintInfo prints info message
func PrintInfo(msg string) {
	fmt.Printf("%s %s\n", cyan("ℹ"), msg)
}

// PrintProgress shows a progress indicator
func PrintProgress(current, total int, prefix string) {
	percent := float64(current) / float64(total) * 100
	bar := progressBar(percent, 40)
	fmt.Printf("\r%s [%s] %.1f%% (%d/%d)", prefix, bar, percent, current, total)
	if current == total {
		fmt.Println()
	}
}

func progressBar(percent float64, width int) string {
	filled := int(percent / 100 * float64(width))
	empty := width - filled
	return green(strings.Repeat("█", filled)) + strings.Repeat("░", empty)
}

// PrintTradeSignal prints a trade signal
func PrintTradeSignal(side string, symbol string, price float64, quantity float64, timestamp time.Time) {
	var sideColor func(a ...interface{}) string
	var arrow string
	
	if side == "buy" {
		sideColor = green
		arrow = "↑"
	} else {
		sideColor = red
		arrow = "↓"
	}
	
	fmt.Printf("%s %s %s %s @ $%.2f (qty: %.4f) - %s\n",
		sideColor(arrow),
		sideColor(strings.ToUpper(side)),
		yellow(symbol),
		"",
		price,
		quantity,
		timestamp.Format("15:04:05"),
	)
}

// PrintTradeResult prints a completed trade
func PrintTradeResult(trade *engine.Trade) {
	pnlColor := red
	arrow := "↓"
	if trade.NetPnL > 0 {
		pnlColor = green
		arrow = "↑"
	}
	
	fmt.Printf("  %s Trade closed: %s %s | Entry: $%.2f | Exit: $%.2f | P&L: %s (%.2f%%)\n",
		pnlColor(arrow),
		yellow(trade.Symbol),
		trade.Side,
		trade.EntryPrice,
		trade.ExitPrice,
		pnlColor(fmt.Sprintf("$%.2f", trade.NetPnL)),
		(trade.NetPnL/trade.EntryPrice)*100,
	)
}

// PrintPerformanceSummary prints final performance metrics
func PrintPerformanceSummary(account *engine.Account, initialBalance float64) {
	PrintSection("PERFORMANCE SUMMARY")
	
	totalPnL := account.Balance - initialBalance
	totalTrades := len(account.TradeHistory)
	
	winningTrades := 0
	losingTrades := 0
	totalWinAmount := 0.0
	totalLossAmount := 0.0
	maxWin := 0.0
	maxLoss := 0.0
	
	for _, trade := range account.TradeHistory {
		if trade.NetPnL > 0 {
			winningTrades++
			totalWinAmount += trade.NetPnL
			if trade.NetPnL > maxWin {
				maxWin = trade.NetPnL
			}
		} else {
			losingTrades++
			totalLossAmount += trade.NetPnL
			if trade.NetPnL < maxLoss {
				maxLoss = trade.NetPnL
			}
		}
	}
	
	winRate := 0.0
	if totalTrades > 0 {
		winRate = float64(winningTrades) / float64(totalTrades) * 100
	}
	
	avgWin := 0.0
	avgLoss := 0.0
	if winningTrades > 0 {
		avgWin = totalWinAmount / float64(winningTrades)
	}
	if losingTrades > 0 {
		avgLoss = totalLossAmount / float64(losingTrades)
	}
	
	profitFactor := 0.0
	if totalLossAmount != 0 {
		profitFactor = totalWinAmount / -totalLossAmount
	}
	
	returnPct := (totalPnL / initialBalance) * 100
	
	// Print metrics
	fmt.Printf("  %-25s %s\n", "Initial Balance:", cyan(fmt.Sprintf("$%.2f", initialBalance)))
	fmt.Printf("  %-25s %s\n", "Final Balance:", cyan(fmt.Sprintf("$%.2f", account.Balance)))
	
	pnlColor := red
	if totalPnL > 0 {
		pnlColor = green
	}
	fmt.Printf("  %-25s %s (%.2f%%)\n", "Total P&L:", pnlColor(fmt.Sprintf("$%.2f", totalPnL)), returnPct)
	
	fmt.Println()
	fmt.Printf("  %-25s %s\n", "Total Trades:", yellow(fmt.Sprintf("%d", totalTrades)))
	fmt.Printf("  %-25s %s (%s)\n", "Winning Trades:", green(fmt.Sprintf("%d", winningTrades)), green(fmt.Sprintf("%.1f%%", winRate)))
	fmt.Printf("  %-25s %s\n", "Losing Trades:", red(fmt.Sprintf("%d", losingTrades)))
	
	fmt.Println()
	fmt.Printf("  %-25s %s\n", "Average Win:", green(fmt.Sprintf("$%.2f", avgWin)))
	fmt.Printf("  %-25s %s\n", "Average Loss:", red(fmt.Sprintf("$%.2f", avgLoss)))
	fmt.Printf("  %-25s %s\n", "Max Win:", green(fmt.Sprintf("$%.2f", maxWin)))
	fmt.Printf("  %-25s %s\n", "Max Loss:", red(fmt.Sprintf("$%.2f", maxLoss)))
	
	pfColor := red
	if profitFactor > 1 {
		pfColor = green
	}
	fmt.Printf("  %-25s %s\n", "Profit Factor:", pfColor(fmt.Sprintf("%.2f", profitFactor)))
	
	fmt.Println()
}

// PrintPositionTable prints current open positions
func PrintPositionTable(positions []*engine.Position) {
	if len(positions) == 0 {
		PrintInfo("No open positions")
		return
	}
	
	PrintSection("OPEN POSITIONS")
	
	fmt.Printf("  %-10s %-8s %-12s %-12s %-12s %-10s\n",
		"Symbol", "Side", "Entry", "Current", "Quantity", "P&L",
	)
	fmt.Printf("  %s\n", strings.Repeat("─", 70))
	
	for _, pos := range positions {
		pnlColor := red
		if pos.UnrealizedPnL > 0 {
			pnlColor = green
		}
		
		fmt.Printf("  %-10s %-8s $%-11.2f $%-11.2f %-12.4f %s\n",
			pos.Symbol,
			pos.Side,
			pos.EntryPrice,
			pos.CurrentPrice,
			pos.Quantity,
			pnlColor(fmt.Sprintf("$%.2f", pos.UnrealizedPnL)),
		)
	}
	fmt.Println()
}

// PrintConfigSummary prints configuration summary
func PrintConfigSummary(symbol string, interval string, candles int, strategy string) {
	PrintSection("CONFIGURATION")
	fmt.Printf("  %-20s %s\n", "Symbol:", yellow(symbol))
	fmt.Printf("  %-20s %s\n", "Interval:", cyan(interval))
	fmt.Printf("  %-20s %s\n", "Candles Loaded:", magenta(fmt.Sprintf("%d", candles)))
	fmt.Printf("  %-20s %s\n", "Strategy:", green(strategy))
	fmt.Println()
}
