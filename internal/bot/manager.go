package bot

import (
	"candlecore/internal/engine"
	"candlecore/internal/logger"
	"context"
	"sync"
)

// BotInstance represents a single trading entity (e.g. BTC on Alpha Prime)
type BotInstance struct {
	Symbol     string
	Strategy   engine.Strategy
	Broker     engine.Broker
	StateStore engine.StateStore
	DataFeed   []engine.Candle // In production, this would be a channel or real-time feed
}

// MultiBotManager orchestrates multiple bots running in parallel using Go Concurrency.
type MultiBotManager struct {
	Instances []*BotInstance
	Logger    logger.Logger
}

func NewMultiBotManager(log logger.Logger) *MultiBotManager {
	return &MultiBotManager{
		Instances: make([]*BotInstance, 0),
		Logger:    log,
	}
}

func (m *MultiBotManager) AddBot(bi *BotInstance) {
	m.Instances = append(m.Instances, bi)
}

// StartAll launches every bot instance in its own separate Goroutine.
func (m *MultiBotManager) StartAll(ctx context.Context) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(m.Instances))

	for _, instance := range m.Instances {
		wg.Add(1)

		// 🚀 LAUNCHING GOROUTINE
		// This block runs in PARALLEL. Go does not wait for it to finish.
		go func(bi *BotInstance) {
			defer wg.Done()
			
			m.Logger.Info("Starting bot thread", "symbol", bi.Symbol, "strategy", bi.Strategy.Name())

			// Initialize the single engine for this specific pair
			e := engine.New(bi.Broker, bi.Strategy, bi.StateStore, m.Logger)
			
			// Run the loop for this specific bot
			if err := e.Run(ctx, bi.DataFeed); err != nil {
				m.Logger.Error("Bot thread failed", "symbol", bi.Symbol, "error", err)
				errChan <- err
			}
		}(instance)
	}

	// Wait for all bots to finish or the context to be cancelled
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Return the first error we encounter (if any)
	for err := range errChan {
		if err != nil { return err }
	}

	return nil
}
