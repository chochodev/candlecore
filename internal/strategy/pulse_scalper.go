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
		return fmt.Errorf("insufficient data")
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
	lastEMA5 := GetVal(df, "ema_5")
	lastEMA12 := GetVal(df, "ema_12")
	lastRSI := GetVal(df, "rsi_14")

	if lastEMA5 > lastEMA12 && current.Close > lastEMA5 && lastRSI > 45 && lastRSI < 65 {
		return Signal{
			Action: "buy", Price: current.Close, Reason: "Pulse Entry (Long)",
		}
	}

	// Short Pulse: Mirror of Long
	if lastEMA5 < lastEMA12 && current.Close < lastEMA5 && lastRSI < 55 && lastRSI > 35 {
		return Signal{
			Action: "sell", Price: current.Close, Reason: "Pulse Entry (Short)",
		}
	}

	return Signal{Action: "hold"}
}

func (s *PulseScalperStrategy) PopulateExitSignal(df *DataFrame, current Candle, pos Position) Signal {
	lastEMA5 := GetVal(df, "ema_5")
	lastEMA12 := GetVal(df, "ema_12")

	// P&L calculation based on position side
	var pnlPct float64
	if pos.Side == "long" {
		pnlPct = (current.Close - pos.EntryPrice) / pos.EntryPrice
	} else {
		pnlPct = (pos.EntryPrice - current.Close) / pos.EntryPrice
	}

	shieldActive := s.shieldActivated[pos.EntryTime]

	// FEE SHIELD (v1.3.1): Multi-Vector Logic
	if !shieldActive && pnlPct >= 0.001 {
		s.shieldActivated[pos.EntryTime] = true
		
		var lockPrice float64
		if pos.Side == "long" {
			lockPrice = pos.EntryPrice * 1.001
		} else {
			lockPrice = pos.EntryPrice * 0.999
		}

		return Signal{
			Action:     "hold",
			Price:      current.Close,
			Reason:     "Fee Shield Triggered (+0.1%). Locking profit vector.",
			TrailingSL: &lockPrice,
		}
	}

	// Engine now handles Autonomous TP/SL Exits in real-time (Intra-candle).
	// Strategy only handles Trend Reversal (EMA Crosses) at candle close.

	// ── Trend Reversal Exits ───────────────────────────────────────────────────
	if pos.Side == "long" && lastEMA5 < lastEMA12 {
		return Signal{Action: "sell", Price: current.Close, Reason: "Trend Flip Exit (Long)"}
	}
	if pos.Side == "short" && lastEMA5 > lastEMA12 {
		return Signal{Action: "buy", Price: current.Close, Reason: "Trend Flip Exit (Short)"}
	}

	return Signal{Action: "hold"}
}

func (s *PulseScalperStrategy) Configure(params map[string]interface{}) {
	// Config logic
}
