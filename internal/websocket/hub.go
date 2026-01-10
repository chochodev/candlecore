package websocket

import (
	"candlecore/internal/bot"
	"candlecore/internal/exchange"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// EventType represents the type of event
type EventType string

const (
	EventTypeCandle   EventType = "candle"
	EventTypeDecision EventType = "decision"
	EventTypePosition EventType = "position"
	EventTypePnL      EventType = "pnl"
	EventTypeStatus   EventType = "status"
)

// Event represents a WebSocket event
type Event struct {
	Type      EventType   `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// CandleData represents candle  event data
type CandleData struct {
	Symbol    string    `json:"symbol"`
	Timeframe string    `json:"timeframe"`
	Timestamp time.Time `json:"timestamp"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
}

// PnLData represents PnL update
type PnLData struct {
	Balance       float64 `json:"balance"`
	TotalPnL      float64 `json:"total_pnl"`
	UnrealizedPnL float64 `json:"unrealized_pnl,omitempty"`
}

// Hub manages WebSocket connections and broadcasts
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan Event
	Register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan Event, 256),
		Register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("Client connected. Total clients: %d", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("Client disconnected. Total clients: %d", len(h.clients))

		case event := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- event:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastCandle broadcasts a candle update
func (h *Hub) BroadcastCandle(candle exchange.Candle, symbol, timeframe string) {
	h.broadcast <- Event{
		Type:      EventTypeCandle,
		Timestamp: time.Now(),
		Data: CandleData{
			Symbol:    symbol,
			Timeframe: timeframe,
			Timestamp: candle.Timestamp,
			Open:      candle.Open,
			High:      candle.High,
			Low:       candle.Low,
			Close:     candle.Close,
			Volume:    candle.Volume,
		},
	}
}

// BroadcastDecision broadcasts a bot decision
func (h *Hub) BroadcastDecision(decision *bot.Decision) {
	h.broadcast <- Event{
		Type:      EventTypeDecision,
		Timestamp: time.Now(),
		Data:      decision,
	}
}

// BroadcastPosition broadcasts position update
func (h *Hub) BroadcastPosition(position *bot.Position) {
	h.broadcast <- Event{
		Type:      EventTypePosition,
		Timestamp: time.Now(),
		Data:      position,
	}
}

// BroadcastPnL broadcasts PnL update
func (h *Hub) BroadcastPnL(pnl PnLData) {
	h.broadcast <- Event{
		Type:      EventTypePnL,
		Timestamp: time.Now(),
		Data:      pnl,
	}
}

// BroadcastStatus broadcasts bot status
func (h *Hub) BroadcastStatus(status string) {
	h.broadcast <- Event{
		Type:      EventTypeStatus,
		Timestamp: time.Now(),
		Data:      map[string]string{"status": status},
	}
}

// Client represents a WebSocket client
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan Event
}

// NewClient creates a new WebSocket client
func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		hub:  hub,
		conn: conn,
		send: make(chan Event, 256),
	}
}

// ReadPump handles incoming messages (mostly pings)
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// WritePump sends messages to the WebSocket
func (c *Client) WritePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case event, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			data, err := json.Marshal(event)
			if err != nil {
				log.Printf("Error marshaling event: %v", err)
				continue
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
