package strategy

import (
	"candlecore/internal/indicators"
	"fmt"
	"time"
)

// PulseScalperStrategy (v1.3.0 - Fee Shield)
type PulseScalperStrategy struct {
	BaseStrategy
	shieldActivated map[time.Time]bool
}

func NewFeeShieldPulse() IStrategy {
	return &PulseScalperStrategy{
		BaseStrategy: BaseStrategy{
			Name:    "Fee Shield Pulse",
			Version: "1.3.0",
			Config: StrategyConfig{
				Stoploss:      -0.015,
				StakeAmount:   100.0,
				MaxOpenTrades: 10,
				Timeframe:     "5m",
			},
		},
		shieldActivated: make(map[time.Time]bool),
	}
}

func init() {
	Register("pulse_scalper", NewFeeShieldPulse)
}

func (s *PulseScalperStrategy) PopulateIndicators(df *DataFrame) error {
	if len(df.Candles) < 50 {
		return fmt.Errorf("insufficient data: %d", len(df.Candles))
	}
	if df.Indicators == nil { df.Indicators = make(map[string][]float64) }
	closes := ExtractCloses(df.Candles)

	ema5, _ := indicators.EMA(closes, 5)
	ema12, _ := indicators.EMA(closes, 12)
	rsi14, _ := indicators.RSI(closes, 14)

	df.Indicators["ema_5"] = Pad(ema5, len(df.Candles))
	df.Indicators["ema_12"] = Pad(ema12, len(df.Candles))
	df.Indicators["rsi_14"] = Pad(rsi14, len(df.Candles))
	return nil
}

func (s *PulseScalperStrategy) PopulateEntrySignal(df *DataFrame, current Candle) Signal {
	ema5 := GetVal(df, "ema_5")
	ema12 := GetVal(df, "ema_12")
	rsi := GetVal(df, "rsi_14")

	if ema5 == 0 { return Signal{Action: "hold"} }

	// FAST CROSS ENTRY: Catch the ripple as it turns into a wave
	if ema5 > ema12 && rsi < 65 {
		return Signal{
			Action: "buy", Price: current.Close, Reason: "Fast Pulse: Bullish Cross",
		}
	}

	if ema5 < ema12 && rsi > 35 {
		return Signal{
			Action: "sell", Price: current.Close, Reason: "Fast Pulse: Bearish Cross",
		}
	}

	return Signal{Action: "hold"}
}

func (s *PulseScalperStrategy) PopulateExitSignal(df *DataFrame, current Candle, pos Position) Signal {
	// v3.0: PURE ASYMMETRIC COMMITTMENT
	// We let the bot engine's TP (+1.4%) and SL (-0.7%) do the work.
	// No intermediate exit logic to prevent fee-whiplash.
	return Signal{Action: "hold"}
}

func (s *PulseScalperStrategy) Configure(params map[string]interface{}) {
	// Config logic
}
