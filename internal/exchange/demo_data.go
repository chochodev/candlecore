package exchange

import (
	"math"
	"time"
)

// GenerateDemoCandles creates synthetic candles with guaranteed MA crossover
// This ensures the bot will make trades during demo/testing
func GenerateDemoCandles() []Candle {
	baseTime := time.Now().Add(-time.Hour * 24) // Start 24 hours ago
	basePrice := 45000.0
	candles := make([]Candle, 0, 100)

	// Phase 1: Downtrend (candles 0-30) - Fast MA below Slow MA
	for i := 0; i < 30; i++ {
		price := basePrice - float64(i)*50 // Gradual decline
		candles = append(candles, Candle{
			Timestamp: baseTime.Add(time.Duration(i) * time.Minute * 5),
			Open:      price + 20,
			High:      price + 50,
			Low:       price - 30,
			Close:     price,
			Volume:    100 + float64(i)*5,
		})
	}

	// Phase 2: GOLDEN CROSS SETUP (candles 30-45) - Price starts rising
	for i := 30; i < 45; i++ {
		price := basePrice - float64(30-i)*80 // Sharp rise
		candles = append(candles, Candle{
			Timestamp: baseTime.Add(time.Duration(i) * time.Minute * 5),
			Open:      price - 20,
			High:      price + 80,
			Low:       price - 10,
			Close:     price,
			Volume:    150 + float64(i)*10,
		})
	}

	// Phase 3: Uptrend continues (candles 45-65) - Fast MA crosses above Slow MA
	// This is where BUY signal happens!
	for i := 45; i < 65; i++ {
		price := basePrice + float64(i-30)*60 // Strong uptrend
		candles = append(candles, Candle{
			Timestamp: baseTime.Add(time.Duration(i) * time.Minute * 5),
			Open:      price - 15,
			High:      price + 100,
			Low:       price - 5,
			Close:     price,
			Volume:    200 + float64(i)*15,
		})
	}

	// Phase 4: Peak and reversal (candles 65-80) - Price tops out
	for i := 65; i < 80; i++ {
		price := basePrice + float64(80-i)*40 // Gradual decline from peak
		wobble := math.Sin(float64(i)) * 50    // Add some volatility
		candles = append(candles, Candle{
			Timestamp: baseTime.Add(time.Duration(i) * time.Minute * 5),
			Open:      price + wobble,
			High:      price + 80,
			Low:       price - 60,
			Close:     price + wobble/2,
			Volume:    180 - float64(i-65)*5,
		})
	}

	// Phase 5: DEATH CROSS SETUP (candles 80-100) - Sharp decline
	// This is where SELL signal happens!
	for i := 80; i < 100; i++ {
		price := basePrice - float64(i-80)*90 // Sharp fall
		candles = append(candles, Candle{
			Timestamp: baseTime.Add(time.Duration(i) * time.Minute * 5),
			Open:      price + 30,
			High:      price + 40,
			Low:       price - 100,
			Close:     price - 40,
			Volume:    250 - float64(i-80)*8,
		})
	}

	return candles
}

// GetDemoDataProvider returns demo candles
// This is a simple wrapper that returns synthetic candles for guaranteed trades
func GetDemoDataProvider() *DemoProvider {
	return &DemoProvider{
		candles: GenerateDemoCandles(),
	}
}

// DemoProvider holds demo candles
type DemoProvider struct {
	candles []Candle
}

// GetCandles returns demo candles
func (p *DemoProvider) GetCandles(symbol string, timeframe Timeframe, limit int) ([]Candle, error) {
	if limit > 0 && limit < len(p.candles) {
		return p.candles[:limit], nil
	}
	return p.candles, nil
}

// Implement required interface methods (stubs)
func (p *DemoProvider) GetSupportedSymbols() ([]string, error) {
	return []string{"bitcoin"}, nil
}

func (p *DemoProvider) GetSupportedTimeframes() ([]Timeframe, error) {
	return []Timeframe{Timeframe5m, Timeframe1h}, nil
}
