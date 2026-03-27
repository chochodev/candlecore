package strategy

import (
	"candlecore/internal/indicators"
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
	closes := extractCloses(df.Candles)
	sma12, _ := indicators.SMA(closes, 12)
	sma26, _ := indicators.SMA(closes, 26)
	df.Indicators["sma12"] = pad(sma12, len(df.Candles))
	df.Indicators["sma26"] = pad(sma26, len(df.Candles))
	return nil
}

func (s *OriginalStrategy) PopulateEntrySignal(df *DataFrame, current Candle) Signal {
	s12 := getVal(df, "sma12")
	s26 := getVal(df, "sma26")
	prev12 := getPrev(df, "sma12")
	prev26 := getPrev(df, "sma26")

	if s12 > s26 && prev12 <= prev26 {
		return Signal{Action: "buy", Price: current.Close, Reason: "SMA Golden Cross"}
	}
	return Signal{Action: "hold"}
}

func (s *OriginalStrategy) PopulateExitSignal(df *DataFrame, current Candle, position Position) Signal {
	s12 := getVal(df, "sma12")
	s26 := getVal(df, "sma26")
	prev12 := getPrev(df, "sma12")
	prev26 := getPrev(df, "sma26")

	if s12 < s26 && prev12 >= prev26 {
		return Signal{Action: "sell", Price: current.Close, Reason: "SMA Death Cross"}
	}
	return Signal{Action: "hold"}
}
