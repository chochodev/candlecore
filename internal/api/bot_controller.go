package api

import (
	"candlecore/internal/config"
	"candlecore/internal/bot"
	"candlecore/internal/exchange"
	"candlecore/internal/strategies"
	"candlecore/internal/websocket"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	gorillaws "github.com/gorilla/websocket"
)

// BotController manages bot lifecycle and configuration
type BotController struct {
	bot            *bot.Bot
	hub            *websocket.Hub
	provider       exchange.DataProvider
	wallet         *exchange.VirtualWallet // For dry-run mode
	isRunning      bool
	replayMode     bool
	replaySpeed    time.Duration // Delay between candles (default 1s)
	dryRun         bool          // Dry-run mode flag
	symbol         string
	timeframe      exchange.Timeframe
	strategyName   string
	strategyParams map[string]interface{}
	skipSignal     chan struct{} // Channel to trigger fast-forward skip
	isSkipping     bool          // Stateful skip flag
	configManager  *config.ConfigManager
	mu             sync.RWMutex
	stopChan       chan struct{}
}

// NewBotController creates a new bot controller
func NewBotController(provider exchange.DataProvider, hub *websocket.Hub, dataDir string) *BotController {
	cManager := config.NewConfigManager(dataDir)
	return &BotController{
		provider:       provider,
		hub:            hub,
		configManager:  cManager,
		wallet:         exchange.NewVirtualWallet(10000.0), // $10k virtual balance
		isRunning:      false,
		replayMode:     true,                     // Default to replay mode
		replaySpeed:    1 * time.Second,          // Default 1 second per candle
		dryRun:         true,                     // Default to dry-run for safety
		symbol:         "sol",
		timeframe:      exchange.Timeframe1h,
		strategyName:   "ma_crossover",
		stopChan:       make(chan struct{}),
		skipSignal:     make(chan struct{}, 1), // Buffered for 1 skip pulse
	}
}

// saveProfile internal helper to persist current state
func (bc *BotController) saveProfile() {
	profile := config.BotProfile{
		Symbol:      bc.symbol,
		Timeframe:   bc.timeframe,
		Strategy:    bc.strategyName,
		ReplayMode:  bc.replayMode,
		DryRun:      bc.dryRun,
		ReplaySpeed: bc.replaySpeed.Seconds(),
		Parameters:  bc.strategyParams,
		IsRunning:   bc.isRunning,
	}
	bc.configManager.Save(profile)
}

// Start starts the bot
func (bc *BotController) Start() error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if bc.isRunning {
		return fmt.Errorf("bot is already running")
	}

	// Create strategy using registry
	strategy, err := strategies.GetStrategy(bc.strategyName)
	if err != nil {
		return fmt.Errorf("failed to create strategy: %w", err)
	}

	// Apply configuration
	if bc.strategyParams != nil {
		strategy.Configure(bc.strategyParams)
	}

	// Create bot
	bc.bot = bot.NewBot(strategy, bc.provider, bot.Config{
		Symbol:         bc.symbol,
		Timeframe:      bc.timeframe,
		InitialBalance: 10000,
		PositionSize:   10,
	})

	bc.isRunning = true
	bc.stopChan = make(chan struct{})

	// Start processing
	go bc.run()

	bc.saveProfile() // Persist start state
	bc.hub.BroadcastStatus("started")
	log.Printf("Bot started: symbol=%s, timeframe=%s, strategy=%s", bc.symbol, bc.timeframe, bc.strategyName)

	return nil
}

// Stop stops the bot
func (bc *BotController) Stop() error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if !bc.isRunning {
		return fmt.Errorf("bot is not running")
	}

	close(bc.stopChan)
	bc.isRunning = false
	bc.saveProfile() // Persist stop state
	bc.hub.BroadcastStatus("stopped")

	log.Println("Bot stopped")
	return nil
}

// Reset resets the bot state (balance, trades, current index)
func (bc *BotController) Reset() error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if bc.isRunning {
		return fmt.Errorf("cannot reset while bot is running")
	}

	bc.wallet = exchange.NewVirtualWallet(10000.0) // Reset to $10k
	bc.isSkipping = false
	bc.hub.BroadcastStatus("reset")
	
	log.Println("Bot state reset to initial parameters")
	return nil
}

// run processes candles and executes strategy
func (bc *BotController) run() {
	log.Printf("Starting bot: symbol=%s, timeframe=%s, strategy=%s, replay=%v, speed=%v",
		bc.symbol, bc.timeframe, bc.strategyName, bc.replayMode, bc.replaySpeed)

	// 📊 STRICT HISTORICAL DATA LOADING
	candles, err := bc.provider.GetCandles(bc.symbol, bc.timeframe, 0)
	if err != nil || len(candles) == 0 {
		log.Printf("❌ CRITICAL ERROR: Could not load data for %s/%s. File not found in data/historical. Error: %v", bc.symbol, bc.timeframe, err)
		bc.hub.BroadcastStatus("failed: data_missing")
		bc.Stop() 
		return
	}

	log.Printf("Loaded %d REAL candles - Ready for production research", len(candles))

	// Buffer for high-speed jumps to provide context
	var jumpContext []websocket.CandleData

	// Stream candles one-by-one like a real-time WebSocket feed
	for i, candle := range candles {
		select {
		case <-bc.stopChan:
			log.Println("Bot stopped by user")
			return
		default:
		}

		bc.mu.RLock()
		skip := bc.isSkipping
		bc.mu.RUnlock()

		if skip && i % 1000 == 0 {
			log.Printf("🛰️ NEURAL WARP SEARCHING: Index %d/%d (%s)...", i, len(candles), bc.symbol)
		}

		// Prepare current candle data
		cData := websocket.CandleData{
			Symbol:    bc.symbol,
			Timeframe: string(bc.timeframe),
			Timestamp: candle.Timestamp,
			Open:      candle.Open,
			High:      candle.High,
			Low:       candle.Low,
			Close:     candle.Close,
			Volume:    candle.Volume,
			Indicators: make(map[string]float64),
		}

		if i >= 30 {
			decision, err := bc.bot.ProcessCandle(candle)
			if err == nil {
				// Enrich candle with indicators for the chart
				cData.Indicators["fast_ma"] = decision.Indicators["fast_ma"]
				cData.Indicators["slow_ma"] = decision.Indicators["slow_ma"]

				if decision.Signal != "hold" {
					if skip {
						// 🚀 WARP ARRIVED: Release RLock context is already gone, just use skip local
						finalBatch := append(jumpContext, cData)
						log.Printf("🎯 WARP SIGNAL DETECTED at index %d: %s at $%.2f", i, decision.Signal, decision.Price)
						log.Printf("🛰️ NEURAL WARP COMPLETE: Sending %d context candles", len(finalBatch))
						log.Printf("🚀 Warp Arrived: Context Sync Triggered")
						bc.hub.BroadcastHistory(finalBatch)
						
						bc.mu.Lock()
						bc.isSkipping = false
						bc.mu.Unlock()
						skip = false 
						bc.hub.BroadcastStatus("resumed")
					}
					bc.hub.BroadcastDecision(decision)
				}

				if !skip {
					if pos := bc.bot.GetPosition(); pos != nil {
						bc.hub.BroadcastPosition(pos)
					}
					trades := bc.bot.GetTrades()
					wins := 0
					for _, t := range trades {
						if t.RealizedPnL > 0 {
							wins++
						}
					}
					winRate := 0.0
					if len(trades) > 0 {
						winRate = (float64(wins) / float64(len(trades))) * 100
					}
					bc.hub.BroadcastPnL(websocket.PnLData{
						Balance: bc.bot.GetBalance(), TotalPnL: bc.bot.GetTotalPnL(), WinRate: winRate,
					})
				}
			}
		}

		// Update history buffer (Keep last 200)
		jumpContext = append(jumpContext, cData)
		if len(jumpContext) > 200 {
			jumpContext = jumpContext[1:]
		}

		if !skip {
			bc.hub.BroadcastCandle(candle, bc.symbol, string(bc.timeframe), cData.Indicators)
			time.Sleep(bc.replaySpeed)
		}
	}
	
	bc.mu.RLock()
	wasSkipping := bc.isSkipping
	bc.mu.RUnlock()

	if wasSkipping {
		log.Printf("⚠️ Simulation reached EOF without finding a signal. Sending last context.")
		bc.hub.BroadcastHistory(jumpContext)
		bc.hub.BroadcastStatus("finished: no_signal_detected")
	} else {
		bc.hub.BroadcastStatus("finished")
	}

	bc.mu.Lock()
	bc.isSkipping = false
	bc.isRunning = false
	bc.mu.Unlock()
	bc.saveProfile()
}

func (bc *BotController) SkipToNextTrade() {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	if !bc.isRunning { return }
	bc.isSkipping = true
	bc.hub.BroadcastStatus("skipping")
}

// GetStatus returns bot status... (truncating for brevitiy in this turn)
func (bc *BotController) GetStatus() map[string]interface{} {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	status := map[string]interface{}{
		"running":      bc.isRunning,
		"symbol":       bc.symbol,
		"timeframe":    bc.timeframe,
		"strategy":     bc.strategyName,
		"replay_mode":  bc.replayMode,
		"replay_speed": bc.replaySpeed.Seconds(),
		"dry_run":      bc.dryRun,
	}
	if bc.bot != nil {
		status["balance"] = bc.bot.GetBalance()
		status["total_pnl"] = bc.bot.GetTotalPnL()
		status["position"] = bc.bot.GetPosition()
		status["trades_count"] = len(bc.bot.GetTrades())
	}
	return status
}

func (bc *BotController) Configure(symbol string, timeframe exchange.Timeframe, strategy string, replayMode, dryRun bool, replaySpeed float64, strategyParams map[string]interface{}) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	if bc.isRunning { return fmt.Errorf("cannot configure while running") }
	bc.symbol = symbol
	bc.timeframe = timeframe
	bc.strategyName = strategy
	bc.replayMode = replayMode
	bc.dryRun = dryRun
	bc.strategyParams = strategyParams
	if replaySpeed > 0 { bc.replaySpeed = time.Duration(replaySpeed * float64(time.Second)) }
	bc.saveProfile()
	return nil
}

func (bc *BotController) HandleWebSocket(c *gin.Context) {
	conn, _ := upgrader.Upgrade(c.Writer, c.Request, nil)
	client := websocket.NewClient(bc.hub, conn)
	bc.hub.Register <- client
	go client.WritePump()
	go client.ReadPump()
}

func (bc *BotController) SetupRoutes(router *gin.Engine) {
	router.GET("/ws", bc.HandleWebSocket)
	api := router.Group("/api/v1/bot")
	{
		api.POST("/start", func(c *gin.Context) {
			if err := bc.Start(); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
			c.JSON(200, gin.H{"status": "started"})
		})
		api.POST("/stop", func(c *gin.Context) {
			if err := bc.Stop(); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
			c.JSON(200, gin.H{"status": "stopped"})
		})
		api.POST("/skip", func(c *gin.Context) { bc.SkipToNextTrade(); c.JSON(200, gin.H{"status": "skipping"}) })
		api.POST("/reset", func(c *gin.Context) {
			if err := bc.Reset(); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
			c.JSON(200, gin.H{"status": "reset"})
		})
		api.GET("/status", func(c *gin.Context) { c.JSON(200, bc.GetStatus()) })
		api.POST("/configure", func(c *gin.Context) {
			var req struct {
				Symbol string `json:"symbol"`; Timeframe string `json:"timeframe"`; Strategy string `json:"strategy"`; 
				ReplayMode bool `json:"replay_mode"`; DryRun bool `json:"dry_run"`; ReplaySpeed float64 `json:"replay_speed"`;
				StrategyParams map[string]interface{} `json:"strategy_params"`
			}
			if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
			if err := bc.Configure(req.Symbol, exchange.Timeframe(req.Timeframe), req.Strategy, req.ReplayMode, req.DryRun, req.ReplaySpeed, req.StrategyParams); err != nil {
				c.JSON(400, gin.H{"error": err.Error()}); return
			}
			c.JSON(200, gin.H{"status": "configured"})
		})
	}
}

var upgrader = gorillaws.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
