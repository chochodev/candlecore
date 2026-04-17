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

// defaultSimBalance is the replay/paper wallet and bot equity starting point for the dashboard simulation.
const defaultSimBalance = 25.0

// BotController manages bot lifecycle and configuration
type BotController struct {
	wallet         *exchange.VirtualWallet // SHARED WALLET for all bots
	bots           map[string]*bot.Bot     // Fleet of bots (symbol -> bot)
	hub            *websocket.Hub
	provider       exchange.DataProvider
	isRunning      bool
	replayMode     bool
	replaySpeed    time.Duration // Delay between candles (default 1s)
	dryRun         bool          // Dry-run mode flag
	symbols        []string      // Multi-symbol support
	timeframe      exchange.Timeframe
	strategyName   string
	strategyParams map[string]interface{}
	skipSignal     chan struct{} // Channel to trigger fast-forward skip
	isSkipping     bool          // Stateful skip flag
	configManager  *config.ConfigManager
	mu             sync.RWMutex
	stopChan       chan struct{}
	currentCandleIdx int    // PERSISTENT INDEX for resume
	isPaused         bool   // PAUSE STATE flag
	startTime        int64  // Backtest start (Unix)
	endTime          int64  // Backtest end (Unix)
}

// NewBotController creates a new bot controller
func NewBotController(provider exchange.DataProvider, hub *websocket.Hub, dataDir string) *BotController {
	cManager := config.NewConfigManager(dataDir)
	return &BotController{
		provider:       provider,
		hub:            hub,
		configManager:  cManager,
		wallet:         exchange.NewVirtualWallet(defaultSimBalance),
		isRunning:      false,
		replayMode:     true,                     // Default to replay mode
		replaySpeed:    200 * time.Millisecond,   // Faster default for research
		dryRun:         true,                     // Default to dry-run for safety
		symbols:        []string{"sol", "btc", "eth"},
		bots:           make(map[string]*bot.Bot),
		timeframe:      "15m",
		strategyName:   "vanguard_m15",
		stopChan:       make(chan struct{}),
		skipSignal:     make(chan struct{}, 1), // Buffered for 1 skip pulse
		currentCandleIdx: 0,
		isPaused:         false,
	}
}

// saveProfile internal helper to persist current state
func (bc *BotController) saveProfile() {
	profile := config.BotProfile{
		Symbol:      bc.symbols[0],
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

// Start starts the bot fleet
func (bc *BotController) Start() error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if bc.isRunning {
		return fmt.Errorf("bot fleet is already running")
	}

	// Initialize bots for each symbol
	if len(bc.bots) == 0 {
		for _, sym := range bc.symbols {
			strategy, err := strategies.GetStrategy(bc.strategyName)
			if err != nil {
				log.Printf("Failed to create strategy for %s: %v", sym, err)
				continue
			}

			if bc.strategyParams != nil {
				strategy.Configure(bc.strategyParams)
			}

			bc.bots[sym] = bot.NewBot(strategy, bc.provider, bot.Config{
				Symbol:         sym,
				Timeframe:      bc.timeframe,
				InitialBalance: defaultSimBalance,
				PositionSize:   10,
			})
		}
		bc.currentCandleIdx = 0
	}

	bc.isRunning = true
	bc.isPaused = false
	bc.stopChan = make(chan struct{})

	// Start processing
	go bc.run()

	bc.saveProfile()
	bc.hub.BroadcastStatus("started")
	log.Printf("Fleet started: symbols=%v, timeframe=%s, strategy=%s", bc.symbols, bc.timeframe, bc.strategyName)

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
	bc.isPaused = true
	bc.saveProfile() // Persist stop state
	bc.hub.BroadcastStatus("stopped")

	log.Printf("Bot paused at candle index %d", bc.currentCandleIdx)
	return nil
}

// Reset resets the bot state (balance, trades, current index)
func (bc *BotController) Reset() error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if bc.isRunning {
		return fmt.Errorf("cannot reset while bot is running")
	}

	bc.wallet = exchange.NewVirtualWallet(defaultSimBalance)
	bc.isSkipping = false
	bc.currentCandleIdx = 0
	bc.bots = make(map[string]*bot.Bot) // Force re-creation on next Start
	bc.hub.BroadcastStatus("reset")
	
	log.Println("Bot state reset to initial parameters")
	return nil
}

// run processes candles and executes strategy
func (bc *BotController) run() {
	log.Printf("Starting bot fleet: symbols=%v, timeframe=%s, strategy=%s, replay=%v",
		bc.symbols, bc.timeframe, bc.strategyName, bc.replayMode)

	// Load data for all symbols
	fleetCandles := make(map[string][]exchange.Candle)
	maxLen := 0
	for _, sym := range bc.symbols {
		candles, err := bc.provider.GetCandles(sym, bc.timeframe, 0)
		if err != nil {
			log.Printf("Error fetching candles for %s: %s", sym, err)
			continue
		}
		
		// Filter candles
		var filtered []exchange.Candle
		for _, c := range candles {
			t := c.Timestamp.Unix()
			if bc.startTime > 0 && t < bc.startTime { continue }
			if bc.endTime > 0 && t > bc.endTime { break }
			filtered = append(filtered, c)
		}
		fleetCandles[sym] = filtered
		if len(filtered) > maxLen { maxLen = len(filtered) }
	}

	if maxLen == 0 {
		log.Printf("CRITICAL ERROR: No data loaded for any symbols.")
		bc.hub.BroadcastStatus("failed: data_missing")
		bc.Stop() 
		return
	}

	log.Printf("Fleet Sync: Loaded data for %d symbols. Max sequence: %d candles", len(fleetCandles), maxLen)

	// Context buffer for UI
	var jumpContext []websocket.CandleData

	// Fleet processing loop
	for i := bc.currentCandleIdx; i < maxLen; i++ {
		bc.mu.Lock()
		bc.currentCandleIdx = i
		bc.mu.Unlock()

		select {
		case <-bc.stopChan: return
		default:
		}

		bc.mu.RLock()
		skip := bc.isSkipping
		bc.mu.RUnlock()

		// Process each bot in the fleet
		for sym, candles := range fleetCandles {
			if i >= len(candles) { continue }
			candle := candles[i]
			botInstance, ok := bc.bots[sym]
			if !ok { continue }

			cData := websocket.CandleData{
				Symbol:    sym,
				Timeframe: string(bc.timeframe),
				Timestamp: candle.Timestamp,
				Open:      candle.Open,
				Close:     candle.Close,
				Indicators: make(map[string]float64),
			}

			if i >= 200 {
				decision, err := botInstance.ProcessCandle(candle)
				if err == nil {
					if decision.Signal != "hold" {
						if skip {
							bc.mu.Lock()
							bc.isSkipping = false
							bc.mu.Unlock()
							skip = false
							bc.hub.BroadcastStatus("resumed")
						}
						bc.hub.BroadcastDecision(decision)
					}
					
					// Broadcast portfolio status from master wallet
					if !skip && sym == bc.symbols[0] { // Use first symbol for balance updates
						bc.hub.BroadcastPnL(websocket.PnLData{
							Balance:  botInstance.GetBalance(),
							TotalPnL: botInstance.GetTotalPnL(),
							Trades:   botInstance.GetTrades(),
						})
					}
					
					if !skip {
						if pos := botInstance.GetPosition(); pos != nil {
							bc.hub.BroadcastPosition(pos)
						}
					}
				}
			}

			if !skip {
				bc.hub.BroadcastCandle(candle, sym, string(bc.timeframe), cData.Indicators)
			}
			
			// Only keep context for the primary symbol
			if sym == bc.symbols[0] {
				jumpContext = append(jumpContext, cData)
				if len(jumpContext) > 200 { jumpContext = jumpContext[1:] }
			}
		}

		if !skip {
			time.Sleep(bc.replaySpeed)
		}
	}
	bc.mu.Lock()
	wasSkipping := bc.isSkipping
	bc.isSkipping = false
	bc.isRunning = false
	bc.mu.Unlock()

	if wasSkipping {
		bc.hub.BroadcastHistory(jumpContext)
	}
	bc.hub.BroadcastStatus("completed")
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
		"paused":       bc.isPaused, 
		"symbols":      bc.symbols,
		"timeframe":    bc.timeframe,
		"strategy":     bc.strategyName,
		"replay_mode":  bc.replayMode,
		"replay_speed": bc.replaySpeed.Seconds(),
		"dry_run":      bc.dryRun,
		"balance":      bc.wallet.GetBalance(),
		"total_pnl":    bc.wallet.GetTotalPnL(),
		"trades_count": len(bc.wallet.GetTrades()),
	}
	return status
}

func (bc *BotController) Configure(symbol string, timeframe exchange.Timeframe, strategy string, replayMode, dryRun bool, replaySpeed float64, strategyParams map[string]interface{}) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	if bc.isRunning { return fmt.Errorf("cannot configure while running") }
	bc.symbols = []string{symbol, "btc", "eth"}
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
			var config struct {
				Symbol       string                 `json:"symbol"`
				Timeframe    string                 `json:"timeframe"`
				Strategy     string                 `json:"strategy"`
				ReplayMode   bool                   `json:"replay_mode"`
				DryRun       bool                   `json:"dry_run"`
				ReplaySpeed  float64                `json:"replay_speed"`
				StartTime    int64                  `json:"start_time"`
				EndTime      int64                  `json:"end_time"`
				Params       map[string]interface{} `json:"params"`
			}

			if err := c.ShouldBindJSON(&config); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			bc.mu.Lock()
			defer bc.mu.Unlock()

			bc.symbols = []string{config.Symbol, "btc", "eth"}
			bc.timeframe = exchange.Timeframe(config.Timeframe)
			bc.strategyName = config.Strategy
			bc.replayMode = config.ReplayMode
			bc.dryRun = config.DryRun
			bc.replaySpeed = time.Duration(config.ReplaySpeed * float64(time.Second))
			bc.startTime = config.StartTime
			bc.endTime = config.EndTime
			bc.strategyParams = config.Params
			bc.bots = make(map[string]*bot.Bot) // Force re-sync
			bc.saveProfile()
			c.JSON(200, gin.H{"status": "configured"})
		})
		api.GET("/pnl", bc.HandleGetPnL)
	}
}

func (bc *BotController) HandleGetPnL(c *gin.Context) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	vwTrades := bc.wallet.GetTrades()
	tradesCount := len(vwTrades)
	balance := bc.wallet.GetBalance()
	totalPnL := bc.wallet.GetTotalPnL()
	wins := 0
	var jsonTrades []interface{}

	for _, t := range vwTrades {
		if t.PnL > 0 { wins++ }

		investment := t.EntryPrice * t.Quantity
		var roi float64
		if investment > 0 { roi = (t.PnL / investment) * 100 }

		jsonTrades = append(jsonTrades, gin.H{
			"id":            fmt.Sprintf("TR-%d", t.OpenedAt.UnixNano()),
			"symbol":        t.Symbol,
			"side":          t.Side,
			"entry_price":   t.EntryPrice,
			"current_price": t.ExitPrice,
			"realized_pnl":  roi,
			"opened_at":     t.OpenedAt,
			"closed_at":     t.ClosedAt,
			"reasoning":     "Fleet execution",
		})
	}

	winRate := 0.0
	if tradesCount > 0 {
		winRate = (float64(wins) / float64(tradesCount)) * 100
	}

	c.JSON(200, gin.H{
		"total_pnl": totalPnL,
		"win_rate":  winRate,
		"balance":   balance,
		"trades":    jsonTrades,
	})
}

var upgrader = gorillaws.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
