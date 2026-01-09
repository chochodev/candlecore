package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"candlecore/internal/broker"
	"candlecore/internal/cmd"
	"candlecore/internal/config"
	"candlecore/internal/engine"
	"candlecore/internal/fetcher"
	"candlecore/internal/loader"
	"candlecore/internal/logger"
	"candlecore/internal/store"
	"candlecore/internal/strategy"
	"candlecore/internal/ui"
)

func main() {
	// Check if running as CLI command
	if len(os.Args) > 1 {
		cmd.Execute()
		return
	}
	
	// Parse command-line flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	log := logger.New(cfg.LogLevel)
	
	// Display banner
	ui.PrintBanner()
	ui.PrintSuccess("Candlecore trading engine initialized")
	fmt.Println()

	// Initialize state store based on configuration
	var stateStore engine.StateStore
	
	ui.PrintSection("STATE PERSISTENCE")
	
	if cfg.Database.Enabled {
		ui.PrintInfo(fmt.Sprintf("Connecting to PostgreSQL at %s:%d...", cfg.Database.Host, cfg.Database.Port))
		
		pgStore, err := store.NewPostgresStore(
			cfg.GetDatabaseConnectionString(),
			cfg.Database.AccountID,
		)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to connect to PostgreSQL: %v", err))
			os.Exit(1)
		}
		defer pgStore.Close()
		
		// Initialize database schema (creates tables, views, triggers if needed)
		initCtx, initCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer initCancel()
		
		if err := pgStore.Initialize(initCtx); err != nil {
			ui.PrintError(fmt.Sprintf("Failed to initialize database schema: %v", err))
			os.Exit(1)
		}
		ui.PrintSuccess("PostgreSQL connected and schema initialized")
		
		stateStore = pgStore
	} else {
		ui.PrintInfo(fmt.Sprintf("Using file-based storage: %s", cfg.StateDirectory))
		
		fileStore, err := store.NewFileStore(cfg.StateDirectory)
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to initialize file store: %v", err))
			os.Exit(1)
		}
		ui.PrintSuccess("File store initialized")
		
		stateStore = fileStore
	}

	// Initialize paper trading broker
	paperBroker := broker.NewPaperBroker(
		cfg.InitialBalance,
		cfg.TakerFee,
		cfg.MakerFee,
		cfg.SlippageBps,
		log,
	)

	// Load previous state if exists
	if err := stateStore.LoadState(paperBroker); err != nil {
		log.Warn("No previous state found or failed to load", "error", err)
	}

	// Initialize strategy
	// For now, using a simple moving average crossover example
	strat := strategy.NewSimpleMAStrategy(
		cfg.Strategy.FastPeriod,
		cfg.Strategy.SlowPeriod,
		cfg.Strategy.PositionSize,
	)

	// Initialize trading engine
	tradingEngine := engine.New(
		paperBroker,
		strat,
		stateStore,
		log,
	)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Info("Shutdown signal received, stopping engine...")
		cancel()
	}()

	// Load candle data based on configuration
	var candles []engine.Candle
	
	ui.PrintSection("DATA LOADING")
	
	if cfg.LiveData.Enabled {
		ui.PrintInfo(fmt.Sprintf("Fetching live %s data from CoinGecko...", cfg.LiveData.Symbol))
		
		var err error
		candles, err = fetchLiveData(cfg, log)
		if err != nil {
			ui.PrintWarning(fmt.Sprintf("Live data fetch failed: %v", err))
			ui.PrintInfo("Falling back to synthetic data...")
			candles = generateSyntheticData(log)
		} else {
			ui.PrintSuccess(fmt.Sprintf("Fetched %d candles from CoinGecko", len(candles)))
		}
	} else {
		ui.PrintInfo(fmt.Sprintf("Loading data from: %s", cfg.DataSource))
		candles = loadCandleData(cfg.DataSource, log)
	}
	
	ui.PrintConfigSummary(cfg.LiveData.Symbol, "1d", len(candles), strat.Name())
	
	ui.PrintSection("BACKTEST EXECUTION")
	ui.PrintInfo(fmt.Sprintf("Starting backtest with $%.2f initial balance", cfg.InitialBalance))
	fmt.Println()

	// Run the engine
	if err := tradingEngine.Run(ctx, candles); err != nil {
		ui.PrintError(fmt.Sprintf("Engine failed: %v", err))
		os.Exit(1)
	}

	// Print final results
	fmt.Println()
	account := paperBroker.GetAccount()
	
	ui.PrintPerformanceSummary(account, cfg.InitialBalance)
	ui.PrintPositionTable(account.Positions)

	// Save final state
	if err := stateStore.SaveState(paperBroker); err != nil {
		log.Error("Failed to save final state", "error", err)
	}

	log.Info("Candlecore shutdown complete")
}

// loadCandleData loads candle data from CSV file or generates synthetic data
func loadCandleData(source string, log logger.Logger) []engine.Candle {
	log.Info("Loading candle data", "source", source)
	
	// Try to load from CSV file
	csvLoader := loader.NewCSVLoader(source)
	candles, err := csvLoader.Load()
	
	if err == nil {
		log.Info("Loaded candle data from CSV", "count", len(candles))
		return candles
	}
	
	// If CSV loading fails, generate synthetic data for testing
	log.Warn("Failed to load CSV, using synthetic data", "error", err)
	return generateSyntheticData(log)
}

// generateSyntheticData creates synthetic candle data for testing
func generateSyntheticData(log logger.Logger) []engine.Candle {
	startTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := make([]engine.Candle, 100)
	basePrice := 100.0
	
	for i := 0; i < 100; i++ {
		variance := float64(i%10) - 5.0
		open := basePrice + variance
		close := basePrice + variance + 1.0
		high := close + 0.5
		low := open - 0.5
		
		candles[i] = engine.Candle{
			Timestamp: startTime.Add(time.Duration(i) * time.Hour),
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    1000.0 + float64(i*10),
		}
	}
	
	log.Info("Generated synthetic candle data", "count", len(candles))
	return candles
}

// fetchLiveData fetches live candle data from CoinGecko
func fetchLiveData(cfg *config.Config, log logger.Logger) ([]engine.Candle, error) {
	cgFetcher := fetcher.NewCoinGeckoFetcher()
	
	// Convert symbol to CoinGecko coin ID
	coinID := fetcher.CoinIDFromSymbol(cfg.LiveData.Symbol)
	if coinID == "" {
		return nil, fmt.Errorf("unsupported symbol: %s (supported: BTCUSDT, ETHUSDT)", cfg.LiveData.Symbol)
	}
	
	if !fetcher.ValidateCoinID(coinID) {
		return nil, fmt.Errorf("unsupported coin: %s", coinID)
	}
	
	// CoinGecko provides daily candles, calculate days needed
	days := cfg.LiveData.InitialFetch / 24
	if days < 1 {
		days = 1
	}
	if days > 365 {
		days = 365
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	
	log.Info("Fetching candles from CoinGecko",
		"coin", coinID,
		"days", days,
		"expected_candles", cfg.LiveData.InitialFetch,
	)
	
	candles, err := cgFetcher.FetchCandles(ctx, coinID, days)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch candles: %w", err)
	}
	
	// Limit to requested number of candles (most recent)
	if len(candles) > cfg.LiveData.InitialFetch {
		candles = candles[len(candles)-cfg.LiveData.InitialFetch:]
	}
	
	log.Info("Successfully fetched live candles from CoinGecko", "count", len(candles))
	return candles, nil
}
