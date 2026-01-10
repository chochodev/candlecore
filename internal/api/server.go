package api

import (
	"candlecore/internal/exchange"
	ws "candlecore/internal/websocket"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Server represents the API server
type Server struct {
	router     *gin.Engine
	dataDir    string
	hub        *ws.Hub
	controller *BotController
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
	controller := NewBotController(provider, hub)
	
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
