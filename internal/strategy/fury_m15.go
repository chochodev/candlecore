package strategy

import (
	"candlecore/internal/indicators"
	"fmt"
)

// FuryM15Strategy (v1.0)
// True range-scalping logic for the 15m timeframe.
type FuryM15Strategy struct {
	BaseStrategy
}

func NewFuryM15() IStrategy {
	return &FuryM15Strategy{
		BaseStrategy: BaseStrategy{
			Name:    "Fury M15",
			Version: "1.0",
			Config: StrategyConfig{
				Stoploss:      -0.025, // 2.5% SL
				StakeAmount:   100.0,
				MaxOpenTrades: 1,
				Timeframe:     "15m",
			},
		},
	}
}

func init() {
	Register("fury_m15", NewFuryM15)
}

func (s *FuryM15Strategy) PopulateIndicators(df *DataFrame) error {
	if len(df.Candles) < 60 { return fmt.Errorf("need more data") }
	if df.Indicators == nil { df.Indicators = make(map[string][]float64) }

	closes := ExtractCloses(df.Candles)
	highs := ExtractHighs(df.Candles)
	lows := ExtractLows(df.Candles)

	// Range detection
	bb, _ := indicators.BollingerBands(closes, 20, 2.0)
	adx, _ := indicators.ADX(highs, lows, closes, 14)

	df.Indicators["bb_upper"] = Pad(bb.Upper, len(df.Candles))
	df.Indicators["bb_lower"] = Pad(bb.Lower, len(df.Candles))
	df.Indicators["bb_mid"] = Pad(bb.Middle, len(df.Candles))
	df.Indicators["adx"] = Pad(adx, len(df.Candles))

	return nil
}

func (s *FuryM15Strategy) PopulateEntrySignal(df *DataFrame, current Candle) Signal {
	adx := GetVal(df, "adx")
	upper := GetVal(df, "bb_upper")
	lower := GetVal(df, "bb_lower")

	// Only trade in calm markets
	if adx > 18.0 { return Signal{Action: "hold"} }

	// Buy the low of the range
	if current.Close <= lower {
		return Signal{
			Action: "buy", Price: current.Close, Reason: "Fury Range: Lower Bound Snipe",
		}
	}

	// Sell the high of the range
	if current.Close >= upper {
		return Signal{
			Action: "sell", Price: current.Close, Reason: "Fury Range: Upper Bound Snipe",
		}
	}

	return Signal{Action: "hold"}
}

func (s *FuryM15Strategy) PopulateExitSignal(df *DataFrame, current Candle, pos Position) Signal {
	mid := GetVal(df, "bb_mid")
	
	// Quick scalp at the mean (0.8% - 1.2% typical move on M15)
	if pos.Side == "long" && current.Close >= mid {
		return Signal{Action: "sell", Price: current.Close, Reason: "Fury: Mean Reversion Hit"}
	}

	if pos.Side == "short" && current.Close <= mid {
		return Signal{Action: "buy", Price: current.Close, Reason: "Fury: Mean Reversion Hit"}
	}

	return Signal{Action: "hold"}
}

func (s *FuryM15Strategy) Configure(params map[string]interface{}) {}
