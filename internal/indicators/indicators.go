package indicators

import (
	"fmt"
	"math"
)

// SMA calculates Simple Moving Average
func SMA(values []float64, period int) ([]float64, error) {
	if period <= 0 {
		return nil, fmt.Errorf("period must be positive")
	}
	if len(values) < period {
		return nil, fmt.Errorf("insufficient data: need %d, got %d", period, len(values))
	}

	result := make([]float64, len(values)-period+1)
	
	// Calculate first SMA
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += values[i]
	}
	result[0] = sum / float64(period)

	// Slide window for remaining values
	for i := period; i < len(values); i++ {
		sum = sum - values[i-period] + values[i]
		result[i-period+1] = sum / float64(period)
	}

	return result, nil
}

// EMA calculates Exponential Moving Average
func EMA(values []float64, period int) ([]float64, error) {
	if period <= 0 {
		return nil, fmt.Errorf("period must be positive")
	}
	if len(values) < period {
		return nil, fmt.Errorf("insufficient data: need %d, got %d", period, len(values))
	}

	result := make([]float64, len(values))
	multiplier := 2.0 / float64(period+1)

	// First EMA is SMA
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += values[i]
	}
	result[period-1] = sum / float64(period)

	// Calculate EMA for remaining values
	for i := period; i < len(values); i++ {
		result[i] = (values[i]-result[i-1])*multiplier + result[i-1]
	}

	return result[period-1:], nil
}

// RSI calculates Relative Strength Index
func RSI(values []float64, period int) ([]float64, error) {
	if period <= 0 {
		return nil, fmt.Errorf("period must be positive")
	}
	if len(values) < period+1 {
		return nil, fmt.Errorf("insufficient data: need %d, got %d", period+1, len(values))
	}

	// Calculate price changes
	changes := make([]float64, len(values)-1)
	for i := 1; i < len(values); i++ {
		changes[i-1] = values[i] - values[i-1]
	}

	// Separate gains and losses
	gains := make([]float64, len(changes))
	losses := make([]float64, len(changes))
	for i, change := range changes {
		if change > 0 {
			gains[i] = change
		} else {
			losses[i] = -change
		}
	}

	result := make([]float64, len(changes)-period+1)

	// First average
	avgGain := 0.0
	avgLoss := 0.0
	for i := 0; i < period; i++ {
		avgGain += gains[i]
		avgLoss += losses[i]
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	// Calculate RSI
	for i := period; i <= len(changes); i++ {
		if avgLoss == 0 {
			result[i-period] = 100
		} else {
			rs := avgGain / avgLoss
			result[i-period] = 100 - (100 / (1 + rs))
		}

		// Update averages for next iteration
		if i < len(changes) {
			avgGain = (avgGain*float64(period-1) + gains[i]) / float64(period)
			avgLoss = (avgLoss*float64(period-1) + losses[i]) / float64(period)
		}
	}

	return result, nil
}

// MACD calculates Moving Average Convergence Divergence
type MACDResult struct {
	MACD      []float64
	Signal    []float64
	Histogram []float64
}

func MACD(values []float64, fastPeriod, slowPeriod, signalPeriod int) (*MACDResult, error) {
	if fastPeriod >= slowPeriod {
		return nil, fmt.Errorf("fast period must be less than slow period")
	}
	if len(values) < slowPeriod {
		return nil, fmt.Errorf("insufficient data")
	}

	// Calculate fast and slow EMAs
	fastEMA, err := EMA(values, fastPeriod)
	if err != nil {
		return nil, err
	}

	slowEMA, err := EMA(values, slowPeriod)
	if err != nil {
		return nil, err
	}

	// Align lengths
	offset := slowPeriod - fastPeriod
	fastEMA = fastEMA[offset:]

	// Calculate MACD line
	macdLine := make([]float64, len(slowEMA))
	for i := range slowEMA {
		macdLine[i] = fastEMA[i] - slowEMA[i]
	}

	// Calculate signal line (EMA of MACD)
	signalLine, err := EMA(macdLine, signalPeriod)
	if err != nil {
		return nil, err
	}

	// Calculate histogram
	macdTrimmed := macdLine[len(macdLine)-len(signalLine):]
	histogram := make([]float64, len(signalLine))
	for i := range signalLine {
		histogram[i] = macdTrimmed[i] - signalLine[i]
	}

	return &MACDResult{
		MACD:      macdTrimmed,
		Signal:    signalLine,
		Histogram: histogram,
	}, nil
}

// BollingerBands calculates Bollinger Bands
type BollingerBandsResult struct {
	Upper  []float64
	Middle []float64
	Lower  []float64
}

func BollingerBands(values []float64, period int, stdDev float64) (*BollingerBandsResult, error) {
	if period <= 0 {
		return nil, fmt.Errorf("period must be positive")
	}
	if len(values) < period {
		return nil, fmt.Errorf("insufficient data")
	}

	// Calculate SMA (middle band)
	middle, err := SMA(values, period)
	if err != nil {
		return nil, err
	}

	upper := make([]float64, len(middle))
	lower := make([]float64, len(middle))

	// Calculate standard deviation and bands
	for i := range middle {
		// Calculate std dev for this window
		start := i
		end := i + period
		window := values[start:end]

		variance := 0.0
		for _, v := range window {
			diff := v - middle[i]
			variance += diff * diff
		}
		variance /= float64(period)
		stdDeviation := math.Sqrt(variance)

		upper[i] = middle[i] + stdDev*stdDeviation
		lower[i] = middle[i] - stdDev*stdDeviation
	}

	return &BollingerBandsResult{
		Upper:  upper,
		Middle: middle,
		Lower:  lower,
	}, nil
}

// ATR calculates Average True Range
func ATR(highs, lows, closes []float64, period int) ([]float64, error) {
	if len(highs) != len(lows) || len(lows) != len(closes) {
		return nil, fmt.Errorf("input slices must have equal length")
	}
	if len(closes) < period+1 {
		return nil, fmt.Errorf("insufficient data for ATR")
	}

	tr := make([]float64, len(closes))
	// TR = max(H-L, |H-Cp|, |L-Cp|)
	for i := 1; i < len(closes); i++ {
		h_l := highs[i] - lows[i]
		h_cp := math.Abs(highs[i] - closes[i-1])
		l_cp := math.Abs(lows[i] - closes[i-1])
		tr[i] = math.Max(h_l, math.Max(h_cp, l_cp))
	}

	// First ATR is SMA of TR
	atr, err := SMA(tr[1:], period)
	if err != nil {
		return nil, err
	}

	return atr, nil
}

// ADX calculates Average Directional Index
func ADX(highs, lows, closes []float64, period int) ([]float64, error) {
	if len(closes) < period*2 {
		return nil, fmt.Errorf("insufficient data for ADX")
	}

	tr := make([]float64, len(closes))
	plusDM := make([]float64, len(closes))
	minusDM := make([]float64, len(closes))

	for i := 1; i < len(closes); i++ {
		h_l := highs[i] - lows[i]
		h_cp := math.Abs(highs[i] - closes[i-1])
		l_cp := math.Abs(lows[i] - closes[i-1])
		tr[i] = math.Max(h_l, math.Max(h_cp, l_cp))

		upMove := highs[i] - highs[i-1]
		downMove := lows[i-1] - lows[i]

		if upMove > downMove && upMove > 0 {
			plusDM[i] = upMove
		}
		if downMove > upMove && downMove > 0 {
			minusDM[i] = downMove
		}
	}

	smoothTR, _ := EMA(tr[1:], period)
	smoothPlusDM, _ := EMA(plusDM[1:], period)
	smoothMinusDM, _ := EMA(minusDM[1:], period)

	dx := make([]float64, len(smoothTR))
	for i := range smoothTR {
		plusDI := 100 * smoothPlusDM[i] / smoothTR[i]
		minusDI := 100 * smoothMinusDM[i] / smoothTR[i]
		dx[i] = 100 * math.Abs(plusDI-minusDI) / (plusDI + minusDI)
	}

	adx, err := SMA(dx, period)
	if err != nil {
		return nil, err
	}

	return adx, nil
}

// FisherTransform calculates Fisher Transform
func FisherTransform(highs, lows []float64, period int) ([]float64, error) {
	if len(highs) < period {
		return nil, fmt.Errorf("insufficient data for Fisher Transform")
	}

	result := make([]float64, len(highs))
	value := 0.0
	fisher := 0.0

	for i := period - 1; i < len(highs); i++ {
		hh := highs[i]
		ll := lows[i]
		for j := i - period + 1; j < i; j++ {
			if highs[j] > hh { hh = highs[j] }
			if lows[j] < ll { ll = lows[j] }
		}

		price := (highs[i] + lows[i]) / 2.0
		val := 0.0
		if hh != ll {
			val = 0.33 * 2 * ((price - ll) / (hh - ll) - 0.5) + 0.67 * value
		}
		value = val
		
		if value > 0.999 { value = 0.999 }
		if value < -0.999 { value = -0.999 }

		fisher = 0.5 * math.Log((1+value)/(1-value)) + 0.5 * fisher
		result[i] = fisher
	}

	return result[period-1:], nil
}

// TRIX calculates Triple Exponentially Smoothed Moving Average
func TRIX(values []float64, period int) ([]float64, error) {
	if len(values) < period*3 {
		return nil, fmt.Errorf("insufficient data for TRIX")
	}

	ema1, _ := EMA(values, period)
	ema2, _ := EMA(ema1, period)
	ema3, _ := EMA(ema2, period)

	result := make([]float64, len(ema3))
	for i := 1; i < len(ema3); i++ {
		if ema3[i-1] != 0 {
			result[i] = (ema3[i] - ema3[i-1]) / ema3[i-1]
		}
	}

	return result[1:], nil
}


