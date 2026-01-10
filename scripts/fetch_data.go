package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type BinanceKline []interface{}

func main() {
	intervals := []struct {
		interval string
		filename string
		limit    int
		desc     string
	}{
		{"1m", "bitcoin_1m.csv", 1000, "16.6 hours"},
		{"5m", "bitcoin_5m.csv", 1000, "3.5 days"},
		{"15m", "bitcoin_15m.csv", 1000, "10.4 days"},
		{"1h", "bitcoin_1h.csv", 1000, "41 days"},
		{"4h", "bitcoin_4h.csv", 1000, "166 days"},
		{"1d", "bitcoin_1d.csv", 1000, "2.7 years"},
	}

	os.MkdirAll("data/historical", 0755)

	for _, cfg := range intervals {
		fmt.Printf("Fetching %s data (%s worth)...\n", cfg.interval, cfg.desc)
		
		url := fmt.Sprintf("https://api.binance.com/api/v3/klines?symbol=BTCUSDT&interval=%s&limit=%d",
			cfg.interval, cfg.limit)
		
		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
			continue
		}

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			fmt.Printf("  API Error %d: %s\n", resp.StatusCode, string(body))
			continue
		}

		var klines []BinanceKline
		if err := json.NewDecoder(resp.Body).Decode(&klines); err != nil {
			resp.Body.Close()
			fmt.Printf("  Decode error: %v\n", err)
			continue
		}
		resp.Body.Close()

		file, err := os.Create("data/historical/" + cfg.filename)
		if err != nil {
			fmt.Printf("  File error: %v\n", err)
			continue
		}

		file.WriteString("timestamp,open,high,low,close,volume\n")
		for _, k := range klines {
			timestamp := time.Unix(int64(k[0].(float64))/1000, 0).UTC().Format("2006-01-02T15:04:05Z")
			open := k[1].(string)
			high := k[2].(string)
			low := k[3].(string)
			close := k[4].(string)
			volume := k[5].(string)
			
			fmt.Fprintf(file, "%s,%s,%s,%s,%s,%s\n", timestamp, open, high, low, close, volume)
		}
		file.Close()

		firstTime := time.Unix(int64(klines[0][0].(float64))/1000, 0)
		lastTime := time.Unix(int64(klines[len(klines)-1][0].(float64))/1000, 0)
		
		fmt.Printf("  Downloaded %d candles to %s\n", len(klines), cfg.filename)
		fmt.Printf("  Range: %s to %s\n", firstTime.Format("2006-01-02 15:04"), lastTime.Format("2006-01-02 15:04"))
		
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("\nDone! All timeframes ready for real-time simulation:")
	fmt.Println("  bitcoin_1m.csv - 1000 1-minute candles")
	fmt.Println("  bitcoin_5m.csv - 1000 5-minute candles")
	fmt.Println("  bitcoin_15m.csv - 1000 15-minute candles")
	fmt.Println("  bitcoin_1h.csv - 1000 hourly candles")
	fmt.Println("  bitcoin_4h.csv - 1000 4-hour candles")
	fmt.Println("  bitcoin_1d.csv - 1000 daily candles")
}
