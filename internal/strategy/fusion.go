package strategy

import (
	"candlecore/internal/indicators"
	"fmt"
)

type FusionStrategy struct {
	BaseStrategy
}

func NewFusion() IStrategy {
	return &FusionStrategy{
		BaseStrategy: BaseStrategy{
			Name:    "Fusion Elite",
			Version: "1.0.6",
			Config: StrategyConfig{
				Stoploss:      -0.05,
				StakeAmount:   100.0,
				MaxOpenTrades: 5,
				Timeframe:     "1h",
				CustomParams: map[string]interface{}{
					"use_ema_200": true,
					"use_trix":    true,
					"use_fisher":  true,
				},
			},
		},
	}
}

func init() {
	Register("fusion", NewFusion)
}

func (s *FusionStrategy) PopulateIndicators(df *DataFrame) error {
	if len(df.Candles) < 50 {
		return fmt.Errorf("insufficient data for v1.0.6: need 50 candles")
	}
	if df.Indicators == nil { df.Indicators = make(map[string][]float64) }

	closes := ExtractCloses(df.Candles)
	highs, lows := ExtractHighs(df.Candles), ExtractLows(df.Candles)

	ema9, _ := indicators.EMA(closes, 9)
	ema21, _ := indicators.EMA(closes, 21)
	ema50, _ := indicators.EMA(closes, 50)
	
	df.Indicators["ema_9"] = Pad(ema9, len(df.Candles))
	df.Indicators["ema_21"] = Pad(ema21, len(df.Candles))
	df.Indicators["ema_50"] = Pad(ema50, len(df.Candles))

	// Turbo TRIX (Window 9)
	trix, _ := indicators.TRIX(closes, 9)
	df.Indicators["trix"] = Pad(trix, len(df.Candles))

	fisher, _ := indicators.FisherTransform(highs, lows, 14)
	df.Indicators["fisher"] = Pad(fisher, len(df.Candles))

	return nil
}

func (s *FusionStrategy) PopulateEntrySignal(df *DataFrame, current Candle) Signal {
	ema9 := GetVal(df, "ema_9")
	ema21 := GetVal(df, "ema_21")
	trix := GetVal(df, "trix")
	fisher := GetVal(df, "fisher")
	prevFisher := GetPrev(df, "fisher")

	// 🧩 High-Speed Trend Momentum
	isTrixBullish := trix > 0
	
	// 🚀 Fast Momentum Cross
	isEmaBullish := ema9 > ema21

	// 🎯 Elastic Reversion (Any positive movement)
	isFisherTurning := fisher > prevFisher

	if isTrixBullish && isEmaBullish && isFisherTurning {
		return Signal{
			Action: "buy", Price: current.Close, Reason: "Hyper-Fusion Entry (v1.0.6 Turbo)",
		}
	}

	return Signal{Action: "hold"}
}

func (s *FusionStrategy) PopulateExitSignal(df *DataFrame, current Candle, pos Position) Signal {
	ema9 := GetVal(df, "ema_9")
	ema21 := GetVal(df, "ema_21")
	fisher := GetVal(df, "fisher")
	prevFisher := GetPrev(df, "fisher")

	// 🔴 Momentum Exhaustion
	if ema9 < ema21 && GetPrev(df, "ema_9") >= GetPrev(df, "ema_21") {
		return Signal{Action: "sell", Price: current.Close, Reason: "Momentum Exhaustion (v1.0.6)"}
	}

	// 🔴 Cycle Peak
	if fisher > 1.5 && fisher < prevFisher {
		return Signal{Action: "sell", Price: current.Close, Reason: "Fisher Peak (v1.0.6)"}
	}

	return Signal{Action: "hold"}
}
