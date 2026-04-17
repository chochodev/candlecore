package strategy

import (
	"candlecore/internal/indicators"
	"fmt"
)

// VanguardM15Strategy (v1.0)
// Flagship Trend-Following strategy for M15 timeframe.
// Targets high-velocity breakout moves with significant profit windows.
type VanguardM15Strategy struct {
	BaseStrategy
}

func NewVanguardM15() IStrategy {
	return &VanguardM15Strategy{
		BaseStrategy: BaseStrategy{
			Name:    "Vanguard M15",
			Version: "1.1",
			Config: StrategyConfig{
				Stoploss:      -0.050, // Swing SL: 5.0%
				Takeprofit:    0.150,  // Swing TP: 15.0%
				StakeAmount:   100.0,
				MaxOpenTrades: 3,      // Allow more open swings
			},
		},
	}
}

func init() {
	Register("vanguard_m15", NewVanguardM15)
}

func (s *VanguardM15Strategy) PopulateIndicators(df *DataFrame) error {
	if len(df.Candles) < 200 { return fmt.Errorf("need 200 candles") }
	if df.Indicators == nil { df.Indicators = make(map[string][]float64) }

	closes := ExtractCloses(df.Candles)
	highs := ExtractHighs(df.Candles)
	lows := ExtractLows(df.Candles)

	ema12, _ := indicators.EMA(closes, 12)
	ema26, _ := indicators.EMA(closes, 26)
	ema50, _ := indicators.EMA(closes, 50)
	ema200, _ := indicators.EMA(closes, 200)
	adx, _ := indicators.ADX(highs, lows, closes, 14)

	df.Indicators["ema12"] = Pad(ema12, len(df.Candles))
	df.Indicators["ema26"] = Pad(ema26, len(df.Candles))
	df.Indicators["ema50"] = Pad(ema50, len(df.Candles))
	df.Indicators["ema200"] = Pad(ema200, len(df.Candles))
	df.Indicators["adx"] = Pad(adx, len(df.Candles))

	return nil
}

func (s *VanguardM15Strategy) PopulateEntrySignal(df *DataFrame, current Candle) Signal {
	ema12 := GetVal(df, "ema12")
	ema26 := GetVal(df, "ema26")
	ema50 := GetVal(df, "ema50")
	ema200 := GetVal(df, "ema200")
	adx := GetVal(df, "adx")

	if ema200 == 0 || adx == 0 { return Signal{Action: "hold"} }

	// THE OVERLORD GAP: Only enter if trend is clearly "Exploding" (1.5% separation)
	bullGap := (ema12 - ema26) / ema26 >= 0.015
	bearGap := (ema26 - ema12) / ema26 >= 0.015

	// DISCIPLINED ENTRY: Don't buy the peak if it's already overextended
	distFromAnchor := (current.Close - ema26) / ema26
	if distFromAnchor < 0 { distFromAnchor = -distFromAnchor }
	isOverextended := distFromAnchor > 0.030 // Adjusted to 3.0% for 1H swings

	// LONG: Confluence of 12/26/50/200 + Macro Trend (50 > 200) + Proximity
	if current.Close > ema200 && ema50 > ema200 && ema12 > ema26 && bullGap && adx > 25 && !isOverextended {
		return Signal{
			Action: "buy", Price: current.Close, Reason: "Vanguard Overlord: Disciplined Bull Breakout",
		}
	}

	// SHORT: Macro Trend (50 < 200) + Proximity
	if current.Close < ema200 && ema50 < ema200 && ema12 < ema26 && bearGap && adx > 25 && !isOverextended {
		return Signal{
			Action: "sell", Price: current.Close, Reason: "Vanguard Overlord: Disciplined Bear Breakdown",
		}
	}

	return Signal{Action: "hold"}
}

func (s *VanguardM15Strategy) PopulateExitSignal(df *DataFrame, current Candle, pos Position) Signal {
	// Let the hard targets (TP/SL) do the work to maximize profit runs.
	return Signal{Action: "hold"}
}

func (s *VanguardM15Strategy) Configure(params map[string]interface{}) {}
