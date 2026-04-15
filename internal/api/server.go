package api

import (
	"candlecore/internal/exchange"
	"candlecore/internal/strategies"
	ws "candlecore/internal/websocket"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Server represents the API server
type Server struct {
	router     *gin.Engine
	dataDir    string
	hub        *ws.Hub
	controller *BotController
	httpServer *http.Server
}

// NewServer creates a new API server
func NewServer(dataDir string) *Server {
	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()
	router.Use(corsMiddleware())

	// Create WebSocket hub
	hub := ws.NewHub()
	go hub.Run()

	// Create exchange provider
	provider := exchange.NewLocalFileProvider(dataDir)

	// Create bot controller
	controller := NewBotController(provider, hub, dataDir)

	s := &Server{
		router:     router,
		dataDir:    dataDir,
		hub:        hub,
		controller: controller,
	}

	s.setupRoutes()
	controller.SetupRoutes(router)

	return s
}

// setupRoutes configures API endpoints
func (s *Server) setupRoutes() {
	api := s.router.Group("/api/v1")
	{
		// Health check
		api.GET("/health", s.healthCheck)

		// Available symbols and timeframes
		api.GET("/symbols", s.getSymbols)
		api.GET("/timeframes", s.getTimeframes)
		api.GET("/data/range", s.getDataRange)

		// Strategy Reports
		api.GET("/strategies/reports", s.getStrategyReports)
		api.POST("/backtest/fast", s.handleFastBacktest)

		// Available Strategies
		api.GET("/strategies", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"strategies": strategies.ListStrategies(),
			})
		})
	}
}

// Run starts the API server
func (s *Server) Run(port string) error {
	s.httpServer = &http.Server{
		Addr:    ":" + port,
		Handler: s.router,
	}

	return s.httpServer.ListenAndServe()
}

// Stop stops the API server and its components
func (s *Server) Stop() error {
	// 1. Stop the bot if it's running
	if s.controller != nil {
		s.controller.Stop()
	}

	// 2. Shut down the HTTP server
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(ctx)
	}

	return nil
}

// corsMiddleware enables CORS for frontend access
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// getSymbols returns available trading pairs
func (s *Server) getSymbols(c *gin.Context) {
	provider := exchange.NewLocalFileProvider(s.dataDir)
	symbols := provider.GetSupportedSymbols()

	c.JSON(http.StatusOK, gin.H{
		"symbols": symbols,
	})
}

// getTimeframes returns supported timeframes
func (s *Server) getTimeframes(c *gin.Context) {
	provider := exchange.NewLocalFileProvider(s.dataDir)
	timeframes := provider.GetSupportedTimeframes()

	tfStrings := make([]string, len(timeframes))
	for i, tf := range timeframes {
		tfStrings[i] = string(tf)
	}

	c.JSON(http.StatusOK, gin.H{
		"timeframes": tfStrings,
	})
}

// getDataRange returns available candle start/end range for symbol+timeframe.
func (s *Server) getDataRange(c *gin.Context) {
	symbol := c.Query("symbol")
	tfStr := c.Query("timeframe")
	if symbol == "" || tfStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol and timeframe query params are required"})
		return
	}

	timeframe := exchange.Timeframe(tfStr)
	provider := exchange.NewLocalFileProvider(s.dataDir)
	candles, err := provider.GetCandles(symbol, timeframe, 0)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(candles) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no candles found"})
		return
	}

	start := candles[0].Timestamp
	end := candles[len(candles)-1].Timestamp
	c.JSON(http.StatusOK, gin.H{
		"symbol":        symbol,
		"timeframe":     tfStr,
		"start_time":    start.Unix(),
		"end_time":      end.Unix(),
		"start_date":    start,
		"end_date":      end,
		"total_candles": len(candles),
	})
}

// healthCheck returns server health status
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"version": "2.0.0",
		"time":    time.Now(),
		"features": []string{
			"websocket_streaming",
			"bot_control",
			"historical_replay",
			"multi_timeframe",
			"indicators",
		},
	})
}

// getStrategyReports returns all versioned reports from changelog/
func (s *Server) getStrategyReports(c *gin.Context) {
	changelogDir := "changelog"
	files, err := os.ReadDir(changelogDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read reports directory"})
		return
	}

	var reports []interface{}
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(changelogDir, file.Name()))
		if err != nil {
			continue
		}

		var report interface{}
		if err := json.Unmarshal(data, &report); err == nil {
			reports = append(reports, report)
		}
	}

	c.JSON(http.StatusOK, reports)
}
