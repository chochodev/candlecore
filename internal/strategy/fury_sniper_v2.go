package strategy

import (
	"candlecore/internal/indicators"
	"fmt"
)

// FurySniperV2 (The Scraper Logic)
// Modeled for High Win-Rate (90%+) via Skewed Risk/Reward.
type FurySniperV2 struct {
	BaseStrategy
}

func NewFurySniperV2() IStrategy {
	return &FurySniperV2{
		BaseStrategy: BaseStrategy{
			Name:    "Fury Sniper",
			Version: "2.0",
			Config: StrategyConfig{
				Stoploss:      -0.030, // Wide Stop (3%)
				Takeprofit:    0.010,  // 1% Target
				StakeAmount:   100.0,
				MaxOpenTrades: 5,
				Timeframe:     "5m",
			},
		},
	}
}

func init() {
	Register("fury_sniper_v2", NewFurySniperV2)
}

func (s *FurySniperV2) PopulateIndicators(df *DataFrame) error {
	if len(df.Candles) < 60 { return fmt.Errorf("need more data") }
	if df.Indicators == nil { df.Indicators = make(map[string][]float64) }

	closes := ExtractCloses(df.Candles)
	highs := ExtractHighs(df.Candles)
	lows := ExtractLows(df.Candles)

	// Range + Trend filters
	bb, _ := indicators.BollingerBands(closes, 20, 1.5)
	adx, _ := indicators.ADX(highs, lows, closes, 14)
	ema50, _ := indicators.EMA(closes, 50)

	df.Indicators["bb_upper"] = Pad(bb.Upper, len(df.Candles))
	df.Indicators["bb_lower"] = Pad(bb.Lower, len(df.Candles))
	df.Indicators["adx"] = Pad(adx, len(df.Candles))
	df.Indicators["ema50"] = Pad(ema50, len(df.Candles))

	return nil
}

func (s *FurySniperV2) PopulateEntrySignal(df *DataFrame, current Candle) Signal {
	adx := GetVal(df, "adx")
	lower := GetVal(df, "bb_lower")
	upper := GetVal(df, "bb_upper")
	ema := GetVal(df, "ema50")
	
	// Previous candle for crossover detection
	prev := df.Candles[len(df.Candles)-2]

	// 🛑 FURY FILTER: Volatility Check
	if adx > 20.0 || ema == 0 { return Signal{Action: "hold"} }

	// 🟢 LONG: Cross back ABOVE lower band + Trend Support
	if current.Close > ema && prev.Close < lower && current.Close > lower {
		return Signal{
			Action: "buy", Price: current.Close, Reason: "Fury Scrape: Trend-Aligned Bounce (Long)",
		}
	}

	// 🔴 SHORT: Cross back BELOW upper band + Trend Resistance
	if current.Close < ema && prev.Close > upper && current.Close < upper {
		return Signal{
			Action: "sell", Price: current.Close, Reason: "Fury Scrape: Trend-Aligned Rejection (Short)",
		}
	}

	return Signal{Action: "hold"}
}

func (s *FurySniperV2) PopulateExitSignal(df *DataFrame, current Candle, pos Position) Signal {
	// v2.4: SCRAPE OR DIE
	// We no longer exit on mean-reversion (middle band).
	// To reach 95%+ win rate, we MUST hold until the +1.2% TP target or -5.0% SL.
	return Signal{Action: "hold"}
}

func (s *FurySniperV2) Configure(params map[string]interface{}) {}
