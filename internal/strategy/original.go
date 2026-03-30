package strategy

import (
	"candlecore/internal/indicators"
	"fmt"
)

// OriginalStrategy is v1.0.0 (Baseline SMA 12/26)
type OriginalStrategy struct {
	BaseStrategy
}

func NewOriginal() IStrategy {
	return &OriginalStrategy{
		BaseStrategy: BaseStrategy{
			Name:    "Baseline (v1.0.0)",
			Version: "1.0.0",
		},
	}
}

func init() {
	Register("original", NewOriginal)
}

func (s *OriginalStrategy) PopulateIndicators(df *DataFrame) error {
	if len(df.Candles) < 50 {
		return fmt.Errorf("insufficient data for v1.0.0: need 50 candles")
	}
	if df.Indicators == nil { df.Indicators = make(map[string][]float64) }

	closes := ExtractCloses(df.Candles)
	sma12, _ := indicators.SMA(closes, 12)
	sma26, _ := indicators.SMA(closes, 26)

	df.Indicators["sma_f"] = Pad(sma12, len(df.Candles))
	df.Indicators["sma_s"] = Pad(sma26, len(df.Candles))

	return nil
}

func (s *OriginalStrategy) PopulateEntrySignal(df *DataFrame, current Candle) Signal {
	sma12 := GetVal(df, "sma_f")
	sma26 := GetVal(df, "sma_s")
	prev12 := GetPrev(df, "sma_f")
	prev26 := GetPrev(df, "sma_s")

	if sma12 > sma26 && prev12 <= prev26 {
		return Signal{
			Action: "buy", Price: current.Close, Reason: "Golden Cross (v1.0.0)",
		}
	}

	return Signal{Action: "hold"}
}

func (s *OriginalStrategy) PopulateExitSignal(df *DataFrame, current Candle, pos Position) Signal {
	sma12 := GetVal(df, "sma_f")
	sma26 := GetVal(df, "sma_s")
	prev12 := GetPrev(df, "sma_f")
	prev26 := GetPrev(df, "sma_s")

	if sma12 < sma26 && prev12 >= prev26 {
		return Signal{
			Action: "sell", Price: current.Close, Reason: "Death Cross (v1.0.0)",
		}
	}

	return Signal{Action: "hold"}
}
