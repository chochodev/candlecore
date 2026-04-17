package research

import (
	"candlecore/internal/exchange"
	"math"
)

// ComputeM15Features calculates technical signatures for a single candle within a window
func ComputeM15Features(window []exchange.Candle) (FeatureSet, bool) {
	if len(window) < 30 { return FeatureSet{}, false }
	
	current := window[len(window)-1]
	
	// 1. Calculate ATR (Simple version for features)
	totalTR := 0.0
	for i := len(window)-14; i < len(window); i++ {
		tr := math.Max(window[i].High-window[i].Low, 
			  math.Max(math.Abs(window[i].High-window[i-1].Close), 
			  math.Abs(window[i].Low-window[i-1].Close)))
		totalTR += tr
	}
	atr := totalTR / 14
	
	// 2. Identify Candle Type (Engulfing logic)
	prev := window[len(window)-2]
	candleType := "normal"
	if current.Close > current.Open && prev.Close < prev.Open && 
	   current.Close > prev.Open && current.Open < prev.Close {
		candleType = "engulfing_bull"
	} else if current.Close < current.Open && prev.Close > prev.Open && 
	   current.Close < prev.Open && current.Open > prev.Close {
		candleType = "engulfing_bear"
	}

	// 3. Range Expansion (Current vs 10-period average)
	avgRange := 0.0
	for i := len(window)-11; i < len(window)-1; i++ {
		avgRange += (window[i].High - window[i].Low)
	}
	avgRange /= 10
	expansion := (current.High - current.Low) / avgRange

	// 4. Distance from EMA 50 (Normalized by ATR)
	// We'll use a simplified EMA 50 for the feature engine
	ema50 := 0.0
	for _, c := range window[len(window)-50:] { ema50 += c.Close }
	ema50 /= 50
	dist := (current.Close - ema50) / atr
	
	// Categorize Pullback Depth
	absDist := math.Abs(dist)
	depth := "moderate"
	if absDist <= 0.75 { depth = "deep" }
	if absDist >= 2.0 { depth = "extreme" }

	// 5. Relative Volume (Current vs 20-period average)
	avgVol := 0.0
	for _, c := range window[len(window)-21 : len(window)-1] { avgVol += c.Volume }
	avgVol /= 20
	relVol := current.Volume / avgVol
	
	volSpike := "normal"
	if relVol >= 1.5 { volSpike = "high_vol" }
	if relVol <= 0.6 { volSpike = "low_vol" }

	// 6. Break of Structure (Simplified: break of previous 10-bar high/low)
	bos := "none"
	prevHigh := 0.0
	prevLow := 10000000.0
	for _, c := range window[len(window)-11 : len(window)-1] {
		if c.High > prevHigh { prevHigh = c.High }
		if c.Low < prevLow { prevLow = c.Low }
	}
	if current.Close > prevHigh { bos = "high_break" }
	if current.Close < prevLow { bos = "low_break" }

	return FeatureSet{
		Timestamp:        current.Timestamp,
		CandleType:       candleType,
		ATR:              atr,
		RangeExpansion:   expansion,
		PriceDistEMA50:   dist,
		PullbackDepth:    depth,
		VolSpike:         volSpike,
		BreakOfStructure: bos,
	}, true
}
