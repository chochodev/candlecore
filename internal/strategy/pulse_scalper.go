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
			Action: "buy", Price: current.Close, Reason: "Pulse Entry",
		}
	}
	return Signal{Action: "hold"}
}

func (s *PulseScalperStrategy) PopulateExitSignal(df *DataFrame, current Candle, pos Position) Signal {
	lastEMA5 := GetVal(df, "ema_5")
	lastEMA12 := GetVal(df, "ema_12")
	pnlPct := (current.Close - pos.EntryPrice) / pos.EntryPrice
	
	shieldActive := s.shieldActivated[pos.EntryTime]

	// 🛡️ FEE SHIELD: As per User Request
	// Trigger at +0.1% Profit
	if !shieldActive && pnlPct >= 0.001 {
		s.shieldActivated[pos.EntryTime] = true
		return Signal{
		    Action:   "sell",
		    Price:    current.Close,
		    Quantity: pos.Size * 0.5,
		    Reason:   "Fee Shield Triggered (+0.1%). Scaling half to cover fees.",
		}
	}

	if shieldActive {
		// Exit at +0.1% Lock
		if current.Close <= pos.EntryPrice * 1.001 {
			return Signal{
				Action: "sell", Price: current.Close, Reason: "Fee Shield Lock Hit (+0.1%)",
			}
		}
		
		// Major Target
		if pnlPct >= 0.015 {
			return Signal{
				Action: "sell", Price: current.Close, Reason: "Major Target Hit (+1.5%)",
			}
		}
	}

	if lastEMA5 < lastEMA12 {
		return Signal{
			Action: "sell", Price: current.Close, Reason: "Trend Exit",
		}
	}

	return Signal{Action: "hold"}
}
