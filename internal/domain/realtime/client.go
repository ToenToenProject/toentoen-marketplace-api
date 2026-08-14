package realtime

import (
	"log"
	"sync"
	"time"

	"github.com/gofiber/websocket/v2"
)

const (
	// Waktu tunggu untuk menulis pesan ke client
	writeWait = 10 * time.Second
	// Waktu tunggu untuk menerima pong dari client
	pongWait = 60 * time.Second
	// Interval pengiriman ping ke client (harus lebih kecil dari pongWait)
	pingPeriod = (pongWait * 9) / 10
	// Ukuran maksimum buffer pesan masuk (dinaikkan untuk payload lokasi)
	maxMessageSize = 2048
)

// LocationState menyimpan state throttle lokasi per koneksi client
type LocationState struct {
	LastLat         float64
	LastLng         float64
	LastSentAt      time.Time
	IsStationary    bool
	StationarySince time.Time
	UpdateCount     int64
	LastUpdateAt    time.Time
}

// SubscriptionArea menyimpan area yang di-subscribe oleh customer
type SubscriptionArea struct {
	Lat      float64
	Lng      float64
	RadiusKm float64
	Active   bool
}

// Client merepresentasikan satu koneksi WebSocket
type Client struct {
	Hub       *Hub
	Conn      *websocket.Conn
	Send      chan []byte
	AccountID string
	Role      string // "customer" atau "owner"

	// Live Location state
	locState *LocationState
	subArea  *SubscriptionArea
	locMu    sync.Mutex
}

// GetLocationState mengembalikan location state untuk client ini (lazy init)
func (c *Client) GetLocationState() *LocationState {
	c.locMu.Lock()
	defer c.locMu.Unlock()
	if c.locState == nil {
		c.locState = &LocationState{}
	}
	return c.locState
}

// SetSubscriptionArea mengatur area subscription untuk live location
func (c *Client) SetSubscriptionArea(lat, lng, radiusKm float64) {
	c.locMu.Lock()
	defer c.locMu.Unlock()
	c.subArea = &SubscriptionArea{
		Lat:      lat,
		Lng:      lng,
		RadiusKm: radiusKm,
		Active:   true,
	}
}

// GetSubscriptionArea mengembalikan area subscription saat ini
func (c *Client) GetSubscriptionArea() *SubscriptionArea {
	c.locMu.Lock()
	defer c.locMu.Unlock()
	return c.subArea
}

// ClearSubscriptionArea menghapus area subscription
func (c *Client) ClearSubscriptionArea() {
	c.locMu.Lock()
	defer c.locMu.Unlock()
	c.subArea = nil
}

// ReadPump membaca pesan dari WebSocket client dan meroute-kannya
// Dijalankan di goroutine terpisah per client
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("⚠️ WS read error [%s]: %v", c.AccountID, err)
			}
			break
		}

		// Route pesan ke handler yang sesuai
		if len(msg) > 0 {
			c.Hub.HandleIncomingMessage(c, msg)
		}
	}
}

// WritePump menulis pesan dari server ke WebSocket client
// Dijalankan di goroutine terpisah per client
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub menutup channel
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

