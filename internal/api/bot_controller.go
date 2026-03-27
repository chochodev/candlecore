package api

import (
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
	bot          *bot.Bot
	hub          *websocket.Hub
	provider     exchange.DataProvider
	wallet       *exchange.VirtualWallet // For dry-run mode
	isRunning    bool
	replayMode   bool
	replaySpeed  time.Duration // Delay between candles (default 1s)
	dryRun       bool          // Dry-run mode flag
	symbol       string
	timeframe    exchange.Timeframe
	strategyName string
	mu           sync.RWMutex
	stopChan     chan struct{}
}

// NewBotController creates a new bot controller
func NewBotController(provider exchange.DataProvider, hub *websocket.Hub) *BotController {
	return &BotController{
		provider:     provider,
		hub:          hub,
		wallet:       exchange.NewVirtualWallet(10000.0), // $10k virtual balance
		isRunning:    false,
		replayMode:   true,                     // Default to replay mode
		replaySpeed:  1 * time.Second,          // Default 1 second per candle
		dryRun:       true,                     // Default to dry-run for safety
		symbol:       "bitcoin",
		timeframe:    exchange.Timeframe1h,
		strategyName: "ma_crossover",
		stopChan:     make(chan struct{}),
	}
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
	bc.hub.BroadcastStatus("stopped")

	log.Println("Bot stopped")
	return nil
}

// run processes candles and executes strategy
func (bc *BotController) run() {
	log.Printf("Starting bot: symbol=%s, timeframe=%s, strategy=%s, replay=%v, speed=%v",
		bc.symbol, bc.timeframe, bc.strategyName, bc.replayMode, bc.replaySpeed)

	// 📊 STRICT HISTORICAL DATA LOADING
	// We only use the local file provider. If data is missing, we FAIL.
	candles, err := bc.provider.GetCandles(bc.symbol, bc.timeframe, 0)
	if err != nil || len(candles) == 0 {
		log.Printf("❌ CRITICAL ERROR: Could not load data for %s/%s. File not found in data/historical. Error: %v", bc.symbol, bc.timeframe, err)
		bc.hub.BroadcastStatus("failed: data_missing")
		bc.Stop() 
		return
	}

	log.Printf("Loaded %d REAL candles - Ready for production research", len(candles))

	log.Printf("Replay speed: 1 candle every %v (streaming from data/historical)", bc.replaySpeed)

	// Stream candles one-by-one like a real-time WebSocket feed
	for i, candle := range candles {
		// Check for stop signal
		select {
		case <-bc.stopChan:
			log.Println("Bot stopped by user")
			return
		default:
		}

		// Process candle through bot strategy to get indicators
		var indicators map[string]float64
		if i >= 30 {
			decision, err := bc.bot.ProcessCandle(candle)
			if err != nil {
				log.Printf("Error processing candle #%d: %v", i, err)
			} else {
				indicators = decision.Indicators
				// Annotate indicators with signals for chart markers
				switch decision.Signal {
				case bot.SignalBuy:
					indicators["signal"] = 1
				case bot.SignalSell:
					indicators["signal"] = -1
				}

				// Broadcast events to UI
				bc.hub.BroadcastDecision(decision)

				// Log significant trading decisions
				if decision.Signal != "hold" {
					log.Printf("🎯 SIGNAL #%d: %s at $%.2f | Confidence: %.0f%% | %s",
						i, decision.Signal, decision.Price, decision.Confidence, decision.Reasoning)
				}

				// Update and broadcast position if trade was executed
				if pos := bc.bot.GetPosition(); pos != nil {
					bc.hub.BroadcastPosition(pos)
					if decision.Signal == "buy" {
						log.Printf("✅ Position OPENED: Entry $%.2f", pos.EntryPrice)
					}
				} else if decision.Signal == "sell" {
					log.Printf("💰 Position CLOSED: P&L $%.2f", bc.bot.GetTotalPnL())
				}

				// Calculate and broadcast win rate
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
					Balance:  bc.bot.GetBalance(),
					TotalPnL: bc.bot.GetTotalPnL(),
					WinRate:  winRate,
				})
			}
		}

		// 🔴 Stream this candle to frontend with computed indicators
		bc.hub.BroadcastCandle(candle, bc.symbol, string(bc.timeframe), indicators)

		if i < 30 {
			log.Printf("Warming up... %d/%d candles", i+1, 30)
		}

		// 🕐 Simulate real-time streaming delay
		// This makes it feel like getting candles from a live exchange WebSocket
		if bc.replayMode {
			time.Sleep(bc.replaySpeed)
		}
	}

	log.Printf("✅ Finished processing %d candles | Final P&L: $%.2f", len(candles), bc.bot.GetTotalPnL())
	bc.Stop()
}

// GetStatus returns bot status
func (bc *BotController) GetStatus() map[string]interface{} {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	status := map[string]interface{}{
		"running":      bc.isRunning,
		"symbol":       bc.symbol,
		"timeframe":    bc.timeframe,
		"strategy":     bc.strategyName,
		"replay_mode":  bc.replayMode,
		"replay_speed": bc.replaySpeed.Seconds(), // Speed in seconds
		"dry_run":      bc.dryRun,
	}

	// Add wallet info in dry-run mode
	if bc.dryRun {
		status["wallet_balance"] = bc.wallet.GetBalance()
		status["wallet_pnl"] = bc.wallet.GetTotalPnL()
	}

	if bc.bot != nil {
		status["balance"] = bc.bot.GetBalance()
		status["total_pnl"] = bc.bot.GetTotalPnL()
		status["position"] = bc.bot.GetPosition()
		status["trades_count"] = len(bc.bot.GetTrades())
	}

	return status
}

// Configure updates bot configuration
func (bc *BotController) Configure(symbol string, timeframe exchange.Timeframe, strategy string, replayMode, dryRun bool, replaySpeed float64) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if bc.isRunning {
		return fmt.Errorf("cannot configure while bot is running")
	}

	bc.symbol = symbol
	bc.timeframe = timeframe
	bc.strategyName = strategy
	bc.replayMode = replayMode
	bc.dryRun = dryRun

	// Set replay speed (default 1 second if not specified)
	if replaySpeed > 0 {
		bc.replaySpeed = time.Duration(replaySpeed * float64(time.Second))
	} else {
		bc.replaySpeed = 1 * time.Second
	}

	return nil
}

var upgrader = gorillaws.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// HandleWebSocket handles WebSocket connections
func (bc *BotController) HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := websocket.NewClient(bc.hub, conn)
	bc.hub.Register <- client

	// Start client pumps
	go client.WritePump()
	go client.ReadPump()
}

// SetupRoutes adds bot control routes to the API
func (bc *BotController) SetupRoutes(router *gin.Engine) {
	// WebSocket endpoint
	router.GET("/ws", bc.HandleWebSocket)

	// Bot control endpoints
	api := router.Group("/api/v1/bot")
	{
		api.POST("/start", func(c *gin.Context) {
			if err := bc.Start(); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "started"})
		})

		api.POST("/stop", func(c *gin.Context) {
			if err := bc.Stop(); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "stopped"})
		})

		api.GET("/status", func(c *gin.Context) {
			c.JSON(http.StatusOK, bc.GetStatus())
		})

		api.POST("/configure", func(c *gin.Context) {
			var req struct {
				Symbol      string  `json:"symbol" binding:"required"`
				Timeframe   string  `json:"timeframe" binding:"required"`
				Strategy    string  `json:"strategy" binding:"required"`
				ReplayMode  bool    `json:"replay_mode"`
				DryRun      bool    `json:"dry_run"`
				ReplaySpeed float64 `json:"replay_speed"` // Seconds per candle (default 1.0)
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			timeframe := exchange.Timeframe(req.Timeframe)
			if !timeframe.IsValid() {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid timeframe"})
				return
			}

			if err := bc.Configure(req.Symbol, timeframe, req.Strategy, req.ReplayMode, req.DryRun, req.ReplaySpeed); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"status": "configured"})
		})

		api.GET("/trades", func(c *gin.Context) {
			bc.mu.RLock()
			defer bc.mu.RUnlock()

			if bc.bot == nil {
				c.JSON(http.StatusOK, gin.H{"trades": []interface{}{}})
				return
			}

			c.JSON(http.StatusOK, gin.H{"trades": bc.bot.GetTrades()})
		})
	}
	
	// Strategy management endpoints
	strategyRoutes := router.Group("/api/v1/strategies")
	{
		strategyRoutes.GET("", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"strategies": strategies.ListStrategies(),
			})
		})
	}
}
