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
	"candlecore/internal/config"
	"candlecore/internal/engine"
	"candlecore/internal/loader"
	"candlecore/internal/logger"
	"candlecore/internal/store"
	"candlecore/internal/strategy"
)

func main() {
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
	log.Info("Starting Candlecore trading engine")

	// Initialize state store based on configuration
	var stateStore engine.StateStore
	
	if cfg.Database.Enabled {
		log.Info("Initializing PostgreSQL state store", 
			"host", cfg.Database.Host, 
			"database", cfg.Database.DBName,
		)
		
		pgStore, err := store.NewPostgresStore(
			cfg.GetDatabaseConnectionString(),
			cfg.Database.AccountID,
		)
		if err != nil {
			log.Error("Failed to connect to PostgreSQL", "error", err)
			os.Exit(1)
		}
		defer pgStore.Close()
		
		// Initialize database schema (creates tables, views, triggers if needed)
		initCtx, initCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer initCancel()
		
		if err := pgStore.Initialize(initCtx); err != nil {
			log.Error("Failed to initialize database schema", "error", err)
			os.Exit(1)
		}
		log.Info("Database schema initialized successfully")
		
		stateStore = pgStore
	} else {
		log.Info("Using file-based state store", "directory", cfg.StateDirectory)
		
		fileStore, err := store.NewFileStore(cfg.StateDirectory)
		if err != nil {
			log.Error("Failed to initialize file store", "error", err)
			os.Exit(1)
		}
		
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

	// Run the engine with candle data
	// In production, this would load from a file or database
	candles := loadCandleData(cfg.DataSource, log)
	
	log.Info("Starting backtesting run",
		"candles", len(candles),
		"initial_balance", cfg.InitialBalance,
		"strategy", strat.Name(),
	)

	// Run the engine
	if err := tradingEngine.Run(ctx, candles); err != nil {
		log.Error("Engine failed", "error", err)
		os.Exit(1)
	}

	// Print final results
	account := paperBroker.GetAccount()
	log.Info("Backtest completed",
		"final_balance", account.Balance,
		"total_pnl", account.Balance-cfg.InitialBalance,
		"total_trades", len(account.TradeHistory),
	)

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
	
	startTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	candles = make([]engine.Candle, 100)
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
