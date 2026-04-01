package exchange

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

// BinanceWSClient handles live WebSocket connections to Binance
type BinanceWSClient struct {
	conn *websocket.Conn
}

// BinanceKlineStream represents the 1s kline payload
type BinanceKlineStream struct {
	Symbol string `json:"s"`
	Kline  struct {
		Open      string `json:"o"`
		High      string `json:"h"`
		Low       string `json:"l"`
		Close     string `json:"c"`
		Volume    string `json:"v"`
		CloseTime int64  `json:"T"`
		IsClosed  bool   `json:"x"`
	} `json:"k"`
}

func NewBinanceWSClient() *BinanceWSClient {
	return &BinanceWSClient{}
}

// Stream1s connects to the 1-second kline stream for a symbol
func (c *BinanceWSClient) Stream1s(symbol string, candleChan chan<- Candle) error {
	u := url.URL{Scheme: "wss", Host: "stream.binance.com:9443", Path: fmt.Sprintf("/ws/%skline_1s", symbol)}
	log.Printf("Connecting to %s", u.String())

	var err error
	c.conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}

	go func() {
		defer c.conn.Close()
		for {
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				log.Println("WS Read Error:", err)
				return
			}

			var raw BinanceKlineStream
			if err := json.Unmarshal(message, &raw); err != nil {
				continue
			}

			// Convert to our internal Candle format
			candle := Candle{
				Timestamp: time.UnixMilli(raw.Kline.CloseTime),
				Open:      parseFloat(raw.Kline.Open),
				High:      parseFloat(raw.Kline.High),
				Low:       parseFloat(raw.Kline.Low),
				Close:     parseFloat(raw.Kline.Close),
				Volume:    parseFloat(raw.Kline.Volume),
			}
			candleChan <- candle
		}
	}()

	return nil
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
