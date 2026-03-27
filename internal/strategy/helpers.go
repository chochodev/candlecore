package strategy

import (
	"candlecore/internal/exchange"
)

// Common helpers for all strategy implementations

func extractCloses(c []Candle) []float64 { 
	res := make([]float64, len(c))
	for i, v := range c { res[i] = v.Close }
	return res 
}

func extractHighs(c []Candle) []float64 { 
	res := make([]float64, len(c))
	for i, v := range c { res[i] = v.High }
	return res 
}

func extractLows(c []Candle) []float64 { 
	res := make([]float64, len(c))
	for i, v := range c { res[i] = v.Low }
	return res 
}

func getVal(df *DataFrame, k string) float64 { 
	if v, ok := df.Indicators[k]; ok && len(v) > 0 { return v[len(v)-1] }
	return 0 
}

func getPrev(df *DataFrame, k string) float64 { 
	if v, ok := df.Indicators[k]; ok && len(v) > 1 { return v[len(v)-2] }
	return 0 
}

func pad(data []float64, targetLen int) []float64 {
	if len(data) >= targetLen { return data }
	padded := make([]float64, targetLen)
	offset := targetLen - len(data)
	for i := 0; i < len(data); i++ { padded[i+offset] = data[i] }
	return padded
}

func getPrevVal(df *DataFrame, k string, offset int) float64 {
	if v, ok := df.Indicators[k]; ok && len(v) > offset { return v[len(v)-1-offset] }
	return 0
}

func exchangeToStratCandles(candles []exchange.Candle) []Candle {
	res := make([]Candle, len(candles))
	for i, c := range candles {
		res[i] = Candle{
			Timestamp: c.Timestamp,
			Open:      c.Open,
			High:      c.High,
			Low:       c.Low,
			Close:     c.Close,
			Volume:    c.Volume,
		}
	}
	return res
}
