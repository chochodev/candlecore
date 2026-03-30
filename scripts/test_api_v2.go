package main

import (
	"fmt"
	"net/http"
	"time"
	"io/ioutil"
)

func main() {
	endpoints := []string{
		"https://api.binance.com/api/v3/klines?symbol=BTCUSDT&interval=1h&limit=1",
		"https://api.binance.us/api/v3/klines?symbol=BTCUSDT&interval=1h&limit=1",
		"https://data-api.binance.vision/api/v3/klines?symbol=BTCUSDT&interval=1h&limit=1",
		"https://api1.binance.com/api/v3/klines?symbol=BTCUSDT&interval=1h&limit=1",
		"https://api2.binance.com/api/v3/klines?symbol=BTCUSDT&interval=1h&limit=1",
		"https://api3.binance.com/api/v3/klines?symbol=BTCUSDT&interval=1h&limit=1",
	}

	client := &http.Client{Timeout: 5 * time.Second}

	for _, url := range endpoints {
		fmt.Printf("🔍 Testing: %s... ", url)
		resp, err := client.Get(url)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			continue
		}
		
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == 200 {
			fmt.Printf("✅ Success! (Code 200)\n")
			fmt.Printf("   Sample Data: %s\n", string(body)[:50]+"...")
			return
		} else {
			fmt.Printf("⚠️  Failed: Status %d\n", resp.StatusCode)
		}
	}
	fmt.Println("\n🛑 All API endpoints failed. Connectivity issue detected.")
}
