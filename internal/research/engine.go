package research

import (
	"candlecore/internal/indicators"
	"fmt"
	"time"
)

// GenerateDataset creates a full statistical research dataset for a symbol
func (dt *DataTransformer) GenerateDataset(symbol string) ([]ResearchRow, error) {
	// 1. Load H1 Trend Context
	h1Candles, _ := dt.provider.GetCandles(symbol, "1h", 0)
	hCloses := extractCloses(h1Candles)
	
	h1EMA50, _ := indicators.EMA(hCloses, 50)
	h1EMA200, _ := indicators.EMA(hCloses, 200)

	// Since indicators.EMA returns result[period-1:], we must offset
	offset50 := 50 - 1
	offset200 := 200 - 1

	h1Context := make(map[int64]string)
	for i := 0; i < len(h1Candles); i++ {
		// Only check trend if we have enough data for BOTH EMAs
		if i < offset200 { continue }
		
		idx50 := i - offset50
		idx200 := i - offset200
		
		state := "bearish"
		if h1EMA50[idx50] > h1EMA200[idx200] { state = "bullish" }
		h1Context[h1Candles[i].Timestamp.Unix()] = state
	}

	// 2. Load M15 Data
	m15Candles, err := dt.provider.GetCandles(symbol, "15m", 0)
	if err != nil { return nil, err }

	var results []ResearchRow

	// 3. Process every candle (leaving room for labels and indicators)
	for i := 100; i < len(m15Candles)-100; i++ {
		current := m15Candles[i]
		
		// Align Trend
		h1Ts := current.Timestamp.Truncate(time.Hour).Add(-time.Hour).Unix()
		trend, ok := h1Context[h1Ts]
		if !ok { continue }

		// Extract Local Features
		features, ok := ComputeM15Features(m15Candles[0 : i+1])
		if !ok { continue }
		features.H1Trend = trend
		features.Session = identifySession(current.Timestamp)

		// Label Outcomes (Both Long and Short possibilities)
		labelLong := LabelOutcome(m15Candles, i, "long", features.ATR)
		
		results = append(results, ResearchRow{
			Features: features,
			Labels:   labelLong,
		})
	}

	fmt.Printf("Research Complete: Generated %d data points for %s\n", len(results), symbol)
	return results, nil
}
