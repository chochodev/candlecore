package strategy

import (
	"candlecore/internal/indicators"
	"fmt"
)

// MACrossoverStrategy implements baseline-matching logic with added safety
type MACrossoverStrategy struct {
	BaseStrategy
	FastPeriod   int
	SlowPeriod   int
	StopLimit    float64
}

// NewMACrossover creates version 1.0.2 "Managed Baseline"
func NewMACrossover() IStrategy {
	return &MACrossoverStrategy{
		BaseStrategy: BaseStrategy{
			Name:    "MA Crossover",
			Version: "1.0.2",
			Config: StrategyConfig{
				Stoploss:      -0.03, // 3% Protective Stop
				StakeAmount:   100.0,
				MaxOpenTrades: 5,
				Timeframe:     "1h",
			},
		},
		FastPeriod: 12,
		SlowPeriod: 26,
		StopLimit:  -0.03,
	}
}

func init() {
	Register("ma_crossover", NewMACrossover)
}

func (s *MACrossoverStrategy) PopulateIndicators(df *DataFrame) error {
	if len(df.Candles) < s.SlowPeriod {
		return fmt.Errorf("warmup: need %d candles", s.SlowPeriod)
	}
	if df.Indicators == nil { df.Indicators = make(map[string][]float64) }

	closes := make([]float64, len(df.Candles))
	for i, v := range df.Candles { closes[i] = v.Close }

	// Use SMA (Baseline Identical)
	f, _ := indicators.SMA(closes, s.FastPeriod)
	sP, _ := indicators.SMA(closes, s.SlowPeriod)

	df.Indicators["sma_f"] = pad(f, len(df.Candles))
	df.Indicators["sma_s"] = pad(sP, len(df.Candles))
	return nil
}

func (s *MACrossoverStrategy) PopulateEntrySignal(df *DataFrame, current Candle) Signal {
	f := getVal(df, "sma_f")
	sP := getVal(df, "sma_s")
	pf := getPrev(df, "sma_f")
	ps := getPrev(df, "sma_s")

	// Same entry logic as v1.0.0
	if pf <= ps && f > sP {
		return Signal{
			Action:     "buy",
			Confidence: 75,
			Reason:     "Golden Cross (Baseline Match)",
			Price:      current.Close,
		}
	}
	return Signal{Action: "hold"}
}

func (s *MACrossoverStrategy) PopulateExitSignal(df *DataFrame, current Candle, position Position) Signal {
	// 🛡️ ENFORCED SAFETY
	profitPct := (current.Close - position.EntryPrice) / position.EntryPrice
	if profitPct <= s.StopLimit {
		return Signal{Action: "sell", Reason: "Enforced Stop Loss (3%)", Price: current.Close}
	}

	f := getVal(df, "sma_f")
	sP := getVal(df, "sma_s")
	pf := getPrev(df, "sma_f")
	ps := getPrev(df, "sma_s")

	// Standard technical exit
	if pf >= ps && f < sP {
		return Signal{Action: "sell", Reason: "Technical Exit", Price: current.Close}
	}

	return Signal{Action: "hold"}
}
