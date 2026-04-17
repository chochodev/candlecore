package research

import (
	"time"
)

// FeatureSet represents the statistical state of a candle
type FeatureSet struct {
	Timestamp      time.Time `json:"timestamp"`
	Symbol         string    `json:"symbol"`
	
	// HTF Context (H1)
	H1Trend        string    `json:"h1_trend"`         // "bullish" (50 > 200), "bearish" (50 < 200)
	H1EMADistance  float64   `json:"h1_ema_distance"`  // Normalized distance from 50 EMA
	
	// Local Price Behavior (M15)
	PriceDistEMA50 float64   `json:"price_dist_ema50"` // Normalized distance to M15 EMA 50
	PullbackDepth  string    `json:"pullback_depth"`   // "deep", "moderate", "overextended"
	VolSpike       string    `json:"vol_spike"`        // "high_vol", "normal", "low_vol"
	CandleType     string    `json:"candle_type"`      // "engulfing_bull", "engulfing_bear", "doji", "normal"
	ATR            float64   `json:"atr"`              // Volatility
	RangeExpansion float64   `json:"range_expansion"`  // Current range / average range of last 10
	
	// Market Structure
	BreakOfStructure string  `json:"bos"`              // "high_break", "low_break", "none"
	Session          string  `json:"session"`          // "Asia", "London", "NY", "Unknown"
}

// OutcomeLabel represents the Triple Barrier Method result
type OutcomeLabel struct {
	MFE         float64   `json:"mfe"`          // Max Favorable Excursion (Profit seen)
	MAE         float64   `json:"mae"`          // Max Adverse Excursion (Loss seen)
	HitBarrier  string    `json:"hit_barrier"`  // "tp", "sl", "time"
	RR          float64   `json:"rr"`           // Max R-ratio achieved
	TimeElapsed int       `json:"time_elapsed"` // Minutes until exit
}

// ResearchRow combines features and outcomes for a single event
type ResearchRow struct {
	Features FeatureSet   `json:"features"`
	Labels   OutcomeLabel `json:"labels"`
}
