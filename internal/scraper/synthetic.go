package scraper

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"path/filepath"
	"time"

	"candlecore/internal/engine"
)

// SyntheticDataGenerator creates realistic crypto price data
type SyntheticDataGenerator struct {
	dataDir string
	seed    int64
}

// NewSyntheticDataGenerator creates a new generator
func NewSyntheticDataGenerator(dataDir string) *SyntheticDataGenerator {
	return &SyntheticDataGenerator{
		dataDir: dataDir,
		seed:    time.Now().UnixNano(),
	}
}

// GenerateRealisticData creates multi-year realistic crypto data
func (g *SyntheticDataGenerator) GenerateRealisticData(ctx context.Context, symbol, interval string, years int) error {
	coinID, ok := g.getCoinID(symbol)
	if !ok {
		return fmt.Errorf("unsupported symbol: %s", symbol)
	}
	
	candlesPerDay := g.getCandlesPerDay(interval)
	if candlesPerDay == 0 {
		return fmt.Errorf("unsupported interval: %s", interval)
	}
	
	totalCandles := years * 365 * candlesPerDay
	
	fmt.Printf("Generating %d years of %s %s data...\n", years, symbol, interval)
	fmt.Printf("Total candles: %d\n\n", totalCandles)
	
	candles := g.generatePriceData(symbol, interval, years, candlesPerDay)
	
	filename := filepath.Join(g.dataDir, fmt.Sprintf("%s_%s.csv", coinID, interval))
	
	scraper := &BinanceBulkScraper{dataDir: g.dataDir}
	if err := scraper.writeCandles(filename, candles); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}
	
	fmt.Printf("Generated %d candles for %s %s\n", len(candles), symbol, interval)
	fmt.Printf("Saved to: %s\n", filename)
	fmt.Printf("Date range: %s to %s\n\n",
		candles[0].Timestamp.Format("2006-01-02"),
		candles[len(candles)-1].Timestamp.Format("2006-01-02"),
	)
	
	return nil
}

// generatePriceData creates realistic OHLCV data
func (g *SyntheticDataGenerator) generatePriceData(symbol, interval string, years, candlesPerDay int) []engine.Candle {
	rand.Seed(g.seed)
	
	// Starting parameters based on symbol
	var basePrice, volatility, trend float64
	switch symbol {
	case "BTCUSDT":
		basePrice = 10000.0
		volatility = 0.03
		trend = 0.15 // 15% annual growth
	case "ETHUSDT":
		basePrice = 200.0
		volatility = 0.04
		trend = 0.20
	default:
		basePrice = 100.0
		volatility = 0.02
		trend = 0.10
	}
	
	totalCandles := years * 365 * candlesPerDay
	candles := make([]engine.Candle, totalCandles)
	
	startDate := time.Now().AddDate(-years, 0, 0)
	intervalMinutes := g.getIntervalMinutes(interval)
	
	currentPrice := basePrice
	
	for i := 0; i < totalCandles; i++ {
		timestamp := startDate.Add(time.Duration(i*intervalMinutes) * time.Minute)
		
		// Trend component (gradual upward movement)
		trendComponent := trend / float64(365*candlesPerDay)
		
		// Random walk component
		randomWalk := (rand.Float64() - 0.5) * volatility
		
		// Cyclical component (simulate bull/bear cycles)
		cycle := math.Sin(float64(i) / float64(365*candlesPerDay) * 2 * math.Pi) * 0.1
		
		// Calculate price change
		priceChange := currentPrice * (trendComponent + randomWalk + cycle)
		currentPrice += priceChange
		
		// Ensure price doesn't go negative
		if currentPrice < basePrice*0.1 {
			currentPrice = basePrice * 0.1
		}
		
		// Generate OHLC with realistic intrabar movement
		candleVolatility := currentPrice * volatility * 0.5
		
		open := currentPrice
		close := currentPrice + (rand.Float64()-0.5)*candleVolatility
		high := math.Max(open, close) + rand.Float64()*candleVolatility*0.5
		low := math.Min(open, close) - rand.Float64()*candleVolatility*0.5
		
		// Generate realistic volume (higher volatility = higher volume)
		baseVolume := 1000000.0
		volumeVariation := math.Abs(close-open) / open
		volume := baseVolume * (1 + volumeVariation*10) * (0.5 + rand.Float64())
		
		candles[i] = engine.Candle{
			Timestamp: timestamp,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
		}
		
		currentPrice = close
	}
	
	return candles
}

func (g *SyntheticDataGenerator) getCoinID(symbol string) (string, bool) {
	coins := map[string]string{
		"BTCUSDT": "bitcoin",
		"ETHUSDT": "ethereum",
	}
	id, ok := coins[symbol]
	return id, ok
}

func (g *SyntheticDataGenerator) getCandlesPerDay(interval string) int {
	intervals := map[string]int{
		"5m":  288,  // 24*60/5
		"15m": 96,   // 24*60/15
		"1h":  24,   // 24
		"4h":  6,    // 24/4
		"1d":  1,    // 1
	}
	return intervals[interval]
}

func (g *SyntheticDataGenerator) getIntervalMinutes(interval string) int {
	intervals := map[string]int{
		"5m":  5,
		"15m": 15,
		"1h":  60,
		"4h":  240,
		"1d":  1440,
	}
	return intervals[interval]
}
