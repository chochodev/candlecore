package research

import (
	"candlecore/internal/exchange"
	"candlecore/internal/indicators"
	"fmt"
	"time"
)

// DataTransformer aligns multiple timeframes and computes features
type DataTransformer struct {
	provider exchange.DataProvider
}

func NewDataTransformer(provider exchange.DataProvider) *DataTransformer {
	return &DataTransformer{provider: provider}
}

// TransformSOL generates a ResearchRow dataset from SOL data
func (dt *DataTransformer) TransformSOL() ([]ResearchRow, error) {
	// 1. Load H1 Trend Context
	h1Candles, err := dt.provider.GetCandles("sol", "1h", 0)
	if err != nil {
		return nil, fmt.Errorf("failed to load h1 data: %w", err)
	}

	// 2. Compute H1 Indicators (Macro Trend)
	h1Closes := extractCloses(h1Candles)
	h1EMA50, _ := indicators.EMA(h1Closes, 50)
	h1EMA200, _ := indicators.EMA(h1Closes, 200)

	// Build a map of H1 context by timestamp for O(1) alignment
	h1Context := make(map[int64]struct {
		bullish bool
		ema50   float64
	})
	
	for i := 0; i < len(h1Candles); i++ {
		if h1EMA200[i] == 0 { continue }
		h1Context[h1Candles[i].Timestamp.Unix()] = struct {
			bullish bool
			ema50   float64
		}{
			bullish: h1EMA50[i] > h1EMA200[i],
			ema50:   h1EMA50[i],
		}
	}

	// 3. Load M15 Execution Data
	m15Candles, err := dt.provider.GetCandles("sol", "15m", 0)
	if err != nil {
		return nil, fmt.Errorf("failed to load m15 data: %w", err)
	}

	// 4. Align and Extract Features
	var dataset []ResearchRow
	for i := 200; i < len(m15Candles); i++ {
		current := m15Candles[i]
		
		// Find corresponding H1 context (using the previous hour's candle to prevent look-ahead bias)
		h1Ts := current.Timestamp.Truncate(1 * time.Hour).Add(-1 * time.Hour).Unix()
		ctx, ok := h1Context[h1Ts]
		if !ok { continue }

		trend := "bearish"
		if ctx.bullish { trend = "bullish" }

		row := ResearchRow{
			Features: FeatureSet{
				Timestamp: current.Timestamp,
				Symbol:    "sol",
				H1Trend:   trend,
				Session:   identifySession(current.Timestamp),
				// More features will be added in Step 3
			},
		}
		dataset = append(dataset, row)
	}

	return dataset, nil
}

// Helpers
func extractCloses(candles []exchange.Candle) []float64 {
	closes := make([]float64, len(candles))
	for i, c := range candles { closes[i] = c.Close }
	return closes
}

func identifySession(t time.Time) string {
	hour := t.Hour()
	switch {
	case hour >= 0 && hour < 8: return "Asia"
	case hour >= 8 && hour < 13: return "London"
	case hour >= 13 && hour < 21: return "NY"
	default: return "Crossover"
	}
}
