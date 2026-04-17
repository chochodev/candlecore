package cmd

import (
	"candlecore/internal/api"
	"candlecore/internal/bot"
	"candlecore/internal/engine"
	"candlecore/internal/exchange"
	"candlecore/internal/strategies"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	dataDir        string
	symbol         string
	timeframe      string
	strategyName   string
	initialBalance float64
	startStr       string
	endStr         string
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "candlecore",
	Short: "Candlecore - Algorithmic Crypto Trading Bot",
	Long: `Candlecore is a production-ready algorithmic trading bot for cryptocurrency markets.

Features:
  - Real-time WebSocket streaming
  - Bot control & configuration
  - Historical replay & backtesting
  - Multiple trading strategies
  - Technical indicators`,
}

// serveCmd starts the API server
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the API server",
	Long:  "Starts the REST API and WebSocket server for bot control and frontend integration.",
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetString("port")
		
		fmt.Printf("Starting Candlecore API Server on port %s...\n", port)
		fmt.Printf("Data directory: %s\n", dataDir)
		fmt.Println()
		
		server := api.NewServer(dataDir)
		
		if err := server.Run(port); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	},
}

// backtestCmd runs a high-performance backtest
var backtestCmd = &cobra.Command{
	Use:   "backtest",
	Short: "Run a high-performance backtest on historical data",
	Long:  "Executes a trading strategy across the entire historical dataset in milliseconds and provides performance metrics.",
	Run: func(cmd *cobra.Command, args []string) {
		var startTime, endTime time.Time
		if startStr != "" {
			startTime, _ = time.Parse("2006-01-02", startStr)
		}
		if endStr != "" {
			endTime, _ = time.Parse("2006-01-02", endStr)
			// Move to end of day if only date is provided
			endTime = endTime.Add(23*time.Hour + 59*time.Minute)
		}

		tf := exchange.Timeframe(timeframe)
		fmt.Printf("Starting Backtest: strategy=%s, symbol=%s, timeframe=%s\n", strategyName, symbol, tf)
		if !startTime.IsZero() || !endTime.IsZero() {
			fmt.Printf("Window: %s to %s\n", startStr, endStr)
		}

		// 1. Data Provider
		provider := exchange.NewLocalFileProvider(dataDir)

		// 2. Load and Filter Candles
		allCandles, err := provider.GetCandles(symbol, tf, 0)
		if err != nil {
			fmt.Printf("Error loading data: %v\n", err)
			return
		}

		var candles []exchange.Candle
		startIndex := -1
		for i, c := range allCandles {
			if !startTime.IsZero() && c.Timestamp.Before(startTime) {
				continue
			}
			if !endTime.IsZero() && c.Timestamp.After(endTime) {
				break
			}
			if startIndex == -1 {
				startIndex = i
			}
			candles = append(candles, c)
		}

		if len(candles) == 0 {
			fmt.Println("❌ No candles found in specified range.")
			return
		}

		// Add 200 candles of warm-up data before the start
		warmupCount := 200
		warmupStart := startIndex - warmupCount
		if warmupStart < 0 {
			warmupStart = 0
		}
		
		runCandles := append([]exchange.Candle{}, allCandles[warmupStart:startIndex]...)
		runCandles = append(runCandles, candles...)

		fmt.Printf("📊 Running on %d candles (+%d warm-up)\n", len(candles), len(runCandles)-len(candles))

		// 3. Strategy
		strategy, err := strategies.GetStrategy(strategyName)
		if err != nil {
			fmt.Printf("❌ Error finding strategy: %v\n", err)
			return
		}

		// 4. Initialize Bot (Headless)
		b := bot.NewBot(strategy, provider, bot.Config{
			Symbol:         symbol,
			Timeframe:      tf,
			InitialBalance: initialBalance,
			PositionSize:   10,
		})

		// 5. Run Backtest
		start := time.Now()
		if err := b.RunBacktest(runCandles); err != nil {
			fmt.Printf("Backtest error: %v\n", err)
			return
		}
		duration := time.Since(start)

		// 6. Calculate Metrics (Filter results back to the requested window)
		history := b.GetBalanceHistory()
		botTrades := b.GetTrades()
		
		// Trim history to just the requested window (exclude warm-up phase if possible)
		// Simplified: we just use the final balance from the run
		
		engineTrades := make([]engine.Trade, 0)
		for _, t := range botTrades {
			// Only count trades that were opened within the window
			if !startTime.IsZero() && t.OpenedAt.Before(startTime) {
				continue 
			}
			engineTrades = append(engineTrades, engine.Trade{
				PnL: t.RealizedPnL,
			})
		}

		report := engine.CalculateMetrics(initialBalance, engineTrades, history)
		report.StrategyName = strategyName
		report.Symbol = symbol
		report.Timeframe = timeframe
		report.Duration = duration

		// 7. Output Results
		fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("🏁 BACKTEST RESULTS: %s\n", strategyName)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("💰 Initial Balance:  $%.2f\n", report.InitialBalance)
		fmt.Printf("💵 Final Balance:    $%.2f\n", report.FinalBalance)
		fmt.Printf("📈 Total PnL:        $%.2f (%.2f%%)\n", report.TotalPnL, report.TotalPnLPct)
		fmt.Printf("🔄 Total Trades:     %d\n", report.TotalTrades)
		fmt.Printf("✅ Win Rate:         %.2f%%\n", report.WinRate)
		fmt.Printf("📉 Max Drawdown:     $%.2f (%.2f%%)\n", report.MaxDrawdown, report.MaxDrawdownPct)
		fmt.Printf("📊 Sharpe Ratio:     %.2f\n", report.SharpeRatio)
		fmt.Printf("⏱️  Execution Time:   %v\n", report.Duration)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dataDir, "data-dir", "data/historical", "Directory for storing historical data")
	
	serveCmd.Flags().StringP("port", "p", "8080", "Port to run the server on")
	
	backtestCmd.Flags().StringVarP(&symbol, "symbol", "s", "sol", "Trading symbol")
	backtestCmd.Flags().StringVar(&timeframe, "timeframe", "15m", "Timeframe (e.g., 5m, 15m, 1h)")
	backtestCmd.Flags().StringVar(&strategyName, "strategy", "vanguard_m15", "Strategy name")
	backtestCmd.Flags().Float64VarP(&initialBalance, "balance", "b", 10000.0, "Initial balance")
	backtestCmd.Flags().StringVar(&startStr, "start", "", "Start date (YYYY-MM-DD)")
	backtestCmd.Flags().StringVar(&endStr, "end", "", "End date (YYYY-MM-DD)")

	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(backtestCmd)
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
