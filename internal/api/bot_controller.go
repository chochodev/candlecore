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
	isRunning    bool
	replayMode   bool
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
		isRunning:    false,
		replayMode:   false,
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

	// Create strategy
	var strategy bot.Strategy
	switch bc.strategyName {
	case "ma_crossover":
		strategy = strategies.NewSimpleMAStrategy(10, 30)
	case "rsi":
		strategy = strategies.NewRSIStrategy(14, 30, 70)
	default:
		return fmt.Errorf("unknown strategy: %s", bc.strategyName)
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
	// Get candles
	candles, err := bc.provider.GetCandles(bc.symbol, bc.timeframe, 0)
	if err != nil {
		log.Printf("Error loading candles: %v", err)
		bc.Stop()
		return
	}

	log.Printf("Processing %d candles for %s (%s)", len(candles), bc.symbol, bc.timeframe)

	// Process each candle
	for i, candle := range candles {
		select {
		case <-bc.stopChan:
			return
		default:
		}

		// Broadcast candle
		bc.hub.BroadcastCandle(candle, bc.symbol, string(bc.timeframe))

		// Process candle (skip first 30 for MA warm-up)
		if i >= 30 {
			decision, err := bc.bot.ProcessCandle(candle)
			if err != nil {
				log.Printf("Error processing candle: %v", err)
				continue
			}

			// Broadcast decision
			bc.hub.BroadcastDecision(decision)

			// Broadcast position if exists
			if pos := bc.bot.GetPosition(); pos != nil {
				bc.hub.BroadcastPosition(pos)
			}

			// Broadcast PnL
			bc.hub.BroadcastPnL(websocket.PnLData{
				Balance:  bc.bot.GetBalance(),
				TotalPnL: bc.bot.GetTotalPnL(),
			})
		}

		// Simulate real-time delay in replay mode
		if bc.replayMode {
			time.Sleep(100 * time.Millisecond)
		}
	}

	log.Println("Finished processing all candles")
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
func (bc *BotController) Configure(symbol string, timeframe exchange.Timeframe, strategy string, replayMode bool) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if bc.isRunning {
		return fmt.Errorf("cannot configure while bot is running")
	}

	bc.symbol = symbol
	bc.timeframe = timeframe
	bc.strategyName = strategy
	bc.replayMode = replayMode

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
				Symbol     string `json:"symbol" binding:"required"`
				Timeframe  string `json:"timeframe" binding:"required"`
				Strategy   string `json:"strategy" binding:"required"`
				ReplayMode bool   `json:"replay_mode"`
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

			if err := bc.Configure(req.Symbol, timeframe, req.Strategy, req.ReplayMode); err != nil {
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
}
