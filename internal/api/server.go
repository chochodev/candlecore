package api

import (
	"net/http"
	"time"

	"candlecore/internal/scraper"

	"github.com/gin-gonic/gin"
)

// Server represents the API server
type Server struct {
	router  *gin.Engine
	dataDir string
}

// NewServer creates a new API server
func NewServer(dataDir string) *Server {
	gin.SetMode(gin.ReleaseMode)
	
	router := gin.Default()
	router.Use(corsMiddleware())
	
	s := &Server{
		router:  router,
		dataDir: dataDir,
	}
	
	s.setupRoutes()
	return s
}

// setupRoutes configures API endpoints
func (s *Server) setupRoutes() {
	api := s.router.Group("/api/v1")
	{
		// Data endpoints
		api.GET("/data", s.listData)
		api.GET("/data/:coin/:interval", s.getCandleData)
		
		// Backtest endpoints
		api.POST("/backtest", s.runBacktest)
		api.GET("/backtest/results/:id", s.getBacktestResults)
		
		// Health check
		api.GET("/health", s.healthCheck)
	}
}

// Run starts the API server
func (s *Server) Run(port string) error {
	return s.router.Run(":" + port)
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

// DataListResponse represents available data files
type DataListResponse struct {
	Files []DataFileInfo `json:"files"`
	Total int            `json:"total"`
}

// DataFileInfo represents a single data file
type DataFileInfo struct {
	CoinID       string    `json:"coin_id"`
	Interval     string    `json:"interval"`
	TotalCandles int       `json:"total_candles"`
	FirstDate    time.Time `json:"first_date"`
	LastDate     time.Time `json:"last_date"`
	FileSizeKB   float64   `json:"file_size_kb"`
	FilePath     string    `json:"file_path"`
}

// listData returns all available data files
func (s *Server) listData(c *gin.Context) {
	scraper := scraper.NewDataScraper(s.dataDir)
	info, err := scraper.GetDataInfo()
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get data info",
		})
		return
	}
	
	files := make([]DataFileInfo, 0, len(info))
	for _, data := range info {
		files = append(files, DataFileInfo{
			CoinID:       data.CoinID,
			Interval:     "daily",
			TotalCandles: data.TotalCandles,
			FirstDate:    data.FirstDate,
			LastDate:     data.LastDate,
			FileSizeKB:   float64(data.FileSize) / 1024,
			FilePath:     data.FilePath,
		})
	}
	
	c.JSON(http.StatusOK, DataListResponse{
		Files: files,
		Total: len(files),
	})
}

// CandleResponse represents candle data
type CandleResponse struct {
	Timestamp time.Time `json:"timestamp"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
}

// getCandleData returns candle data for a coin
func (s *Server) getCandleData(c *gin.Context) {
	coinID := c.Param("coin")
	interval := c.Param("interval")
	
	scraper := scraper.NewDataScraper(s.dataDir)
	candles, err := scraper.GetCoinData(coinID)
	
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Data not found for " + coinID,
		})
		return
	}
	
	// Convert to response format
	response := make([]CandleResponse, 0, len(candles))
	for _, candle := range candles {
		response = append(response, CandleResponse{
			Timestamp: candle.Timestamp,
			Open:      candle.Open,
			High:      candle.High,
			Low:       candle.Low,
			Close:     candle.Close,
			Volume:    candle.Volume,
		})
	}
	
	c.JSON(http.StatusOK, gin.H{
		"coin":     coinID,
		"interval": interval,
		"total":    len(response),
		"candles":  response,
	})
}

// BacktestRequest represents a backtest request
type BacktestRequest struct {
	CoinID         string  `json:"coin_id" binding:"required"`
	Interval       string  `json:"interval" binding:"required"`
	Strategy       string  `json:"strategy" binding:"required"`
	InitialBalance float64 `json:"initial_balance"`
	FastPeriod     int     `json:"fast_period"`
	SlowPeriod     int     `json:"slow_period"`
	PositionSize   float64 `json:"position_size"`
}

// runBacktest executes a backtest
func (s *Server) runBacktest(c *gin.Context) {
	var req BacktestRequest
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}
	
	// TODO: Implement actual backtest execution
	// For now, return a mock response
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Backtest queued",
		"id":      "backtest-123",
		"status":  "pending",
	})
}

// getBacktestResults returns backtest results
func (s *Server) getBacktestResults(c *gin.Context) {
	id := c.Param("id")
	
	// TODO: Implement result retrieval
	
	c.JSON(http.StatusOK, gin.H{
		"id":     id,
		"status": "completed",
		"results": gin.H{
			"initial_balance": 10000.0,
			"final_balance":   12500.0,
			"total_pnl":       2500.0,
			"total_trades":    15,
			"win_rate":        0.67,
		},
	})
}

// healthCheck returns server health status
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"version": "1.0.0",
		"time":    time.Now(),
	})
}
