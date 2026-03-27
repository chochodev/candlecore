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
				Stoploss:      -0.03, // 3% Protective
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
		return fmt.Errorf("insufficient data for 1.0.3: need 50 candles")
	}
	if df.Indicators == nil { df.Indicators = make(map[string][]float64) }

	closes := extractCloses(df.Candles)
	highs, lows := extractHighs(df.Candles), extractLows(df.Candles)

	ema9, _ := indicators.EMA(closes, s.FastPeriod)
	ema21, _ := indicators.EMA(closes, s.SlowPeriod)
	ema50, _ := indicators.EMA(closes, s.TrendPeriod)
	df.Indicators["ema_f"] = pad(ema9, len(df.Candles))
	df.Indicators["ema_s"] = pad(ema21, len(df.Candles))
	df.Indicators["ema_t"] = pad(ema50, len(df.Candles))
	df.Indicators["close"] = closes

	bb, _ := indicators.BollingerBands(closes, 20, 2.0)
	df.Indicators["bb_upper"] = pad(bb.Upper, len(df.Candles))
	df.Indicators["bb_middle"] = pad(bb.Middle, len(df.Candles))
	df.Indicators["bb_lower"] = pad(bb.Lower, len(df.Candles))

	adx, _ := indicators.ADX(highs, lows, closes, 14)
	df.Indicators["adx"] = pad(adx, len(df.Candles))

	return nil
}

func (s *EnsembleStrategy) PopulateEntrySignal(df *DataFrame, current Candle) Signal {
	adx := getVal(df, "adx")
	ema9 := getVal(df, "ema_f")
	ema21 := getVal(df, "ema_s")
	ema50 := getVal(df, "ema_t")
	prevEma50 := getPrev(df, "ema_t")
	upper := getVal(df, "bb_upper")
	lower := getVal(df, "bb_lower")
	middle := getVal(df, "bb_middle")

	// 🛡️ CRASH GUARD 1: Trend Health (Slope check)
	if current.Close < ema50 || ema50 < prevEma50 {
		return Signal{Action: "hold", Reason: "Bearish bias or Negative Slope"}
	}

	// 🛡️ CRASH GUARD 2: Overextension Check (BandWidth ceiling 65%)
	bw := (upper - lower) / middle
	if bw > 0.65 {
		return Signal{Action: "hold", Reason: "Market hyper-extended"}
	}

	bbBreakout := current.Close > upper && getPrev(df, "bb_upper") > 0
	maCross := ema9 > ema21 && getPrev(df, "ema_f") <= getPrev(df, "ema_s")

	if (bbBreakout || maCross) && adx < 55 {
		return Signal{
			Action: "buy", Price: current.Close, Reason: "Hybrid Entry (v1.0.3)",
		}
	}
	return Signal{Action: "hold"}
}

func (s *EnsembleStrategy) PopulateExitSignal(df *DataFrame, current Candle, position Position) Signal {
	middle := getVal(df, "bb_middle")
	ema9 := getVal(df, "ema_f")
	ema21 := getVal(df, "ema_s")
	prevClose := getPrev(df, "close")

	// 🛡️ CRASH GUARD 3: 1.5% Hourly Emergency Exit
	hourlyDrop := (current.Close - prevClose) / prevClose
	if hourlyDrop <= -0.015 {
		return Signal{Action: "sell", Price: current.Close, Reason: "Emergency Hourly Stop"}
	}

	profitPct := (current.Close - position.EntryPrice) / position.EntryPrice
	if profitPct <= s.Config.Stoploss {
		return Signal{Action: "sell", Reason: "Hard Stop", Price: current.Close}
	}

	// Loosened 2-candle confirmation for standard exhaustions
	prevMid := getPrev(df, "bb_middle")
	isBBMidBreak := current.Close < middle && prevClose < prevMid && prevMid > 0
	isMACrossDown := ema9 < ema21 && getPrev(df, "ema_f") < getPrev(df, "ema_s")

	if isBBMidBreak || isMACrossDown {
		return Signal{
			Action: "sell", Price: current.Close, Reason: "Confirmed Exhaustion (v1.0.3)",
		}
	}
	return Signal{Action: "hold"}
}
