package strategy

import (
	"candlecore/internal/indicators"
	"fmt"
)

// SolSniperStrategy (v1.0.8 Alpha)
// This is a "surgical" High-Momentum strategy tuned specifically for Solana's (SOL).
type SolSniperStrategy struct {
	BaseStrategy
}

func NewSolSniper() IStrategy {
	return &SolSniperStrategy{
		BaseStrategy: BaseStrategy{
			Name:    "Solana Sniper",
			Version: "1.0.8",
			Config: StrategyConfig{
				Stoploss:      -0.035, // Tight 3.5% Stoploss
				StakeAmount:   100.0,
				MaxOpenTrades: 1,
				Timeframe:     "1h",
			},
		},
	}
}

func init() {
	Register("sol_sniper", NewSolSniper)
}

func (s *SolSniperStrategy) PopulateIndicators(df *DataFrame) error {
	if len(df.Candles) < 50 {
		return fmt.Errorf("insufficient data for v1.0.8: need 50 candles")
	}
	if df.Indicators == nil {
		df.Indicators = make(map[string][]float64)
	}

	closes := ExtractCloses(df.Candles)
	highs, lows := ExtractHighs(df.Candles), ExtractLows(df.Candles)

	ema5, _ := indicators.EMA(closes, 5)
	ema13, _ := indicators.EMA(closes, 13)
	df.Indicators["ema_5"] = Pad(ema5, len(df.Candles))
	df.Indicators["ema_13"] = Pad(ema13, len(df.Candles))

	adx, _ := indicators.ADX(highs, lows, closes, 14)
	df.Indicators["adx"] = Pad(adx, len(df.Candles))

	return nil
}

func (s *SolSniperStrategy) PopulateEntrySignal(df *DataFrame, current Candle) Signal {
	ema5 := GetVal(df, "ema_5")
	ema13 := GetVal(df, "ema_13")
	adx := GetVal(df, "adx")

	isFastCross := ema5 > ema13 && GetPrev(df, "ema_5") <= GetPrev(df, "ema_13")
	isStrongTrend := adx > 25.0

	if isStrongTrend && isFastCross {
		return Signal{
			Action: "buy", Price: current.Close, Reason: "SOL Pulse Entry (v1.0.8)",
		}
	}

	return Signal{Action: "hold"}
}

func (s *SolSniperStrategy) PopulateExitSignal(df *DataFrame, current Candle, pos Position) Signal {
	ema5 := GetVal(df, "ema_5")
	ema13 := GetVal(df, "ema_13")

	if ema5 < ema13 && GetPrev(df, "ema_5") >= GetPrev(df, "ema_13") {
		return Signal{
			Action: "sell", Price: current.Close, Reason: "SOL Sniper Exhaustion (v1.0.8)",
		}
	}

	return Signal{Action: "hold"}
}
