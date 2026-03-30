package strategy

import (
	"candlecore/internal/indicators"
	"fmt"
)

type EnsembleStrategy struct {
	BaseStrategy
	FastPeriod  int
	SlowPeriod  int
	TrendPeriod int
}

func NewEnsemble() IStrategy {
	return &EnsembleStrategy{
		BaseStrategy: BaseStrategy{
			Name:    "Hybrid Alpha",
			Version: "1.0.3",
			Config: StrategyConfig{
				Stoploss:      -0.05, // 5% Protective
				StakeAmount:   100.0,
				MaxOpenTrades: 5,
				Timeframe:     "1h",
				CustomParams: map[string]interface{}{
					"ema_f": 9,
					"ema_s": 21,
					"ema_t": 50,
				},
			},
		},
		FastPeriod:  9,
		SlowPeriod:  21,
		TrendPeriod: 50,
	}
}

func init() {
	Register("ensemble", NewEnsemble)
}

func (s *EnsembleStrategy) PopulateIndicators(df *DataFrame) error {
	if len(df.Candles) < 50 {
		return fmt.Errorf("insufficient data for v1.0.3: need 50 candles")
	}
	if df.Indicators == nil { df.Indicators = make(map[string][]float64) }

	closes := ExtractCloses(df.Candles)

	ema9, _ := indicators.EMA(closes, 9)
	ema21, _ := indicators.EMA(closes, 21)
	ema50, _ := indicators.EMA(closes, 50)
	
	df.Indicators["ema_f"] = Pad(ema9, len(df.Candles))
	df.Indicators["ema_s"] = Pad(ema21, len(df.Candles))
	df.Indicators["ema_50"] = Pad(ema50, len(df.Candles))

	resBB, _ := indicators.BollingerBands(closes, 20, 2.0)
	df.Indicators["bb_upper"] = Pad(resBB.Upper, len(df.Candles))
	df.Indicators["bb_middle"] = Pad(resBB.Middle, len(df.Candles))
	df.Indicators["bb_lower"] = Pad(resBB.Lower, len(df.Candles))

	// BandWidth calculation
	width := make([]float64, len(resBB.Middle))
	for i := 0; i < len(resBB.Middle); i++ {
		width[i] = (resBB.Upper[i] - resBB.Lower[i]) / resBB.Middle[i]
	}
	df.Indicators["bb_width"] = Pad(width, len(df.Candles))

	return nil
}

func (s *EnsembleStrategy) PopulateEntrySignal(df *DataFrame, current Candle) Signal {
	width := GetVal(df, "bb_width")
	prevWidth := GetPrev(df, "bb_width")
	ema9 := GetVal(df, "ema_f")
	ema21 := GetVal(df, "ema_s")
	ema50 := GetVal(df, "ema_50")

	// Filter 1: Trend Health (EMA 50 Slope)
	isTrendRising := ema50 > GetPrev(df, "ema_50")

	// Filter 2: Volatility Ceiling (Avoid 'Blow-off' tops)
	isVolatilitySafe := width < 0.65 

	// Filter 3: Squeeze Breakout (Bollinger Squeeze)
	isSqueezeBreaking := width > prevWidth && GetPrev(df, "bb_width") < GetPrevVal(df, "bb_width", 5)

	if isTrendRising && isVolatilitySafe && isSqueezeBreaking && ema9 > ema21 {
		return Signal{
			Action: "buy", Price: current.Close, Reason: "Squeeze Breakout (v1.0.3 Optimized)",
		}
	}
	return Signal{Action: "hold"}
}

func (s *EnsembleStrategy) PopulateExitSignal(df *DataFrame, current Candle, position Position) Signal {
	middle := GetVal(df, "bb_middle")
	ema9 := GetVal(df, "ema_f")
	ema21 := GetVal(df, "ema_s")

	// Hard Stop-loss check (already handled by engine but good for signal reason)
	profitPct := (current.Close - position.EntryPrice) / position.EntryPrice
	if profitPct <= s.Config.Stoploss {
		return Signal{Action: "sell", Price: current.Close, Reason: "Hard Stoploss"}
	}

	// Momentum Exit: EMA cross down
	if ema9 < ema21 && GetPrev(df, "ema_f") >= GetPrev(df, "ema_s") {
		return Signal{Action: "sell", Price: current.Close, Reason: "Momentum Loss"}
	}
	
	// Trend Exit: Price below BB Middle for 2 consecutive candles
	if current.Close < middle && GetPrev(df, "close") < GetPrev(df, "bb_middle") {
		return Signal{Action: "sell", Price: current.Close, Reason: "Trend Breakout"}
	}
	
	return Signal{Action: "hold"}
}

func (s *EnsembleStrategy) CustomStoploss(current Candle, position Position, currentProfit float64) *float64 {
	return nil // Use fixed SL
}
