package research

import (
	"candlecore/internal/exchange"
	"math"
)

// LabelOutcome simulates forward results using the Triple Barrier Method
func LabelOutcome(candles []exchange.Candle, startIndex int, side string, atr float64) OutcomeLabel {
	if atr == 0 { return OutcomeLabel{} }
	
	entryCandle := candles[startIndex]
	entryPrice := entryCandle.Close
	
	// Define Dynamic Barriers (Risk = 1.5 * ATR, Reward = 3.0 * ATR for 2R profile)
	risk := 1.5 * atr
	reward := 3.0 * atr // 2R target
	
	var tpPrice, slPrice float64
	if side == "long" {
		tpPrice = entryPrice + reward
		slPrice = entryPrice - risk
	} else {
		tpPrice = entryPrice - reward
		slPrice = entryPrice + risk
	}

	// Simulation Window: Max 48 candles (12 hours on M15)
	maxLookahead := 48
	mfe := 0.0
	mae := 1000.0 // Large default to minimize
	
	for j := 1; j <= maxLookahead; j++ {
		idx := startIndex + j
		if idx >= len(candles) { break }
		
		c := candles[idx]
		
		// 1. Check Max Excursions
		if side == "long" {
			mfe = math.Max(mfe, c.High - entryPrice)
			mae = math.Min(mae, c.Low - entryPrice)
			
			// Check Barriers
			if c.Low <= slPrice {
				return OutcomeLabel{MAE: slPrice - entryPrice, MFE: mfe, HitBarrier: "sl", TimeElapsed: j * 15}
			}
			if c.High >= tpPrice {
				return OutcomeLabel{MAE: mae, MFE: tpPrice - entryPrice, HitBarrier: "tp", TimeElapsed: j * 15}
			}
		} else {
			mfe = math.Max(mfe, entryPrice - c.Low)
			mae = math.Min(mae, entryPrice - c.High)
			
			if c.High >= slPrice {
				return OutcomeLabel{MAE: entryPrice - slPrice, MFE: mfe, HitBarrier: "sl", TimeElapsed: j * 15}
			}
			if c.Low <= tpPrice {
				return OutcomeLabel{MAE: mae, MFE: entryPrice - tpPrice, HitBarrier: "tp", TimeElapsed: j * 15}
			}
		}
	}

	return OutcomeLabel{MFE: mfe, MAE: mae, HitBarrier: "time", TimeElapsed: maxLookahead * 15}
}
