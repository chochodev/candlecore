package strategy

import (
	"candlecore/internal/indicators"
	"fmt"
)

// AlphaPrimeStrategy (v1.0.7)
type AlphaPrimeStrategy struct {
	BaseStrategy
}

func NewAlphaPrime() IStrategy {
	return &AlphaPrimeStrategy{
		BaseStrategy: BaseStrategy{
			Name:    "Alpha Prime",
			Version: "1.0.7",
			Config: StrategyConfig{
				Stoploss:      -0.05,
				StakeAmount:   100.0,
				MaxOpenTrades: 3,
				Timeframe:     "1h",
			},
		},
	}
}

func init() {
	Register("alpha_prime", NewAlphaPrime)
}

func (s *AlphaPrimeStrategy) PopulateIndicators(df *DataFrame) error {
	if len(df.Candles) < 50 {
		return fmt.Errorf("insufficient data: need 50 candles")
	}
	if df.Indicators == nil { df.Indicators = make(map[string][]float64) }

	closes := ExtractCloses(df.Candles)
	highs, lows := ExtractHighs(df.Candles), ExtractLows(df.Candles)

	ema8, _ := indicators.EMA(closes, 8)
	sma12, _ := indicators.SMA(closes, 12)
	sma26, _ := indicators.SMA(closes, 26)
	adx, _ := indicators.ADX(highs, lows, closes, 14)

	df.Indicators["ema_8"] = Pad(ema8, len(df.Candles))
	df.Indicators["sma_12"] = Pad(sma12, len(df.Candles))
	df.Indicators["sma_26"] = Pad(sma26, len(df.Candles))
	df.Indicators["adx"] = Pad(adx, len(df.Candles))

	return nil
}

func (s *AlphaPrimeStrategy) PopulateEntrySignal(df *DataFrame, current Candle) Signal {
	sma12 := GetVal(df, "sma_12")
	sma26 := GetVal(df, "sma_26")
	adx := GetVal(df, "adx")
	prev12 := GetPrev(df, "sma_12")
	prev26 := GetPrev(df, "sma_26")

	// 🧩 Golden Cross + ADX Filter
	isCrossover := sma12 > sma26 && prev12 <= prev26
	isStrongTrend := adx > 15.0

	// 🧩 Re-Entry Logic (Pullback to SMA 26)
	isPullback := current.Low <= sma26 && current.Close > sma26 && sma12 > sma26

	if isStrongTrend && (isCrossover || isPullback) {
		return Signal{
			Action: "buy", Price: current.Close, Reason: "Alpha Prime Entry (v1.0.7)",
		}
	}

	return Signal{Action: "hold"}
}

func (s *AlphaPrimeStrategy) PopulateExitSignal(df *DataFrame, current Candle, pos Position) Signal {
	ema8 := GetVal(df, "ema_8")

	// 🔴 Fast EMA Exit (Momentum loss)
	if current.Close < ema8 && GetPrev(df, "ema_8") >= GetPrev(df, "sma_12") {
		return Signal{Action: "sell", Price: current.Close, Reason: "Momentum Exit (EMA 8)"}
	}

	return Signal{Action: "hold"}
}

func (s *AlphaPrimeStrategy) CustomStoploss(current Candle, position Position, currentProfit float64) *float64 {
	// Apes don't trail, they ride. 
	return nil
}
